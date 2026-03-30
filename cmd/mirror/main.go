package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/BladiCreator/mirror/internal/generator"
	"github.com/BladiCreator/mirror/internal/languages"
	"github.com/BladiCreator/mirror/internal/parser"
	"github.com/BladiCreator/mirror/internal/watcher"
)

type Command struct {
	Names   []string
	Summary string
	Action  func(args []string)
}

var commandRegistry = make(map[string]Command)
var commandOrder [][]string

func registerCommand(summary string, action func(args []string), names ...string) {
	cmd := Command{Names: names, Summary: summary, Action: action}
	for _, name := range names {
		commandRegistry[name] = cmd
	}
	commandOrder = append(commandOrder, names)
}

func init() {
	registerCommand("Initialize a new mirror.yml by analyzing existing models", runInit, "init")
	registerCommand("List available plugins or internal languages", runLs, "ls", "list-languages", "list-lang", "list-plugins")
	registerCommand("Generate code from a mirror.yml file (default command)", runGenerate, "generate")
	registerCommand("Show the default template used for a specific language", runShowTemplate, "show-template", "st")
	registerCommand("Manage language plugins (install, uninstall, etc.)", runLangManagement, "lang", "language")
	// registerCommand("Login to the Mirror registry", func(args []string) { fmt.Println("login: Not fully implemented in this version.") }, "login")
	registerCommand("Show this help message", func(args []string) { printHelp() }, "help", "--help", "-h")
}

func main() {
	if len(os.Args) < 2 {
		runGenerate([]string{}) // default to generate
		return
	}

	cmdName := os.Args[1]
	if cmd, ok := commandRegistry[cmdName]; ok {
		cmd.Action(os.Args[2:])
		return
	}

	// Default behavior for .yml files or flags
	if strings.HasSuffix(cmdName, ".yml") || strings.HasSuffix(cmdName, ".yaml") ||
		cmdName == "--watch" || cmdName == "--verbose" {
		runGenerate(os.Args[1:])
		return
	}

	fmt.Printf("Unknown command: %s\n", cmdName)
	printHelp()
	os.Exit(1)
}

func printHelp() {
	fmt.Println("Mirror - A universal code generator for models")
	fmt.Println("\nUsage:")
	fmt.Println("  mirror [command] [options]")
	fmt.Println("\nAvailable Commands:")
	for _, names := range commandOrder {
		cmd := commandRegistry[names[0]]
		fmt.Printf("  %-25s %s\n", strings.Join(names, ", "), cmd.Summary)
	}
	fmt.Println("\nGeneration Options:")
	fmt.Println("  --watch              Monitor mirror.yml and included files for changes")
	fmt.Println("  --verbose            Show detailed generation logs")
	fmt.Println("  --lang-dir <path>    Specify a directory to search for external plugins")
	fmt.Println("\nExamples:")
	fmt.Println("  mirror init")
	fmt.Println("  mirror generate mirror.yml --watch")
	fmt.Println("  mirror mirror.yml")
}

func runInit(args []string) {
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
	watchMode := fs.Bool("watch", false, "watch .yml files and regenerate")
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

func runLs(args []string) {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	langDir := fs.String("lang-dir", "", "directory to find external generators")
	fs.Parse(args)

	positional := fs.Args()
	subCmd := ""
	if len(positional) > 0 {
		subCmd = positional[0]
	}

	reg := languages.NewRegistry(*langDir)

	switch subCmd {
	case "plugin", "plugins", "languages", "lang":
		printAll(reg)
	default:
		printAll(reg)
		fmt.Println("\n(use 'mirror ls plugin' to see this list again)")
	}
}

func printAll(reg *languages.Registry) {
	fmt.Println("Available languages (internal):")
	langs := reg.ListInternal()
	sort.Strings(langs)
	for _, l := range langs {
		fmt.Println(" ", l)
	}
	fmt.Println("\nAvailable plugins (external):")
	plugins := reg.ListExternal()
	if len(plugins) == 0 {
		fmt.Println("  (none found)")
	} else {
		for _, p := range plugins {
			fmt.Println(" ", p)
		}
	}
}

func runShowTemplate(args []string) {
	fs := flag.NewFlagSet("show-template", flag.ExitOnError)
	langDir := fs.String("lang-dir", "", "directory to find external generators")
	fs.Parse(args)

	positional := fs.Args()
	if len(positional) == 0 {
		fmt.Println("usage: mirror show-template <lang>")
		os.Exit(1)
	}
	langName := positional[0]

	reg := languages.NewRegistry(*langDir)
	l, ok := reg.Get(langName)
	if !ok {
		fmt.Printf("Language/Plugin %s not found\n", langName)
		os.Exit(1)
	}

	tmpl, err := l.Template()
	if err != nil {
		fmt.Printf("Error getting template for %s: %v\n", langName, err)
		os.Exit(1)
	}

	fmt.Printf("Showing default template for %s:\n", langName)
	fmt.Println(tmpl)
}

func runLangManagement(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: mirror lang <install|uninstall|update|list|create|upload>")
		os.Exit(1)
	}
	subCmd := args[0]
	fmt.Printf("mirror lang %s: Not fully implemented in this version.\n", subCmd)
}
