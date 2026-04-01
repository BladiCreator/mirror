# Mirror CLI

**Mirror** is a model-based code generation system. It allows you to define data models in a YAML configuration (`mirror.yml`) and generate code in multiple languages (Dart, Go, TypeScript, etc.) using templates powered by Go's `text/template` engine.

## Key Features
- **Interactive Initialization**: `mirror init` scans your codebase to automatically detect and extract models.
- **Dynamic Command Registry**: A modern CLI with built-in help and aliases.
- **Multi-language Support**: Built-in support for Go and Dart, with easy extensibility for other languages.
- **Powerful Templates**: Standard Go templates extended with custom plugin functions.
- **Modular Plugin System**: Support for internal and external plugins for both languages and utility functions.
- **Recursive Schema Imports**: Split your models into multiple files using `include`.
- **Intelligent Imports**: Automate language-specific imports (e.g., `package:`, `import "..."`).

## Installation

```sh
go install github.com/BladiCreator/mirror/cmd/main@latest
```

## Quick Start

### 1. Initialize a project
Run `mirror init` in your project root. It will scan for existing structures and help you create a `mirror.yml`.

The `init` command supports flags:

```sh
mirror init --directory . --pattern "*_model.go" --languages "go,dart" --include-paths
```

Flags:
- `--directory` (default: "."): Directory to scan for models.
- `--pattern` (default: ""): File pattern to match (e.g., *_model.go). If empty, scans all supported files.
- `--languages` (default: ""): Comma-separated list of languages. If empty, detects the predominant language.
- `--include-paths` (default: true): Include file paths in schema metadata.
- `--split` (default: false): Create a `mirror/` directory and save schemas in separate files using `include` statements.

Examples:
- `mirror init --help`: Show help for the init command.
- `mirror init`: Interactive mode (legacy).
- `mirror init --pattern "*_model.go"`: Scan for Go model files, detect language.
- `mirror init --directory src --languages "go,dart"`: Scan src directory for all files, generate for Go and Dart.
- `mirror init --split`: Create mirror/ directory with separate schema files.

### 2. Configure your models
Define your schemas and targets in `mirror.yml`:

```yaml
languages:
  - go:
      output:
        filepath: "./internal/models"
        format: pascal
  - dart:
      output:
        filepath: "./lib/models"
        format: snake

schemas:
  - name: User
    fields:
      - name: id
        type: int
      - name: email
        type: string
```

### 3. Generate code
Run `mirror` to generate the code files.

```sh
mirror
```

## CLI Commands

- `mirror [generate] [file.yml]`: Generate code (default).
- `mirror init`: Start the interactive setup and model discovery.
- `mirror ls [plugins|lang]`: List available language generators and function plugins.
- `mirror [st|show-template] <lang>`: Show the default template for a specific language.
- `mirror [h|help|--help]`: Show detailed command usage.

## Advanced Usage

### Custom Filepaths per Schema
You can override the output directory for specific schemas using `meta`:

```yaml
schemas:
  - name: AuthUser
    meta:
      go:
        filepath: "auth/" # Generates in ./internal/models/auth/
```

### Plugin Functions in Templates
Use namespaced functions in your templates:

```text
type {{ .Name }} struct {
{{ range .Fields }}
  {{ .Name }} {{ .Type }} `json:"{{ .Name | fn:strings:toSnake }}"`
{{ end }}
}
```

### Imports Management
Automate dependencies and cross-references between schemas:

```yaml
schemas:
  - name: User
    import:
      go: ["fmt", "os"]
      dart: ["auto:profile"] # 'auto:' automatically resolves the path for schema 'profile'
```

## Documentation
For more detailed information, see the [Architecture and User Guide (document.md)](./document.md).

## License
MIT
