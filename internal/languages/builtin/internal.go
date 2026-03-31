package builtin

import (
	"path/filepath"
	"strings"

	pm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/template"
)

// InternalLanguage returns built-in language initialized with the template engine.
func InternalLanguage() []pm.Language {
	funcs := map[string]any{
		"formatName":    model.ApplyFormat,
		"filepath_Base": filepath.Base,
	}
	eng := &template.Engine{
		Funcs: funcs,
	}
	return []pm.Language{
		&GoLanguage{Engine: eng},
		&DartLanguage{Engine: eng},
		&TypeScriptLanguage{Engine: eng},
		&SurrealQLLanguage{Engine: eng},
	}
}

// ResolveTypeHelper is a utility for languages to handle the "baseType lang:override" syntax.
// It returns the override if found for the given lang, otherwise it returns the baseType.
func ResolveTypeHelper(lang, typeStr string) (baseType string, override string) {
	parts := strings.Fields(typeStr)
	if len(parts) == 0 {
		return "", ""
	}

	baseType = parts[0]
	prefix := lang + ":"
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, prefix) {
			return baseType, strings.TrimPrefix(p, prefix)
		}
	}
	return baseType, ""
}

// FilterFieldsByOmit filters schema fields based on meta.<lang>.binding.omit settings.
func FilterFieldsByOmit(lang string, schemas []*model.Schema) []*model.Schema {
	filteredSchemas := make([]*model.Schema, len(schemas))
	for i, s := range schemas {
		newS := *s
		var newFields []*model.Field
		omits := make(map[string]bool)
		if langMeta, ok := s.Meta[lang]; ok {
			if binding, ok := langMeta["binding"].(map[string]any); ok {
				if omitList, ok := binding["omit"].([]any); ok {
					for _, o := range omitList {
						if name, ok := o.(string); ok {
							omits[name] = true
						}
					}
				}
			}
		}
		for _, f := range s.Fields {
			if !omits[f.Name] {
				newFields = append(newFields, f)
			}
		}
		newS.Fields = newFields
		filteredSchemas[i] = &newS
	}
	return filteredSchemas
}
