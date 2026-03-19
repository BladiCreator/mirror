package model

import "path/filepath"

// MRRFile represents a parsed .mrr definition.
type MRRFile struct {
	Filepath string
	Plugins  []string
	Paths    []*PathEntry
	Schemas  map[string]*Schema
	Imports  []string
}

// PathEntry describes output options for a set of generated files.
type PathEntry struct {
	Ext       string
	Plugins   []string
	OutputDir string
	Suffix    string
	Format    string
}

// Schema represents a model schema with fields and tags.
type Schema struct {
	Name   string
	Tags   map[string]string
	Fields []*Field
}

// Field represents a field on a schema.
type Field struct {
	Name string
	Type string
	Tags map[string]string
}

// OutputConfig is the per-plugin generation config.
type OutputConfig struct {
	Path   string
	Suffix string
	Format string
}

// GeneratedFile is the code file produced by a plugin.
type GeneratedFile struct {
	Path    string
	Content string
}

// ResolveOutputPath returns the absolute output directory for a PathEntry.
func (p *PathEntry) ResolveOutputPath(baseDir string) string {
	if filepath.IsAbs(p.OutputDir) {
		return p.OutputDir
	}
	return filepath.Join(baseDir, p.OutputDir)
}
