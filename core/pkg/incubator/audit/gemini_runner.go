package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ── Gemini Mission Runner ───────────────────────────────────────────────────
//
// HTTP-only client for Google Gemini API. No SDK dependency.
// Renders mission templates → sends to Gemini → parses structured findings.
//
// Usage:
//
//   runner := audit.NewGeminiRunner(audit.GeminiConfig{
//       APIKey: os.Getenv("GEMINI_API_KEY"),
//   })
//   results, err := runner.RunAllMissions(ctx, registry, vars)

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultModel  = "gemini-2.5-flash"
)

// GeminiConfig configures the Gemini API client.
type GeminiConfig struct {
	APIKey      string        `json:"api_key"`
	OAuthToken  string        `json:"-"`       // Bearer token (from gcloud ADC)
	Project     string        `json:"project"` // GCP project for Vertex AI
	Region      string        `json:"region"`  // default: us-central1
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
}

// DefaultGeminiConfig returns production-safe defaults.
func DefaultGeminiConfig(apiKey string) GeminiConfig {
	return GeminiConfig{
		APIKey:      apiKey,
		Region:      "us-central1",
		Model:       defaultModel,
		MaxTokens:   8192,
		Temperature: 0.1, // Near-deterministic
		Timeout:     120 * time.Second,
		MaxRetries:  3,
	}
}

// GeminiRunner executes AI audit missions via the Gemini API.
type GeminiRunner struct {
	config GeminiConfig
	client *http.Client
	parser *AIParser
}

// LoadOAuthToken tries multiple credential sources for Gemini API access:
//  1. gcloud application-default credentials (has generativelanguage scope)
//  2. gcloud auth print-access-token
//  3. Gemini CLI OAuth token (may lack scopes)
func LoadOAuthToken() (string, error) {
	// Try gcloud ADC first (has the right scopes)
	if token, err := loadGcloudADCToken(); err == nil && token != "" {
		return token, nil
	}

	// Try gcloud auth print-access-token
	if out, err := execCommand("gcloud", "auth", "print-access-token"); err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" && !strings.Contains(token, "ERROR") {
			return token, nil
		}
	}

	// Fallback: Gemini CLI OAuth (may have insufficient scopes)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	credsPath := filepath.Join(home, ".gemini", "oauth_creds.json")
	data, err := os.ReadFile(credsPath)
	if err != nil {
		return "", fmt.Errorf("no oauth credentials found")
	}
	var creds struct {
		AccessToken string `json:"access_token"`
		ExpiryDate  int64  `json:"expiry_date"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("parse oauth: %w", err)
	}
	expiry := creds.ExpiryDate
	if expiry > 1e12 {
		expiry = expiry / 1000
	}
	if time.Unix(expiry, 0).Before(time.Now()) {
		return "", fmt.Errorf("oauth token expired")
	}
	return creds.AccessToken, nil
}

func loadGcloudADCToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	data, err := os.ReadFile(adcPath)
	if err != nil {
		return "", err
	}
	var adc struct {
		Type         string `json:"type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &adc); err != nil {
		return "", err
	}
	if adc.RefreshToken == "" || adc.ClientID == "" {
		return "", fmt.Errorf("incomplete ADC")
	}
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"client_id":     {adc.ClientID},
		"client_secret": {adc.ClientSecret},
		"refresh_token": {adc.RefreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// detectGCPProject tries to find a GCP project from gcloud config.
func detectGCPProject() string {
	if out, err := execCommand("gcloud", "config", "get-value", "project"); err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" && p != "(unset)" {
			return p
		}
	}
	// Fallback: check for Gemini API project from gcloud projects list
	if out, err := execCommand("gcloud", "projects", "list", "--filter=name:Gemini", "--format=value(projectId)"); err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" {
			return strings.Split(p, "\n")[0]
		}
	}
	return ""
}

func execCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// NewGeminiRunner creates a runner with the given config.
func NewGeminiRunner(config GeminiConfig, parser *AIParser) *GeminiRunner {
	if config.Model == "" {
		config.Model = defaultModel
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 8192
	}
	if config.Temperature == 0 {
		config.Temperature = 0.1
	}
	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &GeminiRunner{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		parser: parser,
	}
}

// geminiRequest is the Gemini API request body.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
}

// geminiResponse is the Gemini API response.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// RunMission executes a single mission against the Gemini API.
func (r *GeminiRunner) RunMission(ctx context.Context, mission *Mission, vars map[string]string) (*MissionResult, error) {
	if r.config.APIKey == "" && r.config.OAuthToken == "" && !HasGeminiCLI() {
		return nil, fmt.Errorf("gemini: no API key, OAuth token, or gemini CLI found")
	}

	// Render prompt from mission template
	registry := NewMissionRegistry()
	registry.Register(mission)
	prompt, err := registry.RenderPrompt(mission.ID, vars)
	if err != nil {
		return nil, fmt.Errorf("gemini: render prompt for %s: %w", mission.ID, err)
	}

	start := time.Now()

	// Call Gemini API with retries
	var rawOutput string
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(15*(attempt+1)) * time.Second // 15s, 30s, 45s for free-tier
			slog.Info("gemini: retrying",
				"mission", mission.ID,
				"attempt", attempt,
				"backoff", backoff,
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		rawOutput, lastErr = r.CallAPI(ctx, prompt)
		if lastErr == nil {
			break
		}

		// Don't retry non-retryable errors (403, 401, 404)
		errStr := lastErr.Error()
		if strings.Contains(errStr, "(403)") || strings.Contains(errStr, "(401)") || strings.Contains(errStr, "(404)") {
			slog.Warn("gemini: non-retryable error",
				"mission", mission.ID,
				"error", lastErr,
			)
			break
		}

		slog.Warn("gemini: API call failed",
			"mission", mission.ID,
			"attempt", attempt,
			"error", lastErr,
		)
	}

	elapsed := time.Since(start)

	if lastErr != nil {
		return &MissionResult{
			MissionID: mission.ID,
			Name:      mission.Name,
			Category:  mission.Category,
			Status:    "failed",
			Output:    fmt.Sprintf("ERROR: %v", lastErr),
		}, lastErr
	}

	// Parse findings from output
	findings := r.parser.ParseMissionOutput(mission.ID, mission.Category, rawOutput)

	slog.Info("gemini: mission complete",
		"mission", mission.ID,
		"findings", len(findings),
		"elapsed", elapsed.Round(time.Millisecond),
		"tokens_approx", len(rawOutput)/4,
	)

	return &MissionResult{
		MissionID:    mission.ID,
		Name:         mission.Name,
		Category:     mission.Category,
		Status:       "completed",
		Output:       rawOutput,
		FindingCount: len(findings),
	}, nil
}

// RunAllMissions executes all active missions sequentially.
func (r *GeminiRunner) RunAllMissions(ctx context.Context, registry *MissionRegistry, vars map[string]string) ([]MissionResult, error) {
	missions := registry.Active()
	if len(missions) == 0 {
		return nil, nil
	}

	slog.Info("gemini: starting L2 audit",
		"missions", len(missions),
		"model", r.config.Model,
	)

	var results []MissionResult
	for _, m := range missions {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := r.RunMission(ctx, m, vars)
		if result != nil {
			results = append(results, *result)

			// Record metrics back to registry
			findings := r.parser.ParseMissionOutput(m.ID, m.Category, result.Output)
			registry.RecordFindingCount(m.ID, len(findings))
		}
		if err != nil {
			slog.Warn("gemini: mission failed",
				"mission", m.ID,
				"error", err,
			)
		}

		// Inter-mission delay to respect rate limits
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return results, nil
}

// HasGeminiCLI checks if the gemini CLI is installed and available.
func HasGeminiCLI() bool {
	if os.Getenv("HELM_DISABLE_GEMINI_CLI") == "1" {
		return false
	}
	_, err := exec.LookPath("gemini")
	return err == nil
}

// CallAPI sends a prompt to the Gemini API and returns the response text.
func (r *GeminiRunner) CallAPI(ctx context.Context, prompt string) (string, error) {
	// Fallback to gemini CLI if no direct API credentials are set.
	// Uses the same auth as ../mindburn/qa/lib/gemini-api.sh (Google Ultra sub).
	if r.config.APIKey == "" && r.config.Project == "" {
		// Create temp files to capture output (avoids Go pipe deadlocks on stdout/stderr)
		outFile, err := os.CreateTemp("", "helm-gemini-*.txt")
		if err != nil {
			return "", fmt.Errorf("gemini: create temp file: %w", err)
		}
		outPath := outFile.Name()
		outFile.Close()
		defer os.Remove(outPath)

		errPath := outPath + ".stderr"
		defer os.Remove(errPath)

		// Write prompt to temp file to avoid shell quoting issues with large prompts
		promptFile, err := os.CreateTemp("", "helm-gemini-prompt-*.txt")
		if err != nil {
			return "", fmt.Errorf("gemini: create prompt file: %w", err)
		}
		promptPath := promptFile.Name()
		if _, err := promptFile.WriteString(prompt); err != nil {
			promptFile.Close()
			return "", fmt.Errorf("gemini: write prompt: %w", err)
		}
		promptFile.Close()
		defer os.Remove(promptPath)

		// Use -p flag with the prompt from a file.
		// Direct exec (not shell) avoids quoting issues entirely.
		cmd := exec.CommandContext(ctx, "gemini", "-p", prompt, "--model", r.config.Model)
		// CRITICAL: Isolate into its own process group. Without this, Node.js
		// MCP server children leak into Go's process group and block the next
		// invocation from starting. Confirmed via /tmp/gtest_main.go repro.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// Redirect stdout/stderr to temp files (avoids Go pipe buffer deadlocks)
		fOut, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return "", fmt.Errorf("gemini: open stdout file: %w", err)
		}
		fErr, err := os.OpenFile(errPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			fOut.Close()
			return "", fmt.Errorf("gemini: open stderr file: %w", err)
		}
		cmd.Stdout = fOut
		cmd.Stderr = fErr

		err = cmd.Run()
		fOut.Close()
		fErr.Close()

		if err != nil {
			stderrData, _ := os.ReadFile(errPath)
			return "", fmt.Errorf("gemini cli failed: %w\nstderr: %s", err, string(stderrData))
		}

		outputData, err := os.ReadFile(outPath)
		if err != nil {
			return "", fmt.Errorf("gemini: read output: %w", err)
		}

		result := strings.TrimSpace(string(outputData))
		if result == "" {
			return "", fmt.Errorf("gemini cli returned empty response")
		}

		return result, nil
	}

	// Build URL — API key uses generativelanguage API, OAuth uses Vertex AI
	var url string
	if r.config.APIKey != "" {
		// Consumer API with API key
		url = fmt.Sprintf("%s/%s:generateContent?key=%s",
			geminiBaseURL, r.config.Model, r.config.APIKey)
	} else if r.config.Project != "" {
		// Vertex AI endpoint (accepts cloud-platform scope)
		region := r.config.Region
		if region == "" {
			region = "us-central1"
		}
		url = fmt.Sprintf(
			"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
			region, r.config.Project, region, r.config.Model,
		)
	} else {
		// Fallback to consumer API with bearer token (may fail on scopes)
		url = fmt.Sprintf("%s/%s:generateContent",
			geminiBaseURL, r.config.Model)
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			MaxOutputTokens: r.config.MaxTokens,
			Temperature:     r.config.Temperature,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("gemini: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.config.OAuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.config.OAuthToken)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode == 429 {
		return "", fmt.Errorf("gemini: rate limited (429)")
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("gemini: server error (%d): %s", resp.StatusCode, string(respBytes))
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini: API error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBytes, &gemResp); err != nil {
		return "", fmt.Errorf("gemini: unmarshal response: %w", err)
	}

	if gemResp.Error != nil {
		return "", fmt.Errorf("gemini: API error %d: %s", gemResp.Error.Code, gemResp.Error.Message)
	}

	if len(gemResp.Candidates) == 0 {
		return "", fmt.Errorf("gemini: no candidates in response")
	}

	candidate := gemResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty content")
	}

	return candidate.Content.Parts[0].Text, nil
}
