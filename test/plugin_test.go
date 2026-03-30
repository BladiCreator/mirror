package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BladiCreator/mirror/internal/generator"
	"github.com/BladiCreator/mirror/internal/languages"
	"github.com/BladiCreator/mirror/internal/parser"
)

func TestPluginFunctions(t *testing.T) {
	// 1. Create a dummy plugin integration test
	tmp := t.TempDir()
	samplePath := filepath.Join(tmp, "sample.yml")
	sample := `plugin:
  - strings:st
languages:
  - go:
      filepath: "./out"
      template: t.mrr
schemas:
  - name: my_schema
    fields:
      - name: email_address
        type: string
        meta:
          go:
            tag: json:"email_address"
      - name: score
        type: "float go:float32"
`

	templateStr := `package models
type {{ .Name | fn:st:toTitle }} struct {
{{ range .Fields }}
  // checking uppercase: {{ .Name | fn:st:toUpper }}
  {{ .Name }} {{ type .Type }}
{{ end }}
}`
	tmplPath := filepath.Join(tmp, "t.mrr")

	if err := os.WriteFile(samplePath, []byte(sample), 0644); err != nil {
		t.Fatalf("failed to write yml: %v", err)
	}
	if err := os.WriteFile(tmplPath, []byte(templateStr), 0644); err != nil {
		t.Fatalf("failed to write tmpl: %v", err)
	}

	// Parse
	mrr, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(mrr.Plugins) != 1 || mrr.Plugins[0] != "strings:st" {
		t.Fatalf("failed to parse plugins from yaml: %v", mrr.Plugins)
	}

	// Generate
	reg := languages.NewRegistry("") // Using internal plugins only

	res, err := generator.Generate(mrr, reg, tmp, false)
	if err != nil {
		t.Fatalf("generation failed: %v, errors: %v", err, res.Errors)
	}

	if len(res.WrittenFiles) != 1 {
		t.Fatalf("expected 1 file generated, got %d", len(res.WrittenFiles))
	}

	content, err := os.ReadFile(res.WrittenFiles[0])
	if err != nil {
		t.Fatalf("read generated file failed: %v", err)
	}

	strContent := string(content)

	if !strings.Contains(strContent, "type My_schema struct") {
		t.Errorf("expected plugin fn:st:toTitle to capitalize structure name, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, "checking uppercase: EMAIL_ADDRESS") {
		t.Errorf("expected plugin fn:st:toUpper to uppercase field name, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, "score float32") {
		t.Errorf("expected type override float32 for score, got:\n%s", strContent)
	}
}
