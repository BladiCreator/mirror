package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mirror/mirror/internal/generator"
	"github.com/mirror/mirror/internal/languages"
	"github.com/mirror/mirror/internal/parser"
	"github.com/mirror/mirror/internal/watcher"
)

func main() {
	if len(os.Args) < 2 {
		runGenerate([]string{}) // default to generate
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit()
	case "generate":
		runGenerate(os.Args[2:])
	case "list-languages", "list-lang":
		runListLang()
	case "show-template":
		runShowTemplate(os.Args[2:])
	case "lang", "language":
		runLangManagement(os.Args[2:])
	case "login":
		fmt.Println("login: Not fully implemented in this version.")
	default:
		// if it's a file, assume default command is generate
		if cmd == "--init" {
			runInit()
			return
		}
		if strings.HasSuffix(cmd, ".yml") || strings.HasSuffix(cmd, ".yaml") || cmd == "--watch" || cmd == "--verbose" {
			runGenerate(os.Args[1:])
		} else {
			fmt.Printf("Unknown command: %s\n", cmd)
			os.Exit(1)
		}
	}
}

func runInit() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Initializing mirror project...")
	
	// Registry provides analyzers
	reg := languages.NewRegistry("")
	analyzers := reg.Analyzers()
	
	fmt.Println("Which directory should I analyze to find your models? (default: '.')")
	var scanDir string
	if scanner.Scan() {
		scanDir = scanner.Text()
	}
	if scanDir == "" {
		scanDir = "."
	}
	
	absScanDir, _ := filepath.Abs(scanDir)
	detected, err := parser.DetectPredominantLanguage(absScanDir, analyzers)
	if err != nil {
		fmt.Println("Error detecting predominant language:", err)
		os.Exit(1)
	}
	fmt.Printf("Predominant language detected: %s\n", detected)
	
	var availableLangs []string
	for l := range analyzers {
		availableLangs = append(availableLangs, l)
	}

	fmt.Printf("\nAvailable internal languages: %s\n", strings.Join(availableLangs, ", "))
	fmt.Printf("Detected: [%s]\n", detected)
	fmt.Println("Enter languages to add (comma-separated) or just press Enter to use detected:")

	var input string
	if scanner.Scan() {
		input = scanner.Text()
	}

	selected := []string{detected}
	if input != "" {
		selected = []string{}
		parts := strings.Split(input, ",")
		for _, p := range parts {
			selected = append(selected, strings.TrimSpace(p))
		}
	}

	fmt.Println("Extracting schemas...")
	schemas, err := parser.ExtractSchemas(detected, absScanDir, analyzers)
	if err != nil {
		fmt.Println("Error extracting schemas:", err)
		os.Exit(1)
	}

	mrr, err := parser.InitialSetup(detected, schemas, selected)
	if err != nil {
		fmt.Println("Error setup:", err)
		os.Exit(1)
	}

	// Write mirror.yml
	// data, _ := os.Marshal(mrr) // This line was commented out in the provided snippet
	// Marshal is not YAML. Wait. I should use YAML.
	// Actually, parser.parseYAMLFile uses gopkg.in/yaml.v3

	// Create a simple YAML manually for better control of comments/layout
	var sb strings.Builder
	sb.WriteString("languages:\n")
	for lang, cfg := range mrr.Languages {
		sb.WriteString(fmt.Sprintf("  - %s:\n", lang))
		sb.WriteString(fmt.Sprintf("      filepath: '%s'\n", cfg.Filepath))
		sb.WriteString(fmt.Sprintf("      format: %s\n", cfg.Format))
	}
	sb.WriteString("\nschemas:\n")
	for name, s := range mrr.Schemas {
		sb.WriteString(fmt.Sprintf("  - name: %s\n", name))
		sb.WriteString("    fields:\n")
		for _, f := range s.Fields {
			sb.WriteString(fmt.Sprintf("      - name: %s\n", f.Name))
			sb.WriteString(fmt.Sprintf("        type: %s\n", f.Type))
		}
	}

	if err := os.WriteFile("mirror.yml", []byte(sb.String()), 0644); err != nil {
		fmt.Println("Error writing mirror.yml:", err)
		os.Exit(1)
	}

	fmt.Println("\nSuccessfully created mirror.yml with extracted schemas.")
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	watchMode := fs.Bool("watch", false, "watch .yml and .mrr files and regenerate")
	verbose := fs.Bool("verbose", false, "verbose output")
	langDir := fs.String("lang-dir", "", "directory to find external generators")
	fs.Parse(args)

	positional := fs.Args()
	configFile := "mirror.yml"
	if len(positional) > 0 {
		configFile = positional[0]
	}

	if !filepath.IsAbs(configFile) {
		configFile = filepath.Clean(configFile)
	}

	baseOutput := filepath.Dir(configFile)

	if err := doGenerate(configFile, baseOutput, *langDir, *verbose); err != nil {
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

		paths := []string{configFile}
		file, _ := parser.ParseFile(configFile)
		if file != nil {
			paths = append(paths, file.Imports...)
		}

		fmt.Println("watch mode enabled, monitoring files:", strings.Join(paths, ", "))
		if err := watcher.Watch(paths, func() error {
			return doGenerate(configFile, baseOutput, *langDir, *verbose)
		}, stopCh); err != nil {
			fmt.Println("watcher error:", err)
			os.Exit(1)
		}
	}
}

func doGenerate(mrrPath, baseOutput, langDir string, verbose bool) error {
	mrr, err := parser.ParseFile(mrrPath)
	if err != nil {
		return err
	}

	reg := languages.NewRegistry(langDir)

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

func runListLang() {
	// Normally we would list all items in the internal registry and the ~/.mirror/languages
	fmt.Println("Available languages (internal):")
	fmt.Println("  dart")
	fmt.Println("  go")
	fmt.Println("(external plugins are dynamically resolved via PATH or --lang-dir)")
}

func runShowTemplate(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: mirror show-template <lang>")
		os.Exit(1)
	}
	lang := args[0]
	// In a complete implementation, this would query the plugin.
	// For internal plugins, we hardcode showing it.
	switch lang {
	case "dart":
		fmt.Println("Showing default dart template:")
		fmt.Println("class {{ formatName .Name \"pascal\" }} { ... }")
	case "go":
		fmt.Println("Showing default go template:")
		fmt.Println("type {{ formatName .Name \"pascal\" }} struct { ... }")
	default:
		fmt.Printf("Cannot show template for external/unknown language: %s\n", lang)
	}
}

func runLangManagement(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: mirror lang <install|uninstall|update|list|create|upload>")
		os.Exit(1)
	}
	subCmd := args[0]
	fmt.Printf("mirror lang %s: Not fully implemented in this version.\n", subCmd)
}
