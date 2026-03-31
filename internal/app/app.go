package app

import (
	"path/filepath"

	"newclaw/internal/agents"
	"newclaw/internal/auth"
	"newclaw/internal/config"
	"newclaw/internal/identity"
	"newclaw/internal/store"
	"newclaw/internal/workspace"
	"newclaw/pkg/types"
)

func Bootstrap(root string) (types.RuntimeConfig, *agents.Service, error) {
	runtimeDir := config.RuntimeDir(root)
	if err := store.EnsureDir(runtimeDir); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	if err := store.EnsureDir(filepath.Join(runtimeDir, "skills")); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	cfg, err := config.LoadOrCreate(root)
	if err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	if err := workspace.Ensure(root); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	if _, _, err := identity.Ensure(root); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	if err := auth.Ensure(root); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	if err := agents.EnsureAgent(root, cfg, "main"); err != nil {
		return types.RuntimeConfig{}, nil, err
	}
	return cfg, agents.NewService(root, cfg), nil
}
