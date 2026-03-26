package builtin

import (
	"path/filepath"
	"strings"

	pm "github.com/mirror/mirror/internal/languages/model"
	"github.com/mirror/mirror/internal/model"
	"github.com/mirror/mirror/internal/template"
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





