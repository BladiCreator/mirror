package model

import "path/filepath"

// MRRFile represents a parsed configuration file (.yml).
type MRRFile struct {
	Filepath  string
	Languages map[string]LanguageConfig `json:"languages"`
	Schemas   map[string]*Schema        `json:"schemas"`
	Imports   []string                  `json:"imports"`
	Plugins   []string                  `json:"plugin"`
}

// LanguageConfig describes output options for a set of generated files.
type LanguageConfig struct {
	Filepath string `json:"filepath" yaml:"filepath"`
	Suffix   string `json:"suffix" yaml:"suffix"`
	Format   string `json:"format" yaml:"format"`
	Template string `json:"template" yaml:"template"`
}

// Schema represents a model schema with fields and tags.
type Schema struct {
	Name   string                    `json:"name" yaml:"name"`
	Meta   map[string]map[string]any `json:"meta,omitempty" yaml:"meta,omitempty"`
	Fields []*Field                  `json:"fields" yaml:"fields"`
	Import *ImportConfig             `json:"import,omitempty" yaml:"import,omitempty"`
}

// ImportConfig manages code imports for generated files.
type ImportConfig struct {
	Disable bool                `json:"disable" yaml:"disable"`
	Langs   map[string][]string `json:"langs" yaml:"langs"`
}

// Field represents a field on a schema.
type Field struct {
	Name string                    `json:"name" yaml:"name"`
	Type string                    `json:"type" yaml:"type"`
	Meta map[string]map[string]any `json:"meta,omitempty" yaml:"meta,omitempty"`
}

// OutputConfig is the per-plugin generation config (passed to generators).
type OutputConfig struct {
	Filepath string   `json:"filepath"`
	Suffix   string   `json:"suffix"`
	Format   string   `json:"format"`
	Template string   `json:"template"`
	Plugins  []string `json:"plugin"`
}


// GeneratedFile is the code file produced by a plugin.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ResolveOutputPath returns the absolute output directory for a given config.
func (c *LanguageConfig) ResolveOutputPath(baseDir string) string {
	if filepath.IsAbs(c.Filepath) {
		return c.Filepath
	}
	return filepath.Join(baseDir, c.Filepath)
}
