package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

// StdioMCPClient talks to an MCP server via stdio.
// MVP: Simplified JSON-RPC logic.
type StdioMCPClient struct {
	Command string
	Args    []string
}

func NewStdioMCPClient(cmd string, args ...string) *StdioMCPClient {
	return &StdioMCPClient{Command: cmd, Args: args}
}

type mcpRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	ID      int            `json:"id"`
}

func (s *StdioMCPClient) Call(tool string, params map[string]any) error {
	// 1. Prepare Request
	req := mcpRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": tool, "arguments": params},
		ID:      1,
	}
	reqBytes, _ := json.Marshal(req) //nolint:errcheck // JSON marshal error ignored for simplicity

	// 2. Exec Process (One-shot for MVP)
	// In production, this would be a long-running process with a proper transport.
	//nolint:gosec // G204: Command args are controlled by internal caller
	cmd := exec.CommandContext(context.Background(), s.Command, s.Args...)

	// stdin
	stdin, _ := cmd.StdinPipe() //nolint:errcheck // Pipe error ignored for demo
	go func() {
		defer func() { _ = stdin.Close() }() //nolint:errcheck // best-effort close
		_, _ = stdin.Write(reqBytes)         //nolint:errcheck // best-effort write
		_, _ = stdin.Write([]byte("\n"))     //nolint:errcheck // best-effort write
	}()

	// stdout
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mcp error: %w, output: %s", err, out)
	}

	slog.Debug("mcp stdio output", "output", string(out))
	return nil
}
