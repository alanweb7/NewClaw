package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateAppliesDefaults(t *testing.T) {
	root := t.TempDir()
	cfg, err := LoadOrCreate(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Port != 7840 {
		t.Fatalf("expected default port 7840, got %d", cfg.HTTP.Port)
	}
	path := ConfigPath(root)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not created: %v", err)
	}

	custom := []byte(`{"version":1,"http":{"host":"","port":0},"model":{"provider":"","base_url":"","api_key_env":"","default_model":"","request_timeout_seconds":0,"max_retries":0},"tools":{"exec_allow":null,"exec_deny":null}}`)
	if err := os.WriteFile(filepath.Join(root, ".newclaw", "newclaw.json"), custom, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadOrCreate(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model.DefaultModel == "" || cfg.HTTP.Host == "" {
		t.Fatal("defaults not applied")
	}
}
