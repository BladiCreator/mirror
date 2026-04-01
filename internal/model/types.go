package model

import "path/filepath"

// MirrorFile represents a parsed configuration file (.yml).
type MirrorFile struct {
	Filepath  string
	Languages map[string]LanguageConfig `json:"languages"`
	Schemas   map[string]*Schema        `json:"schemas"`
	Imports   []string                  `json:"imports"`
	Plugins   []string                  `json:"plugin"`
}

// LanguageConfig describes output options for a set of generated files.
type LanguageConfig struct {
	Template string          `json:"template" yaml:"template"`
	Output   *OutputSettings `json:"output" yaml:"output"`
}

// OutputSettings holds nested configuration for file generation.
type OutputSettings struct {
	Filepath any    `json:"filepath" yaml:"filepath"` // string or []string
	Suffix   string `json:"suffix" yaml:"suffix"`
	Format   string `json:"format" yaml:"format"`
}

// GetFilepaths returns a slice of all output paths configured for the language.
func (c *LanguageConfig) GetFilepaths() []string {
	var paths []string

	if c.Output == nil || c.Output.Filepath == nil {
		return paths
	}

	switch v := c.Output.Filepath.(type) {
	case string:
		if v != "" {
			paths = append(paths, v)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				paths = append(paths, s)
			}
		}
	case []string:
		for _, s := range v {
			if s != "" {
				paths = append(paths, s)
			}
		}
	}

	return paths
}

// GetSuffix returns the suffix from the Output settings.
func (c *LanguageConfig) GetSuffix() string {
	if c.Output != nil {
		return c.Output.Suffix
	}
	return ""
}

// GetFormat returns the format from the Output settings.
func (c *LanguageConfig) GetFormat() string {
	if c.Output != nil {
		return c.Output.Format
	}
	return ""
}

// Schema represents a model schema with fields and tags.
type Schema struct {
	Name    string                    `json:"name" yaml:"name"`
	Meta    map[string]map[string]any `json:"meta,omitempty" yaml:"meta,omitempty"`
	Fields  []*Field                  `json:"fields" yaml:"fields"`
	Binding []string                  `json:"binding,omitempty" yaml:"binding,omitempty"`
	Import  *ImportConfig             `json:"import,omitempty" yaml:"import,omitempty"`
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
	Language string   `json:"language"`
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

// ResolvePaths returns a list of absolute output directories for a given config.
func (c *LanguageConfig) ResolvePaths(baseDir string) []string {
	var res []string
	for _, p := range c.GetFilepaths() {
		if filepath.IsAbs(p) {
			res = append(res, p)
		} else {
			res = append(res, filepath.Join(baseDir, p))
		}
	}
	return res
}

// ResolveOutputPath returns the first absolute output directory for a given config (legacy support).
func (c *LanguageConfig) ResolveOutputPath(baseDir string) string {
	paths := c.ResolvePaths(baseDir)
	if len(paths) == 0 {
		return baseDir
	}
	return paths[0]
}
