package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	lm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
	"github.com/bmatcuk/doublestar/v4"
)

// DartLanguage generates simple Dart classes using MRR templates.
type DartLanguage struct {
	Engine *template.Engine
}

func (p *DartLanguage) Name() string { return "dart" }

const defaultDartTemplate = `{{ range imports . }}{{ . }}
{{ end }}
class {{ formatName .Name "pascal" }} {

{{ range $_, $field := .Fields }}
  final {{ type $field.Type }} {{ $field.Name }};
{{ end }}
  {{ formatName .Name "pascal" }}({
{{ range $_, $field := .Fields }}
    required this.{{ $field.Name }},
{{ end }}
  });
}
`

func (p *DartLanguage) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	tmpl := defaultDartTemplate
	if cfg.Template != "" {
		content, err := os.ReadFile(cfg.Template)
		if err != nil {
			return nil, err
		}
		tmpl = string(content)
	}

	// Filter fields based on meta.dart.binding.omit
	filteredSchemas := FilterFieldsByOmit("dart", schemas)

	extraFuncs := map[string]any{
		"type": p.ResolveType,
		"imports": func(s *model.Schema) []string {
			var res []string
			// Dart default disable is false
			disable := false
			if s.Import != nil {
				disable = s.Import.Disable
			}

			if disable {
				return res
			}

			for _, imp := range s.Import.Langs["dart"] {
				if strings.HasPrefix(imp, "auto:") {
					name := strings.TrimPrefix(imp, "auto:")
					// Convert schema name to file name (snake case by default for files)
					fileName := model.ApplyFormat(name, "snake") + cfg.Suffix + ".dart"
					res = append(res, fmt.Sprintf("import '%s';", fileName))
				} else {
					res = append(res, imp)
				}
			}
			return res
		},
	}

	files, err := p.Engine.Render(tmpl, filteredSchemas, cfg, extraFuncs)
	for i := range files {
		files[i].Path += ".dart"
	}
	return files, err
}

func (p *DartLanguage) ResolveType(t string) string {
	return DartTypeMapper(t)
}

func DartTypeMapper(typeStr string) string {
	base, override := ResolveTypeHelper("dart", typeStr)
	if override != "" {
		return override
	}
	if strings.HasPrefix(base, "object:") {
		return model.ApplyFormat(strings.TrimPrefix(base, "object:"), "pascal")
	}
	switch base {
	case "int":
		return "int"
	case "float":
		return "double"
	case "string":
		return "String"
	case "bool":
		return "bool"
	default:
		return base
	}
}

func (p *DartLanguage) Analyzer() lm.Analyzer {
	return &DartAnalyzer{}
}

func (p *DartLanguage) Template() (string, error) {
	return defaultDartTemplate, nil
}

type DartAnalyzer struct{}

func (a *DartAnalyzer) Detect(dir string, pattern string) (int, error) {
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
		} else if pattern == "" && filepath.Ext(path) == ".dart" {
			count++
		}
		return nil
	})
	return count, err
}

func (a *DartAnalyzer) Extract(dir string, pattern string) ([]*model.Schema, error) {
	var schemas []*model.Schema
	classRegex := regexp.MustCompile(`(?s)class\s+(\w+)\s*\{([^}]*)\}`)
	fieldRegex := regexp.MustCompile(`\bfinal\s+(\w+)\s+(\w+);`)

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
		} else if pattern == "" && filepath.Ext(path) != ".dart" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		matches := classRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			className := match[1]
			body := match[2]

			schema := &model.Schema{
				Name: className,
				Meta: map[string]map[string]any{
					"dart": {"filepath": relPath},
				},
				Fields: []*model.Field{},
			}

			fieldMatches := fieldRegex.FindAllStringSubmatch(body, -1)
			for _, fm := range fieldMatches {
				dartType := fm[1]
				fieldName := fm[2]
				schema.Fields = append(schema.Fields, &model.Field{
					Name: fieldName,
					Type: a.dartTypeToMirror(dartType),
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

func (a *DartAnalyzer) dartTypeToMirror(dt string) string {
	switch dt {
	case "int":
		return "int"
	case "String":
		return "string"
	case "bool":
		return "bool"
	case "double":
		return "float"
	default:
		return "object:" + dt
	}
}
