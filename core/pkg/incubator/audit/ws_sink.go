package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// ── WebSocket Event Sink ────────────────────────────────────────────────────
//
// Real-time audit event streaming via WebSocket. Allows external dashboards,
// monitoring systems, and anomaly detectors to observe the audit stream.
//
// Usage:
//
//	sink := audit.NewWebSocketSink(":8081")
//	bus.AddSink(sink)
//	go sink.ListenAndServe()  // Starts WebSocket server
//
// Clients connect to ws://localhost:8081/ws and receive JSON events.

// WSEvent is the JSON payload sent to WebSocket clients.
type WSEvent struct {
	Type      EventType              `json:"type"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Sequence  int64                  `json:"sequence"` // Monotonic counter
}

// WebSocketSink broadcasts audit events to connected WebSocket clients.
type WebSocketSink struct {
	mu       sync.RWMutex
	addr     string
	clients  map[*websocket.Conn]bool
	sequence int64
	server   *http.Server
	bufSize  int
}

// NewWebSocketSink creates a sink bound to the given address.
func NewWebSocketSink(addr string) *WebSocketSink {
	return &WebSocketSink{
		addr:    addr,
		clients: make(map[*websocket.Conn]bool),
		bufSize: 100,
	}
}

// Record implements the sink interface for the audit Bus.
func (s *WebSocketSink) Record(_ interface{}, eventType EventType, action, resource string, metadata map[string]interface{}) error {
	s.mu.Lock()
	s.sequence++
	seq := s.sequence
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	event := WSEvent{
		Type:      eventType,
		Action:    action,
		Resource:  resource,
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
		Sequence:  seq,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ws sink: marshal failed: %w", err)
	}

	// Broadcast to all connected clients
	var disconnected []*websocket.Conn
	for _, client := range clients {
		if _, err := client.Write(data); err != nil {
			disconnected = append(disconnected, client)
		}
	}

	// Remove disconnected clients
	if len(disconnected) > 0 {
		s.mu.Lock()
		for _, c := range disconnected {
			delete(s.clients, c)
			_ = c.Close()
		}
		s.mu.Unlock()
	}

	return nil
}

// ListenAndServe starts the WebSocket server.
func (s *WebSocketSink) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.Handle("/ws", websocket.Handler(s.handleConnection))
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	slog.Info("audit ws sink listening", "addr", s.addr)
	return s.server.ListenAndServe()
}

// Shutdown cleanly stops the server.
func (s *WebSocketSink) Shutdown() error {
	s.mu.Lock()
	for c := range s.clients {
		_ = c.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)
	s.mu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// ClientCount returns the number of connected clients.
func (s *WebSocketSink) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

func (s *WebSocketSink) handleConnection(ws *websocket.Conn) {
	s.mu.Lock()
	s.clients[ws] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	slog.Info("ws client connected",
		"remote", ws.Request().RemoteAddr,
		"clients_total", clientCount,
	)

	// Send initial handshake
	hello := map[string]interface{}{
		"type":     "CONNECTED",
		"server":   "helm-audit-stream",
		"version":  "1.0.0",
		"sequence": s.sequence,
	}
	data, _ := json.Marshal(hello)
	_, _ = ws.Write(data)

	// Keep connection alive — read until client disconnects
	buf := make([]byte, 512)
	for {
		_, err := ws.Read(buf)
		if err != nil {
			break
		}
	}

	s.mu.Lock()
	delete(s.clients, ws)
	s.mu.Unlock()
	_ = ws.Close()

	slog.Info("ws client disconnected",
		"remote", ws.Request().RemoteAddr,
	)
}

func (s *WebSocketSink) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	count := len(s.clients)
	seq := s.sequence
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"clients":        count,
		"events_emitted": seq,
	})
}
