package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"newclaw/internal/httpapi"
)

func TestBootstrapAndChatPersistence(t *testing.T) {
	root := t.TempDir()
	_, svc, err := Bootstrap(root)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := svc.SendMessage(context.Background(), "main", "", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Role != "assistant" {
		t.Fatalf("expected assistant role, got %s", resp.Role)
	}

	list, err := svc.ListSessions("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one session")
	}
	if _, err := os.Stat(filepath.Join(root, ".newclaw", "agents", "main", "sessions", "sessions.json")); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPSessionMessageFlow(t *testing.T) {
	root := t.TempDir()
	_, svc, err := Bootstrap(root)
	if err != nil {
		t.Fatal(err)
	}
	server := httpapi.New(root, svc)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	createResp, err := http.Post(ts.URL+"/v1/sessions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer createResp.Body.Close()
	var created struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.SessionID == "" {
		t.Fatal("empty session id")
	}

	msgBody := `{"message":"ping"}`
	msgResp, err := http.Post(ts.URL+"/v1/sessions/"+created.SessionID+"/messages", "application/json", strings.NewReader(msgBody))
	if err != nil {
		t.Fatal(err)
	}
	defer msgResp.Body.Close()
	var event struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(msgResp.Body).Decode(&event); err != nil {
		t.Fatal(err)
	}
	if event.Role != "assistant" {
		t.Fatalf("expected assistant role, got %s", event.Role)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = ctx
}
