package internal

import (
	"fmt"
	"strings"

	"github.com/mirror/mirror/internal/model"
)

// DartPlugin generates simple Dart classes.
type DartPlugin struct{}

func (p *DartPlugin) Name() string { return "dart_mrr_parser" }

func (p *DartPlugin) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	files := []model.GeneratedFile{}
	for _, s := range schemas {
		name := ApplyFormat(s.Name, cfg.Format)
		filename := fmt.Sprintf("%s%s.dart", name, cfg.Suffix)
		var b strings.Builder
		b.WriteString("class " + name + " {\n")
		for _, f := range s.Fields {
			fieldType := dartTypeForField(f.Type)
			b.WriteString(fmt.Sprintf("\tfinal %s %s;\n", fieldType, f.Name))
		}
		b.WriteString("\n\t" + name + "({\n")
		for _, f := range s.Fields {
			b.WriteString(fmt.Sprintf("\t\trequired this.%s,\n", f.Name))
		}
		b.WriteString("\t});\n")
		b.WriteString("}\n")
		files = append(files, model.GeneratedFile{Path: filename, Content: b.String()})
	}
	return files, nil
}

func dartTypeForField(t string) string {
	switch strings.ToLower(t) {
	case "string":
		return "String"
	case "int":
		return "int"
	case "float":
		return "double"
	case "bool":
		return "bool"
	default:
		if strings.HasPrefix(strings.ToLower(t), "object:") {
			return strings.Title(strings.TrimPrefix(t, "object:"))
		}
		return "dynamic"
	}
}
