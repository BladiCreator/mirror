package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirror/mirror/internal/parser"
)

const sample = `plugin
  - dart_mrr_parser
  - go_mrr_parser
  # - ts_mrr_parser

paths # Comment
  - dart ` + "`p::dart_mrr_parser f::'./lib/models' suffix:'_dart' format:snake`" + `
  - go ` + "`p::go_mrr_parser f::'./internal/models' format:pascal`" + `

schemas
  - usuario
    - id ` + "`int`" + `
    - nombre ` + "`string`" + `
    - perfil ` + "`object:perfil`" + `
  - perfil
    - bio ` + "`string`" + `
    - avatar ` + "`string`" + `
`

func TestParseBasic(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.mrr")
	if err := os.WriteFile(path, []byte(sample), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	mrr, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(mrr.Plugins) != 2 || len(mrr.Paths) != 2 || len(mrr.Schemas) != 2 {
		t.Fatalf("unexpected parse counts: plugins=%d paths=%d schemas=%d", len(mrr.Plugins), len(mrr.Paths), len(mrr.Schemas))
	}
	if mrr.Paths[0].Plugins[0] != "dart_mrr_parser" || mrr.Paths[1].Plugins[0] != "go_mrr_parser" {
		t.Fatalf("unexpected path plugins: %v", mrr.Paths)
	}
	if _, ok := mrr.Schemas["usuario"]; !ok {
		t.Fatal("usuario schema missing")
	}
	if _, ok := mrr.Schemas["perfil"]; !ok {
		t.Fatal("perfil schema missing")
	}
}

func TestParseSchemaOnlyImport(t *testing.T) {
	tmp := t.TempDir()
	commonPath := filepath.Join(tmp, "common.mrr")
	common := `schemas
  - direccion
    - calle ` + "`string`" + `
    - ciudad ` + "`string`" + `
`
	if err := os.WriteFile(commonPath, []byte(common), 0644); err != nil {
		t.Fatalf("write common: %v", err)
	}

	rootPath := filepath.Join(tmp, "main.mrr")
	root := fmt.Sprintf(`plugin
  - go_mrr_parser

paths
  - go `+"`p::go_mrr_parser f::'./out/internal/models' format:pascal`"+`

schemas
  - usuario
    - id `+"`int`"+`
  - '%s'
`, filepath.Base(commonPath))
	if err := os.WriteFile(rootPath, []byte(root), 0644); err != nil {
		t.Fatalf("write root: %v", err)
	}

	mrr, err := parser.ParseFile(rootPath)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := mrr.Schemas["usuario"]; !ok {
		t.Fatal("usuario missing")
	}
	if _, ok := mrr.Schemas["direccion"]; !ok {
		t.Fatal("direccion missing from import")
	}
}
