package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"
	mcppkg "github.com/Mindburn-Labs/helm-oss/core/pkg/mcp"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/prg"
)

const maxLocalMCPFileSize = 1 << 20

type mcpRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpRPCResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      any          `json:"id,omitempty"`
	Result  any          `json:"result,omitempty"`
	Error   *mcpRPCError `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newLocalMCPRuntime() (*mcppkg.ToolCatalog, mcppkg.ToolExecutor, error) {
	catalog := mcppkg.NewInMemoryCatalog()
	catalog.RegisterCommonTools()

	signer, err := loadOrGenerateSigner()
	if err != nil {
		return nil, nil, err
	}

	rules := prg.NewGraph()
	for _, toolName := range []string{"file_read", "file_write"} {
		if addErr := rules.AddRule(toolName, prg.RequirementSet{
			ID:    "mcp-" + toolName,
			Logic: prg.AND,
		}); addErr != nil {
			return nil, nil, fmt.Errorf("register policy for %s: %w", toolName, addErr)
		}
	}

	guard := guardian.NewGuardian(signer, rules, nil)
	firewall := mcppkg.NewGovernanceFirewall(guard, catalog)

	return catalog, mcppkg.ToolExecutor(firewall.WrapToolHandler(runLocalMCPTool)), nil
}

func newLocalMCPGateway() (*mcppkg.Gateway, error) {
	catalog, executor, err := newLocalMCPRuntime()
	if err != nil {
		return nil, err
	}

	return mcppkg.NewGateway(catalog, mcppkg.GatewayConfig{}, mcppkg.WithExecutor(executor)), nil
}

func runLocalMCPTool(ctx context.Context, req mcppkg.ToolExecutionRequest) (mcppkg.ToolExecutionResponse, error) {
	switch req.ToolName {
	case "file_read":
		path, _ := req.Arguments["path"].(string)
		resolvedPath, err := resolveLocalMCPPath(path)
		if err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: err.Error(),
				IsError: true,
			}, nil
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: fmt.Sprintf("read %s: %v", resolvedPath, err),
				IsError: true,
			}, nil
		}
		if info.Size() > maxLocalMCPFileSize {
			return mcppkg.ToolExecutionResponse{
				Content: fmt.Sprintf("read %s: file exceeds %d bytes", resolvedPath, maxLocalMCPFileSize),
				IsError: true,
			}, nil
		}

		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: fmt.Sprintf("read %s: %v", resolvedPath, err),
				IsError: true,
			}, nil
		}

		return mcppkg.ToolExecutionResponse{
			Content: string(data),
		}, nil

	case "file_write":
		path, _ := req.Arguments["path"].(string)
		content, _ := req.Arguments["content"].(string)

		resolvedPath, err := resolveLocalMCPPath(path)
		if err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: err.Error(),
				IsError: true,
			}, nil
		}

		if err := os.MkdirAll(filepath.Dir(resolvedPath), 0750); err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: fmt.Sprintf("prepare %s: %v", resolvedPath, err),
				IsError: true,
			}, nil
		}
		if err := os.WriteFile(resolvedPath, []byte(content), 0600); err != nil {
			return mcppkg.ToolExecutionResponse{
				Content: fmt.Sprintf("write %s: %v", resolvedPath, err),
				IsError: true,
			}, nil
		}

		return mcppkg.ToolExecutionResponse{
			Content: fmt.Sprintf("wrote %d bytes to %s", len(content), resolvedPath),
		}, nil
	default:
		return mcppkg.ToolExecutionResponse{
			Content: fmt.Sprintf("tool %q not supported", req.ToolName),
			IsError: true,
		}, nil
	}
}

func resolveLocalMCPPath(rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return "", fmt.Errorf("path is required")
	}

	cleaned := filepath.Clean(rawPath)
	if filepath.IsAbs(cleaned) {
		return cleaned, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return filepath.Join(wd, cleaned), nil
}

func serveLocalMCPStdio(stdin io.Reader, stdout io.Writer) error {
	catalog, executor, err := newLocalMCPRuntime()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(stdin)
	for {
		req, err := readMCPRequest(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		resp, err := handleMCPRPCRequest(req, catalog, executor)
		if err != nil {
			return err
		}
		if resp == nil {
			continue
		}
		if err := writeMCPResponse(stdout, resp); err != nil {
			return err
		}
	}
}

func readMCPRequest(reader *bufio.Reader) (*mcpRPCRequest, error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "{") {
			var req mcpRPCRequest
			if err := json.Unmarshal([]byte(trimmed), &req); err != nil {
				return nil, fmt.Errorf("decode stdio request: %w", err)
			}
			return &req, nil
		}

		headers := map[string]string{}
		for {
			if trimmed == "" {
				break
			}
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid MCP header %q", trimmed)
			}
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])

			nextLine, readErr := reader.ReadString('\n')
			if readErr != nil {
				return nil, readErr
			}
			trimmed = strings.TrimSpace(nextLine)
			if trimmed == "" {
				break
			}
		}

		contentLength, err := strconv.Atoi(headers["content-length"])
		if err != nil || contentLength <= 0 {
			return nil, fmt.Errorf("missing or invalid Content-Length header")
		}

		payload := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, fmt.Errorf("read MCP payload: %w", err)
		}

		var req mcpRPCRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, fmt.Errorf("decode MCP payload: %w", err)
		}
		return &req, nil
	}
}

func writeMCPResponse(stdout io.Writer, resp *mcpRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode MCP response: %w", err)
	}

	if _, err := fmt.Fprintf(stdout, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = stdout.Write(data)
	return err
}

func handleMCPRPCRequest(req *mcpRPCRequest, catalog *mcppkg.ToolCatalog, executor mcppkg.ToolExecutor) (*mcpRPCResponse, error) {
	response := &mcpRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		response.Result = map[string]any{
			"protocolVersion": "2025-03-26",
			"serverInfo": map[string]any{
				"name":    "helm-governance",
				"version": displayVersion(),
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		}
		return response, nil

	case "notifications/initialized":
		return nil, nil

	case "ping":
		response.Result = map[string]any{}
		return response, nil

	case "tools/list":
		tools, err := catalog.Search(context.Background(), "")
		if err != nil {
			response.Error = &mcpRPCError{Code: -32603, Message: err.Error()}
			return response, nil
		}

		payload := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			payload = append(payload, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.Schema,
			})
		}
		response.Result = map[string]any{"tools": payload}
		return response, nil

	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			response.Error = &mcpRPCError{Code: -32602, Message: "invalid tools/call params"}
			return response, nil
		}

		if _, ok := catalog.Lookup(params.Name); !ok {
			response.Error = &mcpRPCError{Code: -32602, Message: fmt.Sprintf("tool %q not found", params.Name)}
			return response, nil
		}

		// Extract delegation context from headers for scope enforcement.
		execReq := mcppkg.ToolExecutionRequest{
			ToolName:  params.Name,
			Arguments: params.Arguments,
			SessionID: "mcp-stdio",
		}
		if delegationID := r.Header.Get("X-HELM-Delegation-Session-ID"); delegationID != "" {
			execReq.DelegationSessionID = delegationID
			execReq.DelegationVerifier = r.Header.Get("X-HELM-Delegation-Verifier")
			if allowedCSV := r.Header.Get("X-HELM-Delegation-Allowed-Tools"); allowedCSV != "" {
				for _, t := range strings.Split(allowedCSV, ",") {
					if trimmed := strings.TrimSpace(t); trimmed != "" {
						execReq.DelegationAllowedTools = append(execReq.DelegationAllowedTools, trimmed)
					}
				}
			}
		}

		execResp, err := executor(context.Background(), execReq)
		if err != nil {
			response.Error = &mcpRPCError{Code: -32603, Message: err.Error()}
			return response, nil
		}

		result := map[string]any{
			"content": []map[string]string{
				{
					"type": "text",
					"text": execResp.Content,
				},
			},
			"isError": execResp.IsError,
		}
		if execResp.ReceiptID != "" {
			result["receipt_id"] = execResp.ReceiptID
		}

		response.Result = result
		return response, nil

	default:
		response.Error = &mcpRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("method %q not found", req.Method),
		}
		return response, nil
	}
}

func newLocalMCPHTTPServer(port int, authMode string) (*http.Server, error) {
	gateway, err := newLocalMCPGateway()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	gateway.RegisterRoutes(mux)
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"transport": "http",
		})
	}
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)

	handler, err := wrapMCPAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}), authMode)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, nil
}

func wrapMCPAuth(next http.Handler, authMode string) (http.Handler, error) {
	switch authMode {
	case "none":
		return next, nil
	case "static-header":
		expectedKey := os.Getenv("HELM_API_KEY")
		if expectedKey == "" {
			return nil, fmt.Errorf("HELM_API_KEY must be set when --auth static-header is used")
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-HELM-API-Key")
			if provided == "" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}
			if provided != expectedKey {
				http.Error(w, "missing or invalid MCP API key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}), nil
	case "oauth":
		return nil, fmt.Errorf("oauth auth mode is not implemented in the OSS runtime")
	default:
		return nil, fmt.Errorf("unknown auth mode %q", authMode)
	}
}
