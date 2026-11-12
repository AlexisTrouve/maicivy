package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Verrouille le retry de callClaude sur erreurs transientes du proxy LLM.
// CONTEXTE : ai.etheryale.com renvoie parfois 502 "Proxy unavailable. Please retry." sur les gros
// prompts (CV ~4K tokens). Sans retry, un seul hoquet = 500 pour l'utilisateur. Doctrine : retry.

// newTestGenService construit un service minimal pointant sur un serveur de test (seuls
// baseURL/apiKey/httpClient sont utilisés par callClaude).
func newTestGenService(baseURL string) *CVGenerationService {
	return &CVGenerationService{
		baseURL:    baseURL,
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestCallClaude_RetriesOnTransient502(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1er appel : 502 transient. 2e appel : 200 OK.
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"type":"proxy_error","message":"Proxy unavailable. Please retry."}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"OK_RESULT"}]}`))
	}))
	defer srv.Close()

	out, err := newTestGenService(srv.URL).callClaude(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("attendu succès après retry sur 502, got err: %v", err)
	}
	if out != "OK_RESULT" {
		t.Fatalf("texte inattendu: %q", out)
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Fatalf("attendu 2 appels (1 échec transient + 1 retry), got %d", n)
	}
}

func TestCallClaude_NoRetryOn400(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest) // 400 = erreur cliente, NON retryable
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	_, err := newTestGenService(srv.URL).callClaude(context.Background(), "prompt")
	if err == nil {
		t.Fatal("attendu une erreur sur 400")
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Fatalf("un 400 ne doit PAS être retenté, got %d appels", n)
	}
}

func TestCallClaude_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable) // 503 persistant
	}))
	defer srv.Close()

	_, err := newTestGenService(srv.URL).callClaude(context.Background(), "prompt")
	if err == nil {
		t.Fatal("attendu une erreur après épuisement des tentatives")
	}
	if n := atomic.LoadInt32(&calls); n != 3 {
		t.Fatalf("attendu 3 tentatives (maxAttempts), got %d", n)
	}
}
