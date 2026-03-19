package internal

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mirror/mirror/internal/model"
)

// GoPlugin generates simple Go structs.
type GoPlugin struct{}

func (p *GoPlugin) Name() string { return "go_mrr_parser" }

func (p *GoPlugin) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	files := []model.GeneratedFile{}
	for _, s := range schemas {
		name := ApplyFormat(s.Name, cfg.Format)
		filename := fmt.Sprintf("%s%s.go", name, cfg.Suffix)
		var b strings.Builder
		b.WriteString("package " + filepath.Base(cfg.Path) + "\n\n")
		b.WriteString(fmt.Sprintf("type %s struct {\n", name))
		for _, f := range s.Fields {
			fieldName := strings.Title(f.Name)
			fieldType := goTypeForField(f.Type)
			b.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, f.Name))
		}
		b.WriteString("}\n")
		files = append(files, model.GeneratedFile{Path: filename, Content: b.String()})
	}
	return files, nil
}

func goTypeForField(t string) string {
	switch strings.ToLower(t) {
	case "string":
		return "string"
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	default:
		if strings.HasPrefix(strings.ToLower(t), "object:") {
			return strings.Title(strings.TrimPrefix(t, "object:"))
		}
		return "interface{}"
	}
}