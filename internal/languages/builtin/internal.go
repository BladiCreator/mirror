package builtin

import (
	"path/filepath"

	"github.com/BladiCreator/mirror/internal/languages/builtin/dart"
	"github.com/BladiCreator/mirror/internal/languages/builtin/golang"
	"github.com/BladiCreator/mirror/internal/languages/builtin/surrealql"
	"github.com/BladiCreator/mirror/internal/languages/builtin/typescript"
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
		&golang.GoLanguage{Engine: eng},
		&dart.DartLanguage{Engine: eng},
		&typescript.TypeScriptLanguage{Engine: eng},
		&surrealql.SurrealQLLanguage{Engine: eng},
	}
}
