package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mirror/mirror/internal/model"
)

var validFormats = map[string]bool{"pascal": true, "snake": true, "camel": true, "kebab": true, "": true}

// ParseFile parses a .mrr file and all its imports.
func ParseFile(path string) (*model.MRRFile, error) {
	return parseFile(path, map[string]bool{}, false)
}

func parseFile(path string, visited map[string]bool, schemaOnly bool) (*model.MRRFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[abs] {
		return nil, fmt.Errorf("circular import detected: %s", abs)
	}
	visited[abs] = true

	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".mrr":
		return parseMRRFile(abs, visited, schemaOnly)
	case ".yml", ".yaml":
		return parseYAMLFile(abs, visited, schemaOnly)
	default:
		return nil, fmt.Errorf("unsupported file extension %q for path %s", ext, abs)
	}
}

func normalizeImportPath(item, parent string) (string, error) {
	trim := strings.TrimSpace(item)
	if strings.HasPrefix(trim, "'") && strings.HasSuffix(trim, "'") {
		trim = strings.Trim(trim, "'")
	}
	if !filepath.IsAbs(trim) {
		return filepath.Join(filepath.Dir(parent), trim), nil
	}
	return trim, nil
}

// Validate performs semantic checks.
func Validate(mrr *model.MRRFile) error {
	if len(mrr.Plugins) == 0 {
		return errors.New("plugin section is required and requires at least one plugin")
	}
	if len(mrr.Paths) == 0 {
		return errors.New("paths section is required and requires at least one path")
	}
	if len(mrr.Schemas) == 0 {
		return errors.New("schemas section is required and requires at least one schema")
	}

	pluginsMap := map[string]bool{}
	for _, p := range mrr.Plugins {
		pluginsMap[p] = true
	}

	for _, pathEntry := range mrr.Paths {
		if len(pathEntry.Plugins) == 0 {
			return fmt.Errorf("path entry extension %q has no plugins", pathEntry.Ext)
		}
		for _, pn := range pathEntry.Plugins {
			if !pluginsMap[pn] {
				return fmt.Errorf("path entry extension %q references plugin that is not declared: %s", pathEntry.Ext, pn)
			}
		}
		if !validFormats[pathEntry.Format] {
			return fmt.Errorf("invalid format %q in path %s", pathEntry.Format, pathEntry.Ext)
		}
	}

	for _, s := range mrr.Schemas {
		if s.Name == "" {
			return errors.New("schema with empty name")
		}
		for _, f := range s.Fields {
			if f.Name == "" || f.Type == "" {
				return fmt.Errorf("schema %s has invalid field", s.Name)
			}
			if strings.HasPrefix(f.Type, "object:") {
				ref := strings.TrimPrefix(f.Type, "object:")
				if _, ok := mrr.Schemas[ref]; !ok {
					return fmt.Errorf("schema %q references unknown object type %q", s.Name, ref)
				}
			}
		}
	}

	return nil
}
