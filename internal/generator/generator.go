package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/BladiCreator/mirror/internal/languages"
	"github.com/BladiCreator/mirror/internal/model"
)

// Result summarizes generation output.
type Result struct {
	WrittenFiles []string
	Errors       []error
}

// Generate runs plugin generation and writes files based on target languages.
func Generate(mrr *model.MirrorFile, reg *languages.Registry, baseOutput string, verbose bool) (*Result, error) {
	res := &Result{}
	var allSchemas []*model.Schema

	// Convert schemas map to list for generators
	for _, s := range mrr.Schemas {
		allSchemas = append(allSchemas, s)
	}

	for langName, config := range mrr.Languages {
		outputPaths := config.ResolvePaths(baseOutput)
		if len(outputPaths) == 0 {
			verbosePrintln(verbose, "[verbose] No output paths for %s, skipping\n", langName)
			continue
		}

		plg, ok := reg.Get(langName)
		if !ok {
			return res, fmt.Errorf("generator for language %q not found", langName)
		}
		verbosePrintln(verbose, "[verbose] Plugin used: %s\n", reflect.TypeOf(plg).Elem().Name())

		if config.Template != "" && !filepath.IsAbs(config.Template) {
			config.Template = filepath.Join(baseOutput, config.Template)
		}

		for _, outputDir := range outputPaths {
			if !filepath.IsAbs(outputDir) {
				outputDir, _ = filepath.Abs(outputDir)
			}

			cfg := model.OutputConfig{
				Language: langName,
				Filepath: outputDir,
				Suffix:   config.GetSuffix(),
				Format:   config.GetFormat(),
				Template: config.Template,
				Plugins:  mrr.Plugins,
			}
			verbosePrintln(verbose, "[verbose] Config for area %s: %+v\n", outputDir, cfg)

			files, err := plg.Generate(allSchemas, cfg)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("%s (%s): %w", langName, outputDir, err))
				continue
			}
			verbosePrintln(verbose, "[verbose] Generated %d files for %s in %s\n", len(files), langName, outputDir)

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
