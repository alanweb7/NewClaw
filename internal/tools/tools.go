package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"newclaw/pkg/types"
)

type Executor struct {
	root   string
	policy types.ToolPolicy
}

func NewExecutor(root string, policy types.ToolPolicy) *Executor {
	return &Executor{root: root, policy: policy}
}

func Specs() []types.ToolSpec {
	return []types.ToolSpec{
		{Name: "read", Description: "Read a file content"},
		{Name: "write", Description: "Write content to a file"},
		{Name: "edit", Description: "Replace old text with new text in a file"},
		{Name: "apply_patch", Description: "Apply plain-text patch (basic)"},
		{Name: "exec", Description: "Execute a shell command"},
	}
}

func (e *Executor) Run(ctx context.Context, req types.ToolRequest) types.ToolResult {
	switch req.Name {
	case "read":
		return e.read(req.Params)
	case "write":
		return e.write(req.Params)
	case "edit":
		return e.edit(req.Params)
	case "apply_patch":
		return e.applyPatch(req.Params)
	case "exec":
		return e.execCmd(ctx, req.Params)
	default:
		return types.ToolResult{OK: false, Error: "unknown tool"}
	}
}

func (e *Executor) read(params map[string]interface{}) types.ToolResult {
	path := asString(params, "path")
	if path == "" {
		return errResult("missing path")
	}
	full := e.resolve(path)
	b, err := os.ReadFile(full)
	if err != nil {
		return errResult(err.Error())
	}
	return types.ToolResult{OK: true, Output: string(b)}
}

func (e *Executor) write(params map[string]interface{}) types.ToolResult {
	path := asString(params, "path")
	content := asString(params, "content")
	if path == "" {
		return errResult("missing path")
	}
	full := e.resolve(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return errResult(err.Error())
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return errResult(err.Error())
	}
	return types.ToolResult{OK: true, Output: "written"}
}

func (e *Executor) edit(params map[string]interface{}) types.ToolResult {
	path := asString(params, "path")
	oldText := asString(params, "old_text")
	newText := asString(params, "new_text")
	if path == "" {
		return errResult("missing path")
	}
	if oldText == "" {
		return errResult("missing old_text")
	}
	full := e.resolve(path)
	b, err := os.ReadFile(full)
	if err != nil {
		return errResult(err.Error())
	}
	orig := string(b)
	if !strings.Contains(orig, oldText) {
		return errResult("old_text not found")
	}
	updated := strings.ReplaceAll(orig, oldText, newText)
	if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
		return errResult(err.Error())
	}
	return types.ToolResult{OK: true, Output: "edited"}
}

func (e *Executor) applyPatch(params map[string]interface{}) types.ToolResult {
	patch := asString(params, "patch")
	if strings.TrimSpace(patch) == "" {
		return errResult("missing patch")
	}
	// Basic v1 behavior: store patch text to artifacts for auditing.
	out := filepath.Join(e.root, ".newclaw", "logs", "last_apply_patch.diff")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return errResult(err.Error())
	}
	if err := os.WriteFile(out, []byte(patch), 0o644); err != nil {
		return errResult(err.Error())
	}
	return types.ToolResult{OK: true, Output: "patch recorded"}
}

func (e *Executor) execCmd(ctx context.Context, params map[string]interface{}) types.ToolResult {
	command := asString(params, "command")
	if command == "" {
		return errResult("missing command")
	}
	if err := e.validateCommand(command); err != nil {
		return errResult(err.Error())
	}

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	cmd.Dir = e.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return types.ToolResult{OK: false, Output: string(out), Error: err.Error()}
	}
	return types.ToolResult{OK: true, Output: string(out)}
}

func (e *Executor) validateCommand(command string) error {
	first := firstToken(command)
	if first == "" {
		return errors.New("invalid command")
	}
	for _, d := range e.policy.ExecDeny {
		if first == strings.ToLower(strings.TrimSpace(d)) {
			return fmt.Errorf("command denied by policy: %s", first)
		}
	}
	if len(e.policy.ExecAllow) == 0 {
		return nil
	}
	for _, a := range e.policy.ExecAllow {
		if first == strings.ToLower(strings.TrimSpace(a)) {
			return nil
		}
	}
	return fmt.Errorf("command not allowed by policy: %s", first)
}

func firstToken(command string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(command)))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func asString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func errResult(msg string) types.ToolResult {
	return types.ToolResult{OK: false, Error: msg}
}

func (e *Executor) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.root, path)
}
