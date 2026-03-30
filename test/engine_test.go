package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
)

func TestEngineRender(t *testing.T) {
	eng := &template.Engine{
		Funcs: map[string]any{
			"formatName": model.ApplyFormat,
			"typeForLang": func(l, typ string) string {
				if l == "dart" && typ == "int" {
					return "num"
				}
				return typ
			},
			"filepath_Base": filepath.Base,
		},
	}

	tmpl := `package {{ filepath_Base input.Config.Filepath }}

type {{ formatName .Name "pascal" }} struct {
{{ range $_, $field := .Fields }}
  {{ formatName $field.Name "pascal" }} {{ typeForLang "go" $field.Type }} ` + "`json:\"{{ $field.Name }}\"`" + `
{{ end }}
}
`

	schemas := []*model.Schema{
		{
			Name: "usuario",
			Fields: []*model.Field{
				{Name: "id", Type: "int"},
				{Name: "nombre", Type: "string"},
			},
		},
	}

	cfg := model.OutputConfig{
		Filepath: "./models",
		Format:   "pascal",
	}

	files, err := eng.Render(tmpl, schemas, cfg, nil)
	if err != nil {
		t.Fatalf("engine error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Path != "Usuario" {
		t.Errorf("expected path Usuario, got %s", files[0].Path)
	}

	expectedCode := `package models

type Usuario struct {

  Id int ` + "`json:\"id\"`" + `

  Nombre string ` + "`json:\"nombre\"`" + `

}
`
	if strings.TrimSpace(files[0].Content) != strings.TrimSpace(expectedCode) {
		t.Errorf("expected:\n%s\ngot:\n%s", expectedCode, files[0].Content)
	}
}
