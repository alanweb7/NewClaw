package tools

import (
	"context"
	"testing"

	"newclaw/pkg/types"
)

func TestExecAllowDenyPolicy(t *testing.T) {
	exec := NewExecutor(t.TempDir(), types.ToolPolicy{
		ExecAllow: []string{"echo"},
		ExecDeny:  []string{"rm"},
	})

	res := exec.Run(context.Background(), types.ToolRequest{Name: "exec", Params: map[string]interface{}{"command": "rm -rf ."}})
	if res.OK {
		t.Fatal("expected deny for rm")
	}

	res = exec.Run(context.Background(), types.ToolRequest{Name: "exec", Params: map[string]interface{}{"command": "Get-Date"}})
	if res.OK {
		t.Fatal("expected not-allowed for non allowlisted command")
	}
}
