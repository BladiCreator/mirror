package builtin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	lm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
	"github.com/bmatcuk/doublestar/v4"
)

// GoLanguage generates simple Go structs using MRR templates.
type GoLanguage struct {
	Engine *template.Engine
}

func (p *GoLanguage) Name() string { return "go" }

const defaultGoTemplate = `package {{ filepath_Base input.Config.Filepath }}
{{ with imports . }}
import (
{{ range . }}  "{{ . }}"
{{ end }})
{{ end }}
type {{ formatName .Name "pascal" }} struct {
{{ range $_, $field := .Fields }}
  {{ formatName $field.Name "pascal" }} {{ type $field.Type }} ` + "`" + `json:"{{ $field.Name }}"` + "`" + `
{{ end }}
}
`

func (p *GoLanguage) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	tmpl := defaultGoTemplate
	if cfg.Template != "" {
		content, err := os.ReadFile(cfg.Template)
		if err != nil {
			return nil, err
		}
		tmpl = string(content)
	}

	// Filter fields based on meta.go.binding.omit
	filteredSchemas := FilterFieldsByOmit("go", schemas)

	extraFuncs := map[string]any{
		"type": p.ResolveType,
		"imports": func(s *model.Schema) []string {
			var res []string
			// Go default disable is true (usually same package)
			disable := true
			if s.Import != nil {
				disable = s.Import.Disable
			}

			if disable {
				return res
			}

			for _, imp := range s.Import.Langs["go"] {
				if strings.HasPrefix(imp, "auto:") {
					continue
				}
				res = append(res, imp)
			}
			return res
		},
	}

	files, err := p.Engine.Render(tmpl, filteredSchemas, cfg, extraFuncs)
	for i := range files {
		files[i].Path += ".go"
	}
	return files, err
}

func (p *GoLanguage) ResolveType(t string) string {
	return GoTypeMapper(t)
}

func GoTypeMapper(typeStr string) string {
	base, override := ResolveTypeHelper("go", typeStr)
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
		return "float64"
	case "string":
		return "string"
	case "bool":
		return "bool"
	default:
		return base
	}
}

func (p *GoLanguage) Analyzer() lm.Analyzer {
	return &GoAnalyzer{}
}

func (p *GoLanguage) Template() (string, error) {
	return defaultGoTemplate, nil
}

type GoAnalyzer struct{}

func (a *GoAnalyzer) Detect(dir string, pattern string) (int, error) {
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
		} else if pattern == "" && filepath.Ext(path) == ".go" {
			count++
		}
		return nil
	})
	return count, err
}

func (a *GoAnalyzer) Extract(dir string, pattern string) ([]*model.Schema, error) {
	var schemas []*model.Schema
	fset := token.NewFileSet()

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
		} else if pattern == "" && filepath.Ext(path) != ".go" {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			schema := &model.Schema{
				Name: ts.Name.Name,
				Meta: map[string]map[string]any{
					"go": {"filepath": relPath},
				},
				Fields: []*model.Field{},
			}

			for _, f := range st.Fields.List {
				if len(f.Names) == 0 {
					continue
				}
				for _, fieldName := range f.Names {
					fieldType := a.goTypeToString(f.Type)
					if fieldType != "" {
						schema.Fields = append(schema.Fields, &model.Field{
							Name: fieldName.Name,
							Type: fieldType,
						})
					}
				}
			}
			if len(schema.Fields) > 0 {
				schemas = append(schemas, schema)
			}
			return true
		})
		return nil
	})

	return schemas, err
}

func (a *GoAnalyzer) goTypeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "int", "int32", "int64":
			return "int"
		case "string":
			return "string"
		case "bool":
			return "bool"
		case "float32", "float64":
			return "float"
		default:
			return "object:" + t.Name
		}
	case *ast.StarExpr:
		return a.goTypeToString(t.X)
	case *ast.ArrayType:
		return "list:" + a.goTypeToString(t.Elt)
	}
	return ""
}
