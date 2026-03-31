package workspace

import (
	"os"
	"path/filepath"

	"newclaw/internal/store"
)

var templates = map[string]string{
	"AGENTS.md":    "# AGENTS.md\n\nWorkspace do NewClaw.\n",
	"SOUL.md":      "# SOUL.md\n\nSeja util, direto e confiavel.\n",
	"USER.md":      "# USER.md\n\n- Name:\n- Timezone:\n- Notes:\n",
	"IDENTITY.md":  "# IDENTITY.md\n\n- Name:\n- Vibe:\n",
	"TOOLS.md":     "# TOOLS.md\n\nNotas locais de ferramentas e ambiente.\n",
	"HEARTBEAT.md": "# HEARTBEAT.md\n\n# Keep empty to skip periodic checks.\n",
	"BOOTSTRAP.md": "# BOOTSTRAP.md\n\nPrimeiro contato: defina identidade e usuario.\n",
}

func WorkspaceDir(root string) string {
	return filepath.Join(root, ".newclaw", "workspace")
}

func Ensure(root string) error {
	dir := WorkspaceDir(root)
	if err := store.EnsureDir(dir); err != nil {
		return err
	}
	for name, content := range templates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadCoreFiles(root string) (map[string]string, error) {
	dir := WorkspaceDir(root)
	out := map[string]string{}
	for name := range templates {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		out[name] = string(b)
	}
	return out, nil
}
