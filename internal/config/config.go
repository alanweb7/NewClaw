package config

import (
	"errors"
	"os"
	"path/filepath"

	"newclaw/internal/store"
	"newclaw/pkg/types"
)

func DefaultConfig() types.RuntimeConfig {
	return types.RuntimeConfig{
		Version: 1,
		HTTP: types.HTTPConfig{
			Host: "127.0.0.1",
			Port: 7840,
		},
		Model: types.ModelConfig{
			Provider:        "openai-compatible",
			BaseURL:         "https://api.openai.com/v1",
			Transport:       "openai-chat-completions",
			APIKeyEnv:       "OPENAI_API_KEY",
			DefaultModel:    "gpt-4o-mini",
			MaxOutputTokens: 128,
			StopOnFirstLine: false,
			RequestTimeout:  60,
			MaxRetries:      2,
		},
		Tools: types.ToolPolicy{
			ExecAllow: []string{},
			ExecDeny:  []string{"rm", "rmdir", "del", "format", "shutdown", "reboot"},
		},
	}
}

func RuntimeDir(root string) string {
	return filepath.Join(root, ".newclaw")
}

func ConfigPath(root string) string {
	return filepath.Join(RuntimeDir(root), "newclaw.json")
}

func LoadOrCreate(root string) (types.RuntimeConfig, error) {
	path := ConfigPath(root)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		if err := store.WriteJSON(path, cfg); err != nil {
			return types.RuntimeConfig{}, err
		}
		return cfg, nil
	}

	var cfg types.RuntimeConfig
	if err := store.ReadJSON(path, &cfg); err != nil {
		return types.RuntimeConfig{}, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func applyDefaults(cfg *types.RuntimeConfig) {
	d := DefaultConfig()
	if cfg.Version == 0 {
		cfg.Version = d.Version
	}
	if cfg.HTTP.Host == "" {
		cfg.HTTP.Host = d.HTTP.Host
	}
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = d.HTTP.Port
	}
	if cfg.Model.Provider == "" {
		cfg.Model.Provider = d.Model.Provider
	}
	if cfg.Model.BaseURL == "" {
		cfg.Model.BaseURL = d.Model.BaseURL
	}
	if cfg.Model.Transport == "" {
		cfg.Model.Transport = d.Model.Transport
	}
	if cfg.Model.APIKeyEnv == "" {
		cfg.Model.APIKeyEnv = d.Model.APIKeyEnv
	}
	if cfg.Model.DefaultModel == "" {
		cfg.Model.DefaultModel = d.Model.DefaultModel
	}
	if cfg.Model.MaxOutputTokens == 0 {
		cfg.Model.MaxOutputTokens = d.Model.MaxOutputTokens
	}
	if cfg.Model.RequestTimeout == 0 {
		cfg.Model.RequestTimeout = d.Model.RequestTimeout
	}
	if cfg.Model.MaxRetries == 0 {
		cfg.Model.MaxRetries = d.Model.MaxRetries
	}
	if cfg.Tools.ExecDeny == nil {
		cfg.Tools.ExecDeny = d.Tools.ExecDeny
	}
	if cfg.Tools.ExecAllow == nil {
		cfg.Tools.ExecAllow = d.Tools.ExecAllow
	}
}
