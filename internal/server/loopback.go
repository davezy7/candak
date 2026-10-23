package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"candak/internal/models"
)

// DefaultPort is the standard loopback port Candak listens on.
const DefaultPort = 41920

// GrabHandler is invoked when a download request is received from an authorized caller.
type GrabHandler func(req models.GrabRequest) error

// LoopbackServer manages the local HTTP listener for browser extensions and external integrations.
type LoopbackServer struct {
	addr       string
	server     *http.Server
	onGrab     GrabHandler
	mu         sync.Mutex
	isRunning  bool
	listenErr  error
	actualAddr string
}

// Config provides configuration options for the LoopbackServer.
type Config struct {
	Port   int
	OnGrab GrabHandler
}

// NewLoopbackServer constructs a new LoopbackServer.
func NewLoopbackServer(cfg Config) *LoopbackServer {
	port := cfg.Port
	if port <= 0 {
		port = DefaultPort
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return &LoopbackServer{
		addr:   addr,
		onGrab: cfg.OnGrab,
	}
}

// Start begins listening on the loopback interface.
func (s *LoopbackServer) Start() error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.listenErr = err
		s.mu.Unlock()
		return fmt.Errorf("failed to bind loopback address %s: %w", s.addr, err)
	}

	s.actualAddr = listener.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/grab", s.handleGrab)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.isRunning = true
	s.mu.Unlock()

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			s.listenErr = err
			s.isRunning = false
			s.mu.Unlock()
		}
	}()

	return nil
}

// Stop gracefully terminates the loopback HTTP server.
func (s *LoopbackServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isRunning || s.server == nil {
		return nil
	}
	s.isRunning = false
	return s.server.Shutdown(ctx)
}

// IsRunning reports whether the loopback server is active.
func (s *LoopbackServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

// Address returns the bound listener address string.
func (s *LoopbackServer) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actualAddr
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return true // Direct non-browser loopback calls (curl, cli, local tools)
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	// Browser extension origins
	if u.Scheme == "chrome-extension" || u.Scheme == "moz-extension" {
		return true
	}
	// Loopback origins
	if u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" {
		return true
	}
	return false
}

func (s *LoopbackServer) setCORSHeaders(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if !isAllowedOrigin(origin) {
		return false
	}

	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Candak-Source")
	w.Header().Set("Access-Control-Max-Age", "86400")
	return true
}

func (s *LoopbackServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.setCORSHeaders(w, r) {
		http.Error(w, "Forbidden: unauthorized origin", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"app":     "candak",
		"version": "0.1.0",
		"service": "loopback-grabber",
	})
}

func (s *LoopbackServer) handleGrab(w http.ResponseWriter, r *http.Request) {
	if !s.setCORSHeaders(w, r) {
		http.Error(w, "Forbidden: unauthorized origin", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body to 1 MB to prevent unbounded memory allocation
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var req models.GrabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON body: %v", err), http.StatusBadRequest)
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	handler := s.onGrab
	s.mu.Unlock()

	if handler != nil {
		if err := handler(req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle download: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Download grabbed successfully",
		"url":     req.URL,
	})
}
