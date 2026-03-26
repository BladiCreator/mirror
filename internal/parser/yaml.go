package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/mirror/mirror/internal/model"
	"gopkg.in/yaml.v3"
)

func parseYAMLFile(path string, visited map[string]bool, schemaOnly bool) (*model.MRRFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type yamlSchemaInline struct {
		Name   string                    `yaml:"name"`
		Meta   map[string]map[string]any `yaml:"meta"`
		Fields []struct {
			Name string                    `yaml:"name"`
			Type string                    `yaml:"type"`
			Meta map[string]map[string]any `yaml:"meta"`
		} `yaml:"fields"`
		Include string `yaml:"include"`
		Import  any    `yaml:"import"`
	}

	type yamlFile struct {
		Plugin    []string                          `yaml:"plugin"`
		Languages []map[string]model.LanguageConfig `yaml:"languages"`
		Lang      []map[string]model.LanguageConfig `yaml:"lang"`
		Schemas   []yamlSchemaInline                `yaml:"schemas"`
	}

	var parsed yamlFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	mrr := &model.MRRFile{
		Filepath:  path,
		Languages: make(map[string]model.LanguageConfig),
		Schemas:   map[string]*model.Schema{},
		Plugins:   parsed.Plugin,
	}

	langs := parsed.Languages
	if len(langs) == 0 {
		langs = parsed.Lang
	}

	for _, langMap := range langs {
		for langName, config := range langMap {
			if config.Filepath == "" {
				config.Filepath = langName
			}
			mrr.Languages[langName] = config
		}
	}

	for _, s := range parsed.Schemas {
		if s.Include != "" {
			importPath, err := normalizeImportPath(s.Include, path)
			if err != nil {
				return nil, err
			}
			mrr.Imports = append(mrr.Imports, importPath)
			child, err := parseFile(importPath, visited, true)
			if err != nil {
				return nil, err
			}
			for k, v := range child.Schemas {
				if _, exists := mrr.Schemas[k]; exists {
					return nil, fmt.Errorf("schema duplicate %q from import %s", k, importPath)
				}
				mrr.Schemas[k] = v
			}
			continue
		}

		if s.Name != "" {
			if _, exists := mrr.Schemas[s.Name]; exists {
				return nil, fmt.Errorf("schema %q already defined in %s", s.Name, path)
			}
			schema := &model.Schema{
				Name:   s.Name,
				Meta:   s.Meta,
				Fields: []*model.Field{},
				Import: processImport(s.Import),
			}
			for _, f := range s.Fields {
				if strings.TrimSpace(f.Name) == "" || strings.TrimSpace(f.Type) == "" {
					return nil, fmt.Errorf("invalid field %q for schema %s in %s", f.Name, s.Name, path)
				}
				schema.Fields = append(schema.Fields, &model.Field{
					Name: f.Name,
					Type: f.Type,
					Meta: f.Meta,
				})
			}
			mrr.Schemas[s.Name] = schema
		}
	}

	// Automatic imports per language defaults
	// Go defaults to disable: true, Dart defaults to disable: false
	for _, schema := range mrr.Schemas {
		if schema.Import == nil {
			schema.Import = &model.ImportConfig{Langs: make(map[string][]string)}
		}

		if schema.Import.Disable {
			continue
		}

		for _, field := range schema.Fields {
			if strings.HasPrefix(field.Type, "object:") {
				target := strings.TrimPrefix(field.Type, "object:")
				// Base logic: if not disabled for the language, we add it.
				// Since we don't know the exact file name convention here without the generator,
				// we'll store the targeted object names and let the generators handle the actual path.
				for langName := range mrr.Languages {
					// Apply language specific defaults if not overridden
					// In this context, if Disable is false, we proceed.
					// We'll use a special prefix "auto:" to distinguish these.
					schema.Import.Langs[langName] = append(schema.Import.Langs[langName], "auto:"+target)
				}
			}
		}
	}

	if !schemaOnly {
		if err := Validate(mrr); err != nil {
			return nil, err
		}
	}

	return mrr, nil
}

func processImport(val any) *model.ImportConfig {
	config := &model.ImportConfig{Langs: make(map[string][]string)}
	if val == nil {
		return config
	}

	switch v := val.(type) {
	case bool:
		config.Disable = v
	case map[string]any:
		if d, ok := v["disable"].(bool); ok {
			config.Disable = d
		}
		for lang, imps := range v {
			if lang == "disable" {
				continue
			}
			addImports(config, lang, imps)
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				for lang, imps := range m {
					if lang == "disable" {
						if d, ok := imps.(bool); ok {
							config.Disable = d
						}
						continue
					}
					addImports(config, lang, imps)
				}
			}
		}
	}
	return config
}

func addImports(config *model.ImportConfig, lang string, imps any) {
	if impList, ok := imps.([]any); ok {
		for _, imp := range impList {
			if s, ok := imp.(string); ok {
				config.Langs[lang] = append(config.Langs[lang], s)
			}
		}
	} else if impStr, ok := imps.(string); ok {
		config.Langs[lang] = append(config.Langs[lang], impStr)
	}
}