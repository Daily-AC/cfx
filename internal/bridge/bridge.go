package bridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const Address = "127.0.0.1:27121"
const WakeURL = "http://" + Address + "/wake"

type Job struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Result struct {
	ID    string          `json:"id"`
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type server struct {
	job       Job
	mu        sync.Mutex
	delivered bool
	result    chan Result
	token     string
}

func Run(ctx context.Context, token, jobType string, payload any, timeout time.Duration) (json.RawMessage, error) {
	if token == "" {
		return nil, errors.New("Chrome extension token is missing; run `cfx extension install`")
	}
	listener, err := net.Listen("tcp", Address)
	if err != nil {
		return nil, fmt.Errorf("start Chrome extension bridge on %s: %w", Address, err)
	}
	defer listener.Close()

	s := &server{
		job:    Job{ID: randomID(), Type: jobType, Payload: payload},
		result: make(chan Result, 1),
		token:  token,
	}
	httpServer := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = httpServer.Serve(listener) }()
	defer httpServer.Shutdown(context.Background())

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case <-waitCtx.Done():
		return nil, fmt.Errorf("Chrome extension did not complete %s: %w; run `cfx extension install` and keep Chrome open", jobType, waitCtx.Err())
	case result := <-s.result:
		if !result.OK {
			return nil, errors.New(result.Error)
		}
		return result.Data, nil
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/job", s.handleJob)
	mux.HandleFunc("/v1/result", s.handleResult)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/wake" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><title>cfx bridge</title>Connecting cfx to Chrome..."))
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && !strings.HasPrefix(origin, "chrome-extension://") {
			http.Error(w, "cfx bridge only accepts Chrome extension requests", http.StatusForbidden)
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Cfx-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Cfx-Token")), []byte(s.token)) != 1 {
			http.Error(w, "invalid cfx extension token", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *server) handleJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.delivered {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.delivered = true
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.job)
}

func (s *server) handleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var result Result
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20)).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if result.ID != s.job.ID {
		http.Error(w, "job id mismatch", http.StatusConflict)
		return
	}
	select {
	case s.result <- result:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "result already received", http.StatusConflict)
	}
}

func randomID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}
