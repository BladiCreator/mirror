package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BladiCreator/mirror/internal/model"
)

var validFormats = map[string]bool{"pascal": true, "snake": true, "camel": true, "kebab": true, "": true}

// ParseFile parses a .yml file and all its imports.
func ParseFile(path string) (*model.MirrorFile, error) {
	return parseFile(path, map[string]bool{}, false)
}

func parseFile(path string, visited map[string]bool, schemaOnly bool) (*model.MirrorFile, error) {
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
	case ".yml", ".yaml":
		return ParseYAMLFile(abs, visited, schemaOnly)
	default:
		return nil, fmt.Errorf("unsupported file extension %q for path %s (only .yml/.yaml allowed for config)", ext, abs)
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

// Validate performs semantic checks on the parsed config.
func Validate(mrr *model.MirrorFile) error {
	if len(mrr.Languages) == 0 {
		return errors.New("languages section is required and requires at least one language")
	}
	if len(mrr.Schemas) == 0 {
		return errors.New("schemas section is required and requires at least one schema")
	}

	for langName, config := range mrr.Languages {
		if !validFormats[config.GetFormat()] {
			return fmt.Errorf("invalid format %q in language %s", config.GetFormat(), langName)
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
			if strings.HasPrefix(f.Type, "list:") {
				ref := strings.TrimPrefix(f.Type, "list:")
				if _, ok := mrr.Schemas[ref]; !ok && !isPrimitive(ref) {
					// list of unknown non-primitive
				}
			}
		}
	}
	return nil
}

func isPrimitive(t string) bool {
	switch t {
	case "string", "int", "float", "bool":
		return true
	}
	return false
}
