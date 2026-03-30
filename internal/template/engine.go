package template

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/mirror/mirror/internal/functions"
	"github.com/mirror/mirror/internal/model"
)

// Engine parses and executes MRR templates.
type Engine struct {
	Funcs template.FuncMap
}

// Render processes MRR template for all provided schemas.
func (e *Engine) Render(tmplContent string, schemas []*model.Schema, cfg model.OutputConfig, extraFuncs template.FuncMap) ([]model.GeneratedFile, error) {
	funcs := template.FuncMap{
		"input": func() map[string]any {
			return map[string]any{
				"Config": cfg,
			}
		},
	}
	maps.Copy(funcs, e.Funcs)
	maps.Copy(funcs, extraFuncs)

	// Resolve dynamic plugin functions from cfg.Plugins
	pluginFuncs := functions.ResolveFuncs(cfg.Plugins)
	maps.Copy(funcs, pluginFuncs)

	// text/template does not support colons in function identifiers.
	// Transpile fn:alias:func to fn_alias_func
	re := regexp.MustCompile(`\bfn:([a-zA-Z0-9_]+):([a-zA-Z0-9_]+)\b`)
	tmplContent = re.ReplaceAllString(tmplContent, "fn_${1}_$2")

	tmpl, err := template.New("mrr").Funcs(funcs).Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var results []model.GeneratedFile
	for _, schema := range schemas {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, schema)
		if err != nil {
			return nil, fmt.Errorf("execute error on schema %s: %w", schema.Name, err)
		}

		fileName := model.ApplyFormat(schema.Name, cfg.Format) + cfg.Suffix
		
		// Handle per-schema subpath from meta
		if langMeta, ok := schema.Meta[cfg.Language]; ok {
			if subPath, ok := langMeta["filepath"].(string); ok {
				fileName = filepath.Join(subPath, fileName)
			}
		}

		// The actual extension is appended by the generator plugins.
		results = append(results, model.GeneratedFile{
			Path:    fileName,
			Content: buf.String(),
		})
	}
	return results, nil
}
