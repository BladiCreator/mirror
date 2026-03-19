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

	type yamlPathConfig struct {
		Plugin   []string `yaml:"plugin"`
		Filepath string   `yaml:"filepath"`
		Suffix   string   `yaml:"suffix"`
		Format   string   `yaml:"format"`
	}

	type yamlPathItem struct {
		Name   string         `yaml:"name"`
		Config yamlPathConfig `yaml:"config"`
	}

	type yamlSchemaItem struct {
		Name    string            `yaml:"name"`
		Fields  map[string]string `yaml:"fields"`
		Include []string          `yaml:"include"`
	}

	type yamlFile struct {
		Plugin  []string         `yaml:"plugin"`
		Paths   []yamlPathItem   `yaml:"paths"`
		Schemas []yamlSchemaItem `yaml:"schemas"`
	}

	var parsed yamlFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	mrr := &model.MRRFile{Filepath: path, Schemas: map[string]*model.Schema{}}
	for _, pluginName := range parsed.Plugin {
		mrr.Plugins = append(mrr.Plugins, normalizePluginName(pluginName))
	}

	for _, p := range parsed.Paths {
		if p.Name == "" {
			return nil, fmt.Errorf("path entry missing name in %s", path)
		}
		plugins := make([]string, 0, len(p.Config.Plugin))
		for _, pluginName := range p.Config.Plugin {
			plugins = append(plugins, normalizePluginName(pluginName))
		}
		entry := &model.PathEntry{
			Ext:       p.Name,
			Plugins:   plugins,
			OutputDir: p.Config.Filepath,
			Suffix:    p.Config.Suffix,
			Format:    p.Config.Format,
		}
		if entry.OutputDir == "" {
			entry.OutputDir = p.Name
		}
		mrr.Paths = append(mrr.Paths, entry)
	}

	for _, s := range parsed.Schemas {
		if s.Name != "" {
			if _, exists := mrr.Schemas[s.Name]; exists {
				return nil, fmt.Errorf("schema %q already defined in %s", s.Name, path)
			}
			schema := &model.Schema{Name: s.Name, Fields: []*model.Field{}}
			for fieldName, fieldType := range s.Fields {
				if strings.TrimSpace(fieldName) == "" || strings.TrimSpace(fieldType) == "" {
					return nil, fmt.Errorf("invalid field for schema %s in %s", s.Name, path)
				}
				schema.Fields = append(schema.Fields, &model.Field{Name: fieldName, Type: fieldType, Tags: map[string]string{}})
			}
			mrr.Schemas[s.Name] = schema
		}
		for _, include := range s.Include {
			importPath, err := normalizeImportPath(include, path)
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
		}
	}

	if !schemaOnly {
		if err := Validate(mrr); err != nil {
			return nil, err
		}
	}

	return mrr, nil
}

func normalizePluginName(name string) string {
	switch name {
	case "dart":
		return "dart_mrr_parser"
	case "go":
		return "go_mrr_parser"
	default:
		return name
	}
}
