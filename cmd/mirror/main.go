package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mirror/mirror/internal/generator"
	"github.com/mirror/mirror/internal/parser"
	"github.com/mirror/mirror/internal/plugin"
	"github.com/mirror/mirror/internal/watcher"
)

func main() {
	watchMode := flag.Bool("watch", false, "watch .mrr files and regenerate")
	verbose := flag.Bool("verbose", false, "verbose output")
	pluginsDir := flag.String("plugins-dir", "", "directory to find external plugins")
	outputDir := flag.String("output-dir", "", "base output directory for generated files")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: mirror [options] <archivo.mrr|archivo.yml>")
		os.Exit(1)
	}

	mrrPath := args[0]
	if !filepath.IsAbs(mrrPath) {
		mrrPath = filepath.Clean(mrrPath)
	}

	base := *outputDir
	if base == "" {
		base = filepath.Dir(mrrPath)
	}

	if err := runOnce(mrrPath, base, *pluginsDir, *verbose); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if *watchMode {
		stopCh := make(chan struct{})
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			close(stopCh)
		}()

		paths := []string{mrrPath}
		file, _ := parser.ParseFile(mrrPath)
		paths = append(paths, file.Imports...)

		fmt.Println("watch mode enabled, monitoring files:", strings.Join(paths, ", "))
		if err := watcher.Watch(paths, func() error {
			return runOnce(mrrPath, base, *pluginsDir, *verbose)
		}, stopCh); err != nil {
			fmt.Println("watcher error:", err)
			os.Exit(1)
		}
	}
}

func runOnce(mrrPath, baseOutput, pluginsDir string, verbose bool) error {
	mrr, err := parser.ParseFile(mrrPath)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println("[verbose] Parsed config:")
		fmt.Println(" [verbose] File:", mrrPath)
		fmt.Println(" [verbose] Plugins:", strings.Join(mrr.Plugins, ", "))
		fmt.Println(" [verbose] Paths:")
		for _, p := range mrr.Paths {
			// fmt.Printf(" [verbose]   - ext=%q, output=%q, suffix=%q, format=%q, plugins=%v\n", p.Ext, p.OutputDir, p.Suffix, p.Format, p.Plugins)
			fmt.Printf(" [verbose]   - %s\n", p)
		}
		fmt.Println(" [verbose] Schemas:")
		for _, s := range mrr.Schemas {
			fmt.Printf(" [verbose]   - %s\n", s.Name)
			for _, f := range s.Fields {
				fmt.Printf(" [verbose]       * %s: %s\n", f.Name, f.Type)
			}
		}
		fmt.Println(" [verbose] Imports:", strings.Join(mrr.Imports, ", "))
	}

	if baseOutput == "" {
		baseOutput = filepath.Dir(mrrPath)
	}

	reg := plugin.NewRegistry(pluginsDir)
	for _, p := range mrr.Plugins {
		if _, ok := reg.Get(p); !ok {
			return fmt.Errorf("plugin %s not found", p)
		}
	}

	result, err := generator.Generate(mrr, reg, baseOutput, verbose)
	if err != nil {
		fmt.Printf("generation failed: %v\n", err)
		for _, e := range result.Errors {
			fmt.Println(" -", e)
		}
		return err
	}

	fmt.Printf("generated %d files\n", len(result.WrittenFiles))
	for _, f := range result.WrittenFiles {
		fmt.Println("  ", f)
	}
	return nil
}
