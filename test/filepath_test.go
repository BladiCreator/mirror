package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BladiCreator/mirror/internal/generator"
	"github.com/BladiCreator/mirror/internal/languages"
	"github.com/BladiCreator/mirror/internal/parser"
)

func TestPerSchemaFilepath(t *testing.T) {
	tmpDir := t.TempDir()

	yamlContent := `
languages:
  - go:
      filepath: "./out"
schemas:
  - name: User
    meta:
      go:
        filepath: "auth/"
    fields:
      - name: id
        type: int
`
	yamlPath := filepath.Join(tmpDir, "mirror.yml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	mrr, err := parser.ParseFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	reg := languages.NewRegistry("")
	res, err := generator.Generate(mrr, reg, tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	expectedPath := filepath.Join(tmpDir, "out", "auth", "User.go")
	for _, f := range res.WrittenFiles {
		if f == expectedPath {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected file at %s not found. Written files: %v", expectedPath, res.WrittenFiles)
	}

	// Check if file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("file %s does not exist on disk", expectedPath)
	}
}
