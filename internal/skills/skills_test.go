package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterAndList(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".newclaw", "skills", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: demo\ndescription: test skill\n---\n# Demo\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(list))
	}
	if list[0].Name != "demo" || list[0].Description != "test skill" {
		t.Fatalf("unexpected parsed skill: %+v", list[0])
	}
}
