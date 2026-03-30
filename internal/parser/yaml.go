package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/BladiCreator/mirror/internal/model"
	"gopkg.in/yaml.v3"
)

func parseYAMLFile(path string, visited map[string]bool, schemaOnly bool) (*model.MirrorFile, error) {
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
		Binding []string `yaml:"binding"`
		Include string   `yaml:"include"`
		Import  any      `yaml:"import"`
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

	mrr := &model.MirrorFile{
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
				Name:    s.Name,
				Meta:    s.Meta,
				Binding: s.Binding,
				Fields:  []*model.Field{},
				Import:  processImport(s.Import),
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

	// Resolve bindings
	if err := resolveBindings(mrr, mrr.Schemas); err != nil {
		return nil, err
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

func resolveBindings(mrr *model.MirrorFile, schemas map[string]*model.Schema) error {
	resolved := make(map[string]bool)
	fetching := make(map[string]bool)

	var resolve func(name string) error
	resolve = func(name string) error {
		if resolved[name] {
			return nil
		}
		if fetching[name] {
			return fmt.Errorf("circular binding detected for schema %q", name)
		}
		fetching[name] = true

		s, ok := schemas[name]
		if !ok {
			return fmt.Errorf("schema %q not found during binding resolution", name)
		}

		if len(s.Binding) == 0 {
			resolved[name] = true
			fetching[name] = false
			return nil
		}

		localFields := s.Fields
		var finalFields []*model.Field
		seen := make(map[string]bool)

		for _, boundName := range s.Binding {
			if err := resolve(boundName); err != nil {
				return err
			}
			boundSchema := schemas[boundName]

			// Inherit fields from bound schema
			for _, f := range boundSchema.Fields {
				if !seen[f.Name] {
					finalFields = append(finalFields, f)
					seen[f.Name] = true
				}
			}

			// Inherit omit lists from bound schema for each language
			for lang := range mrr.Languages {
				if boundMeta, ok := boundSchema.Meta[lang]; ok {
					if boundBinding, ok := boundMeta["binding"].(map[string]any); ok {
						if boundOmit, ok := boundBinding["omit"].([]any); ok {
							// Ensure current schema has necessary meta structure
							if s.Meta == nil {
								s.Meta = make(map[string]map[string]any)
							}
							if s.Meta[lang] == nil {
								s.Meta[lang] = make(map[string]any)
							}
							if s.Meta[lang]["binding"] == nil {
								s.Meta[lang]["binding"] = make(map[string]any)
							}

							currentBinding := s.Meta[lang]["binding"].(map[string]any)
							if currentBinding["omit"] == nil {
								currentBinding["omit"] = []any{}
							}

							currentOmit := currentBinding["omit"].([]any)
							omitMap := make(map[string]bool)
							for _, o := range currentOmit {
								if name, ok := o.(string); ok {
									omitMap[name] = true
								}
							}

							for _, o := range boundOmit {
								if name, ok := o.(string); ok {
									if !omitMap[name] {
										currentOmit = append(currentOmit, name)
										omitMap[name] = true
									}
								}
							}
							currentBinding["omit"] = currentOmit
							s.Meta[lang]["binding"] = currentBinding
						}
					}
				}
			}
		}

		// Overwrite bound fields with local fields if they have the same name
		for i, f := range finalFields {
			for _, lf := range localFields {
				if lf.Name == f.Name {
					finalFields[i] = lf
					break
				}
			}
		}

		// Add new local fields
		for _, lf := range localFields {
			localSeen := false
			for _, f := range finalFields {
				if f.Name == lf.Name {
					localSeen = true
					break
				}
			}
			if !localSeen {
				finalFields = append(finalFields, lf)
			}
		}

		s.Fields = finalFields
		fetching[name] = false
		resolved[name] = true
		return nil
	}

	for name := range schemas {
		if err := resolve(name); err != nil {
			return err
		}
	}
	return nil
}
