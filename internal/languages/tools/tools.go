package tools

import (
	"strings"

	"github.com/BladiCreator/mirror/internal/model"
)

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
		if after, ok := strings.CutPrefix(p, prefix); ok {
			return baseType, after
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
