package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mirror/mirror/internal/model"
	"github.com/mirror/mirror/internal/plugin"
)

// Result summarizes generation output.
type Result struct {
	WrittenFiles []string
	Errors       []error
}

// Generate runs plugin generation and writes files.
func Generate(mrr *model.MRRFile, reg *plugin.Registry, baseOutput string, verbose bool) (*Result, error) {
	res := &Result{}
	var allSchemas []*model.Schema
	for _, s := range mrr.Schemas {
		allSchemas = append(allSchemas, s)
	}

	for _, p := range mrr.Paths {
		outputDir := p.ResolveOutputPath(baseOutput)
		if !filepath.IsAbs(outputDir) {
			outputDir, _ = filepath.Abs(outputDir)
		}
		for _, pluginName := range p.Plugins {
			plg, ok := reg.Get(pluginName)
			if !ok {
				return res, fmt.Errorf("plugin %q not found", pluginName)
			}
			verbosePrintln(verbose, "[verbose] Plugin used: %s\n", reflect.TypeOf(plg).Elem().Name())

			cfg := model.OutputConfig{Path: outputDir, Suffix: p.Suffix, Format: p.Format}
			verbosePrintln(verbose, "[verbose] Config %s\n", cfg)
			files, err := plg.Generate(allSchemas, cfg)
			if err != nil {
				res.Errors = append(res.Errors, err)
				continue
			}
			verbosePrintln(verbose, "[verbose] Generated %d files\n", len(files))

			for _, file := range files {
				target := filepath.Join(outputDir, file.Path)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					res.Errors = append(res.Errors, err)
					continue
				}
				if err := os.WriteFile(target, []byte(file.Content), 0644); err != nil {
					res.Errors = append(res.Errors, err)
					continue
				}
				res.WrittenFiles = append(res.WrittenFiles, target)
				verbosePrintln(verbose, "[verbose] Wrote %s\n", target)
			}
		}
	}

	if len(res.Errors) > 0 {
		return res, fmt.Errorf("generation completed with %d errors", len(res.Errors))
	}
	return res, nil
}

func verbosePrintln(verbose bool, format string, args ...any) {
	if verbose {
		fmt.Printf(format, args...)
	}
}
