package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	lm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/languages/tools"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
	"github.com/bmatcuk/doublestar/v4"
)

// TypeScriptLanguage generates TypeScript interfaces or types using MRR templates.
type TypeScriptLanguage struct {
	Engine *template.Engine
}

func (p *TypeScriptLanguage) Name() string      { return "typescript" }
func (p *TypeScriptLanguage) Aliases() []string { return []string{"ts"} }

const defaultTypeScriptTemplate = `{{ with imports . }}{{ range . }}{{ . }}
{{ end }}{{ end }}
export {{ if eq (getMeta . "typescript" "kind") "type" }}type{{ else }}interface{{ end }} {{ formatName .Name "pascal" }} {
{{ range $_, $field := .Fields }}
  {{ $field.Name }}: {{ type $field.Type }};
{{ end }}
}
`

func (p *TypeScriptLanguage) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	tmpl := defaultTypeScriptTemplate
	if cfg.Template != "" {
		content, err := os.ReadFile(cfg.Template)
		if err != nil {
			return nil, err
		}
		tmpl = string(content)
	}

	// Filter fields based on meta.typescript.binding.omit
	filteredSchemas := tools.FilterFieldsByOmit("typescript", schemas)

	extraFuncs := map[string]any{
		"type": p.ResolveType,
		"getMeta": func(s *model.Schema, lang string, key string) any {
			if l, ok := s.Meta[lang]; ok {
				return l[key]
			}
			return nil
		},
		"imports": func(s *model.Schema) []string {
			var res []string
			disable := false
			if s.Import != nil {
				disable = s.Import.Disable
			}

			if disable {
				return res
			}

			for _, imp := range s.Import.Langs["typescript"] {
				if after, ok := strings.CutPrefix(imp, "auto:"); ok {
					name := after
					// Convert schema name to file name (kebab case usually for TS files)
					fileName := model.ApplyFormat(name, "kebab") + cfg.Suffix
					res = append(res, fmt.Sprintf("import { %s } from './%s';", model.ApplyFormat(name, "pascal"), fileName))
				} else {
					res = append(res, imp)
				}
			}
			return res
		},
	}

	files, err := p.Engine.Render(tmpl, filteredSchemas, cfg, extraFuncs)
	for i := range files {
		files[i].Path += ".ts"
	}
	return files, err
}

func (p *TypeScriptLanguage) ResolveType(t string) string {
	return TypeScriptTypeMapper(t)
}

func TypeScriptTypeMapper(typeStr string) string {
	base, override := tools.ResolveTypeHelper("typescript", typeStr)
	if override != "" {
		return override
	}
	if after, ok := strings.CutPrefix(base, "object:"); ok {
		return model.ApplyFormat(after, "pascal")
	}
	if after, ok := strings.CutPrefix(base, "list:"); ok {
		return TypeScriptTypeMapper(after) + "[]"
	}
	switch base {
	case "int", "float":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	default:
		return base
	}
}

func (p *TypeScriptLanguage) Analyzer() lm.Analyzer {
	return &TypeScriptAnalyzer{}
}

func (p *TypeScriptLanguage) Template() (string, error) {
	return defaultTypeScriptTemplate, nil
}

type TypeScriptAnalyzer struct{}

func (a *TypeScriptAnalyzer) Detect(dir string, pattern string) (int, error) {
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
		} else if pattern == "" && (filepath.Ext(path) == ".ts" || filepath.Ext(path) == ".tsx") {
			count++
		}
		return nil
	})
	return count, err
}

func (a *TypeScriptAnalyzer) Extract(dir string, pattern string) ([]*model.Schema, error) {
	var schemas []*model.Schema
	// Basic regex to find interfaces and types
	interfaceRegex := regexp.MustCompile(`(?s)export\s+(interface|type)\s+(\w+)\s*[\{=]\s*([^;\}]*)[;\}]`)
	fieldRegex := regexp.MustCompile(`(\w+)\s*:\s*([^;,\n]+)`)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(dir, path)
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			return err
		}
		if pattern != "" && !matched {
			return nil
		} else if pattern == "" && (filepath.Ext(path) != ".ts" && filepath.Ext(path) != ".tsx") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		matches := interfaceRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			kind := match[1]
			name := match[2]
			body := match[3]

			schema := &model.Schema{
				Name: name,
				Meta: map[string]map[string]any{
					"typescript": {"filepath": relPath, "kind": kind},
				},
				Fields: []*model.Field{},
			}

			fieldMatches := fieldRegex.FindAllStringSubmatch(body, -1)
			for _, fm := range fieldMatches {
				fieldName := strings.TrimSpace(fm[1])
				fieldType := strings.TrimSpace(fm[2])
				schema.Fields = append(schema.Fields, &model.Field{
					Name: fieldName,
					Type: a.tsTypeToMirror(fieldType),
				})
			}
			if len(schema.Fields) > 0 {
				schemas = append(schemas, schema)
			}
		}
		return nil
	})
	return schemas, err
}

func (a *TypeScriptAnalyzer) tsTypeToMirror(ty string) string {
	ty = strings.TrimSpace(ty)
	if before, ok := strings.CutSuffix(ty, "[]"); ok {
		return "list:" + a.tsTypeToMirror(before)
	}
	switch ty {
	case "number":
		return "int"
	case "string":
		return "string"
	case "boolean":
		return "bool"
	default:
		return "object:" + ty
	}
}
