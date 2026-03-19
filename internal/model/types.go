package model

import "path/filepath"

// MRRFile represents a parsed .mrr definition.
type MRRFile struct {
	Filepath string             `json:"filepath"`
	Plugins  []string           `json:"plugins"`
	Paths    []*PathEntry       `json:"paths"`
	Schemas  map[string]*Schema `json:"schemas"`
	Imports  []string           `json:"imports"`
}

// PathEntry describes output options for a set of generated files.
type PathEntry struct {
	Ext       string   `json:"ext"`
	Plugins   []string `json:"plugins"`
	OutputDir string   `json:"output_dir"`
	Suffix    string   `json:"suffix"`
	Format    string   `json:"format"`
}

// Schema represents a model schema with fields and tags.
type Schema struct {
	Name   string            `json:"name"`
	Tags   map[string]string `json:"tags"`
	Fields []*Field          `json:"fields"`
}

// Field represents a field on a schema.
type Field struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	Tags map[string]string `json:"tags"`
}

// OutputConfig is the per-plugin generation config.
type OutputConfig struct {
	Path   string `json:"path"`
	Suffix string `json:"suffix"`
	Format string `json:"format"`
}

// GeneratedFile is the code file produced by a plugin.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ResolveOutputPath returns the absolute output directory for a PathEntry.
func (p *PathEntry) ResolveOutputPath(baseDir string) string {
	if filepath.IsAbs(p.OutputDir) {
		return p.OutputDir
	}
	return filepath.Join(baseDir, p.OutputDir)
}
