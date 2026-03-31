package skills

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"newclaw/pkg/types"
)

func SkillsDir(root string) string {
	return filepath.Join(root, ".newclaw", "skills")
}

func List(root string) ([]types.SkillDescriptor, error) {
	dir := SkillsDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]types.SkillDescriptor, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, e.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		name, desc := parseFrontmatter(skillFile)
		if name == "" {
			name = e.Name()
		}
		out = append(out, types.SkillDescriptor{
			Name:        name,
			Description: desc,
			FilePath:    skillFile,
			BaseDir:     skillDir,
			SourceInfo: types.SkillSourceInfo{
				Path:   skillFile,
				Source: "newclaw-managed",
			},
		})
	}
	return out, nil
}

func parseFrontmatter(path string) (name, desc string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	if !s.Scan() || strings.TrimSpace(s.Text()) != "---" {
		return "", ""
	}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "---" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch k {
		case "name":
			name = v
		case "description":
			desc = v
		}
	}
	return name, desc
}
