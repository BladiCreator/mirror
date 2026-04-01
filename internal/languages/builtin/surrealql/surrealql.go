package surrealql

import (
	"os"
	"path/filepath"
	"strings"

	lm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/languages/tools"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
	"github.com/bmatcuk/doublestar/v4"
)

// SurrealQLLanguage generates SurrealQL definitions using MRR templates.
type SurrealQLLanguage struct {
	Engine *template.Engine
}

func (p *SurrealQLLanguage) Name() string { return "surrealql" }

const defaultSurrealQLTemplate = `DEFINE TABLE {{ .Name }} SCHEMAFULL;
{{ range $_, $field := .Fields }}
DEFINE FIELD {{ $field.Name }} ON TABLE {{ $.Name }} TYPE {{ type $field.Type }}
  {{- if (getMeta $field "surrealql" "computed") }} VALUE {{ getMeta $field "surrealql" "computed" }}{{ end }}
  {{- if (getMeta $field "surrealql" "assert") }} ASSERT {{ getMeta $field "surrealql" "assert" }}{{ end }}
  {{- if (getMeta $field "surrealql" "default") }} DEFAULT {{ getMeta $field "surrealql" "default" }}{{ end }}
  {{- if (getMeta $field "surrealql" "readonly") }} READONLY{{ end }}
  {{- if (getMeta $field "surrealql" "permissions") }} PERMISSIONS {{ getMeta $field "surrealql" "permissions" }}{{ end }};
{{ end }}
`

func (p *SurrealQLLanguage) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	tmpl := defaultSurrealQLTemplate
	if cfg.Template != "" {
		content, err := os.ReadFile(cfg.Template)
		if err != nil {
			return nil, err
		}
		tmpl = string(content)
	}

	// Filter fields based on meta.surrealql.binding.omit
	filteredSchemas := tools.FilterFieldsByOmit("surrealql", schemas)

	extraFuncs := map[string]any{
		"type": p.ResolveType,
		"getMeta": func(f any, lang string, key string) any {
			if s, ok := f.(*model.Schema); ok {
				if l, ok := s.Meta[lang]; ok {
					return l[key]
				}
			}
			if field, ok := f.(*model.Field); ok {
				if l, ok := field.Meta[lang]; ok {
					return l[key]
				}
			}
			return nil
		},
	}

	files, err := p.Engine.Render(tmpl, filteredSchemas, cfg, extraFuncs)
	for i := range files {
		files[i].Path += ".surql"
	}
	return files, err
}

func (p *SurrealQLLanguage) ResolveType(t string) string {
	return SurrealQLTypeMapper(t)
}

func SurrealQLTypeMapper(typeStr string) string {
	base, override := tools.ResolveTypeHelper("surrealql", typeStr)
	if override != "" {
		return override
	}
	if _, ok := strings.CutPrefix(base, "object:"); ok {
		return "object"
	}
	if after, ok := strings.CutPrefix(base, "list:"); ok {
		return "array<" + SurrealQLTypeMapper(after) + ">"
	}
	switch base {
	case "int":
		return "int"
	case "float":
		return "float"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "datetime":
		return "datetime"
	case "duration":
		return "duration"
	default:
		return "any"
	}
}

func (p *SurrealQLLanguage) Analyzer() lm.Analyzer {
	return &SurrealQLAnalyzer{}
}

func (p *SurrealQLLanguage) Template() (string, error) {
	return defaultSurrealQLTemplate, nil
}

type SurrealQLAnalyzer struct{}

func (a *SurrealQLAnalyzer) Detect(dir string, pattern string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(dir, path)
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			return err
		}
		if pattern != "" && matched {
			count++
		} else if pattern == "" && (filepath.Ext(path) == ".surql" || filepath.Ext(path) == ".sql") {
			count++
		}
		return nil
	})
	return count, err
}

func (a *SurrealQLAnalyzer) Extract(dir string, pattern string) ([]*model.Schema, error) {
	return nil, nil
}
