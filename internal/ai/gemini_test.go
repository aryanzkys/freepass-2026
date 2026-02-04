package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateJSONValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"summary\":\"ok\",\"sentiment\":{\"label\":\"neutral\",\"score\":0},\"top_themes\":[],\"action_items\":[],\"notable_quotes\":[]}"}]}}]}`))
	}))
	defer server.Close()

	client := NewClient("key", "model", time.Second)
	client.baseURL = server.URL

	payload, err := client.GenerateJSON(context.Background(), "system", "user", map[string]interface{}{"type": "object"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(payload) {
		t.Fatalf("invalid json payload: %s", string(payload))
	}
}

func TestGenerateJSONInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"not-json"}]}}]}`))
	}))
	defer server.Close()

	client := NewClient("key", "model", time.Second)
	client.baseURL = server.URL

	_, err := client.GenerateJSON(context.Background(), "system", "user", map[string]interface{}{"type": "object"})
	if err == nil {
		t.Fatalf("expected error")
	}
	var invalid InvalidResponseError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected InvalidResponseError, got: %v", err)
	}
}
