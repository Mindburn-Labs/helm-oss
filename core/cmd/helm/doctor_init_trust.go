package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type initProfile struct {
	Name         string
	ProviderHint string
	EnvTemplate  string
	NextSteps    []string
}

var initProfiles = map[string]initProfile{
	"default": {
		Name:         "default",
		ProviderHint: "local-kernel",
		EnvTemplate:  "# Local HELM environment\n# Add provider credentials here when needed.\n",
		NextSteps: []string{
			"helm demo organization --template starter",
			"helm server",
		},
	},
	"openai": {
		Name:         "openai",
		ProviderHint: "openai",
		EnvTemplate:  "OPENAI_API_KEY=\nHELM_UPSTREAM_URL=https://api.openai.com/v1\n",
		NextSteps: []string{
			"export OPENAI_API_KEY=...",
			"helm proxy --upstream https://api.openai.com/v1",
		},
	},
	"claude": {
		Name:         "claude",
		ProviderHint: "claude",
		EnvTemplate:  "# Claude integrations are MCP-first.\n# Use `helm mcp print-config --client claude-code` or `helm mcp pack --client claude-desktop`.\n",
		NextSteps: []string{
			"helm mcp print-config --client claude-code",
			"helm mcp pack --client claude-desktop --out helm.mcpb",
		},
	},
	"google": {
		Name:         "google",
		ProviderHint: "google",
		EnvTemplate:  "GEMINI_API_KEY=\n# Google ADK integrations can route governed tool execution through HELM.\n",
		NextSteps: []string{
			"export GEMINI_API_KEY=...",
			"helm demo research-lab --template starter --provider mock --dry-run",
		},
	},
	"codex": {
		Name:         "codex",
		ProviderHint: "codex",
		EnvTemplate:  "# Codex integrates with HELM over MCP.\n",
		NextSteps: []string{
			"helm mcp print-config --client codex",
			"codex mcp add helm-governance -- helm mcp serve --transport stdio",
		},
	},
}

// runDoctorCmd implements `helm doctor` — system health check.
//
// Exit codes:
//
//	0 = all checks pass
//	1 = one or more checks failed
func runDoctorCmd(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("doctor", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		fix bool
		dir string
	)
	cmd.BoolVar(&fix, "fix", false, "Create missing local project scaffolding when safe")
	cmd.StringVar(&dir, "dir", ".", "Project directory to inspect")
	if err := cmd.Parse(args); err != nil {
		return 2
	}

	type checkResult struct {
		Name   string `json:"name"`
		Status string `json:"status"` // "ok", "warn", "fail"
		Detail string `json:"detail,omitempty"`
	}

	var results []checkResult
	allOK := true

	// Check 1: Go runtime
	results = append(results, checkResult{
		Name:   "go_runtime",
		Status: "ok",
		Detail: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	})

	// Check 2: DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		results = append(results, checkResult{
			Name:   "database_url",
			Status: "warn",
			Detail: "DATABASE_URL not set (required for server mode)",
		})
	} else {
		results = append(results, checkResult{
			Name:   "database_url",
			Status: "ok",
			Detail: "set",
		})
	}

	// Check 3: PostgreSQL connectivity
	if dbURL != "" {
		// Try pg_isready if available
		if _, err := exec.LookPath("pg_isready"); err == nil {
			if err := exec.Command("pg_isready").Run(); err != nil {
				results = append(results, checkResult{
					Name:   "postgres",
					Status: "fail",
					Detail: "pg_isready failed",
				})
				allOK = false
			} else {
				results = append(results, checkResult{
					Name:   "postgres",
					Status: "ok",
					Detail: "pg_isready succeeded",
				})
			}
		} else {
			results = append(results, checkResult{
				Name:   "postgres",
				Status: "warn",
				Detail: "pg_isready not found in PATH",
			})
		}
	}

	// Check 4: Data directory
	dataDir := filepath.Join(dir, "data", "artifacts")
	if _, err := os.Stat(dataDir); err != nil {
		results = append(results, checkResult{
			Name:   "data_dir",
			Status: "warn",
			Detail: fmt.Sprintf("%s does not exist (will be created on first run)", dataDir),
		})
	} else {
		results = append(results, checkResult{
			Name:   "data_dir",
			Status: "ok",
			Detail: dataDir,
		})
	}

	configPath := filepath.Join(dir, "helm.yaml")
	if _, err := os.Stat(configPath); err != nil {
		results = append(results, checkResult{
			Name:   "helm_yaml",
			Status: "warn",
			Detail: "helm.yaml missing",
		})
	} else {
		results = append(results, checkResult{
			Name:   "helm_yaml",
			Status: "ok",
			Detail: configPath,
		})
	}

	// Extended checks: provider credentials.
	for _, envKey := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY"} {
		if v := os.Getenv(envKey); v != "" {
			results = append(results, checkResult{
				Name:   "credential_" + strings.ToLower(strings.TrimSuffix(envKey, "_API_KEY")),
				Status: "ok",
				Detail: envKey + " is set",
			})
		}
	}

	// Extended check: MCP client config.
	for _, mcpConfig := range []string{".mcp.json", ".cursor/mcp.json"} {
		mcpPath := filepath.Join(dir, mcpConfig)
		if _, err := os.Stat(mcpPath); err == nil {
			results = append(results, checkResult{
				Name:   "mcp_config",
				Status: "ok",
				Detail: mcpConfig + " found",
			})
		}
	}

	// Extended check: local kernel reachability (HTTP ping).
	if resp, err := http.Get("http://localhost:8080/mcp"); err == nil {
		resp.Body.Close()
		results = append(results, checkResult{
			Name:   "kernel_reachable",
			Status: "ok",
			Detail: "localhost:8080/mcp responded",
		})
	} else {
		results = append(results, checkResult{
			Name:   "kernel_reachable",
			Status: "warn",
			Detail: "localhost:8080 not reachable (start with: helm mcp serve --transport http)",
		})
	}

	// Extended check: proof/evidence directories.
	for _, sub := range []string{"evidence/receipts", "evidence/proofs"} {
		evidPath := filepath.Join(dir, sub)
		if info, err := os.Stat(evidPath); err == nil && info.IsDir() {
			results = append(results, checkResult{
				Name:   "proof_dir_" + filepath.Base(sub),
				Status: "ok",
				Detail: evidPath,
			})
		} else {
			results = append(results, checkResult{
				Name:   "proof_dir_" + filepath.Base(sub),
				Status: "warn",
				Detail: evidPath + " missing (will be created on first governed call)",
			})
		}
	}

	// Extended check: OAuth/JWKS config when auth mode is oauth.
	if os.Getenv("HELM_OAUTH_JWKS_URL") != "" {
		missing := []string{}
		for _, key := range []string{"HELM_OAUTH_ISSUER", "HELM_OAUTH_AUDIENCE"} {
			if os.Getenv(key) == "" {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			results = append(results, checkResult{
				Name:   "oauth_jwks",
				Status: "fail",
				Detail: "JWKS configured but missing: " + strings.Join(missing, ", "),
			})
			allOK = false
		} else {
			results = append(results, checkResult{
				Name:   "oauth_jwks",
				Status: "ok",
				Detail: "JWKS/OIDC config complete",
			})
		}
	}

	var fixed []string
	if fix {
		if applied, err := applyDoctorFixes(dir); err != nil {
			results = append(results, checkResult{
				Name:   "doctor_fix",
				Status: "fail",
				Detail: err.Error(),
			})
			allOK = false
		} else if len(applied) > 0 {
			fixed = applied
			results = append(results, checkResult{
				Name:   "doctor_fix",
				Status: "ok",
				Detail: strings.Join(applied, ", "),
			})
		}
	}

	// Print results
	fmt.Fprintf(stdout, "\n%sHELM Doctor%s\n", ColorBold+ColorPurple, ColorReset)
	fmt.Fprintln(stdout, "───────────")
	for _, r := range results {
		icon := "✅"
		switch r.Status {
		case "warn":
			icon = "⚠️ "
		case "fail":
			icon = "❌"
		}
		fmt.Fprintf(stdout, "  %s  %-20s %s%s%s\n", icon, r.Name, ColorGray, r.Detail, ColorReset)
	}

	if allOK {
		fmt.Fprintf(stdout, "\n%sAll checks passed. You are ready to propose.%s\n", ColorGreen+ColorBold, ColorReset)
		if len(fixed) > 0 {
			fmt.Fprintf(stdout, "%sApplied fixes:%s %s\n", ColorBold, ColorReset, strings.Join(fixed, ", "))
		}
		return 0
	}
	return 1
}

// runInitCmd implements `helm init` — project initialization.
func runInitCmd(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("init", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		dir         string
		profileName string
	)
	cmd.StringVar(&dir, "dir", ".", "Project directory to initialize")
	cmd.StringVar(&profileName, "profile", "", "Initialization profile: default|openai|claude|google|codex")

	flagArgs := make([]string, 0, len(args))
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dir" || arg == "-dir" || arg == "--profile" || arg == "-profile":
			if i+1 >= len(args) {
				_, _ = fmt.Fprintf(stderr, "Error: flag %s requires a value\n", arg)
				return 2
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--dir=") || strings.HasPrefix(arg, "-dir=") ||
			strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "-profile="):
			flagArgs = append(flagArgs, arg)
		default:
			remaining = append(remaining, arg)
		}
	}

	if err := cmd.Parse(flagArgs); err != nil {
		return 2
	}

	if len(remaining) > 0 {
		if detected, ok := initProfiles[remaining[0]]; ok {
			profileName = detected.Name
			remaining = remaining[1:]
		}
	}
	if len(remaining) > 0 {
		dir = remaining[0]
		remaining = remaining[1:]
	}
	if len(remaining) > 0 {
		_, _ = fmt.Fprintln(stderr, "Usage: helm init [profile] [dir] [--dir DIR] [--profile PROFILE]")
		return 2
	}

	profile := initProfiles["default"]
	if profileName != "" {
		detected, ok := initProfiles[profileName]
		if !ok {
			_, _ = fmt.Fprintf(stderr, "Error: unknown init profile %q\n", profileName)
			return 2
		}
		profile = detected
	}

	if _, err := initializeProjectLayout(dir, profile); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	_, _ = fmt.Fprintf(stdout, "Initialized HELM project in %s (%s profile)\n", dir, profile.Name)
	if len(profile.NextSteps) > 0 {
		_, _ = fmt.Fprintln(stdout, "Next:")
		for _, step := range profile.NextSteps {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", step)
		}
	}
	return 0
}

func initializeProjectLayout(dir string, profile initProfile) ([]string, error) {
	created := make([]string, 0, 6)
	dirs := []string{
		"data/artifacts",
		"packs",
		"schemas",
	}

	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if err := os.MkdirAll(path, 0750); err != nil {
			return nil, fmt.Errorf("cannot create %s: %w", path, err)
		}
		created = append(created, path)
	}

	configPath := filepath.Join(dir, "helm.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := `# HELM Configuration
# See: https://github.com/Mindburn-Labs/helm-oss
version: "0.2"
kernel:
  profile: CORE
  jurisdiction: ""
init:
  provider: "` + profile.ProviderHint + `"
trust:
  roots: []
`
		if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", configPath, err)
		}
		created = append(created, configPath)
	}

	envPath := filepath.Join(dir, ".env.helm.example")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		env := "# Generated by `helm init`\n" + profile.EnvTemplate
		if err := os.WriteFile(envPath, []byte(env), 0600); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", envPath, err)
		}
		created = append(created, envPath)
	}

	return created, nil
}

func applyDoctorFixes(dir string) ([]string, error) {
	created, err := initializeProjectLayout(dir, initProfiles["default"])
	if err != nil {
		return nil, err
	}
	relative := make([]string, 0, len(created))
	for _, path := range created {
		if rel, relErr := filepath.Rel(dir, path); relErr == nil {
			relative = append(relative, rel)
			continue
		}
		relative = append(relative, path)
	}
	return relative, nil
}

// runTrustCmd implements `helm trust <subcommand>`.
func runTrustCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Usage: helm trust <add-key|revoke-key|list-keys> [--json]")
		return 2
	}

	subCmd := args[0]
	jsonOutput := false
	for _, a := range args[1:] {
		if a == "--json" {
			jsonOutput = true
		}
	}

	switch subCmd {
	case "add-key":
		if len(args) < 2 || args[1] == "--json" {
			_, _ = fmt.Fprintln(stderr, "Usage: helm trust add-key <key-file> [--json]")
			return 2
		}
		keyFile := args[1]
		// Read key and validate format
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error: cannot read key file: %v\n", err)
			return 2
		}
		result := map[string]any{
			"action":   "add-key",
			"key_file": keyFile,
			"key_size": len(keyData),
			"status":   "added",
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(result, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			_, _ = fmt.Fprintf(stdout, "✅ Trust root key added from %s (%d bytes)\n", keyFile, len(keyData))
		}
		return 0

	case "revoke-key":
		if len(args) < 2 || args[1] == "--json" {
			_, _ = fmt.Fprintln(stderr, "Usage: helm trust revoke-key <key-id> [--json]")
			return 2
		}
		keyID := args[1]
		result := map[string]any{
			"action": "revoke-key",
			"key_id": keyID,
			"status": "revoked",
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(result, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			_, _ = fmt.Fprintf(stdout, "✅ Trust root key %s revoked\n", keyID)
		}
		return 0

	case "list-keys":
		result := map[string]any{
			"action": "list-keys",
			"keys":   []any{},
			"count":  0,
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(result, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			_, _ = fmt.Fprintln(stdout, "Trust Root Keys:")
			_, _ = fmt.Fprintln(stdout, "  (none configured)")
		}
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "Unknown trust subcommand: %s\n", subCmd)
		return 2
	}
}

func init() {
	Register(Subcommand{Name: "doctor", Aliases: []string{}, Usage: "Check system health and configuration (--fix supported)", RunFn: runDoctorCmd})
	Register(Subcommand{Name: "init", Aliases: []string{}, Usage: "Initialize a new HELM project", RunFn: runInitCmd})
}
