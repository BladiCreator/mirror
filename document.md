## 1. Introduction

**MRR** (Mirror) is a model and template-based code generation system. It allows you to define data models in a YAML configuration file (`.yml`) and generate code in multiple languages (Dart, TypeScript, Go, etc.) using templates written with Go's `text/template` engine. The templates have the `.mrr` extension and are processed by the standard Go engine, which can be extended with custom functions through plugins.

This document describes the complete architecture: YAML configuration format, template language (`text/template`), language generators (internal and external), function plugins, workflow, and terminal usage.

## 2. Main Configuration File (`.yml`)

The configuration is written in a YAML file with three main sections: `plugin`, `languages` (or `lang`), and `schemas`.

### 2.1. General Structure

```yaml
plugin:
  - <plugin_name>[:<alias>]   # optional, an alias can be specified
languages:   # or "lang"
  - <language_name>:
      filepath: <output_path> # optional
      suffix: <optional_suffix> # optional
      format: <snake|pascal|camel> # optional
      template: <template_path.mrr>   # optional
schemas:
  - name: <schema_name>
    # Optional: language-specific metadata at the schema level
    meta:
      <language_name>:
        <key>: <value>
    fields:
      - name: <field_name>
        type: <type>
        # Optional: language-specific metadata at the field level
        meta:
          <language_name>:
            <key>: <value>
  - include: <path/to/another/file.yml>
```

#### `plugin` Section

Defines a list of plugins that will provide additional functions to the templates. Each element can be:

*   Plugin name only: `- dart_freeze`
    
*   Name with alias: `- dart_freeze:df` (the alias is used as a prefix in functions)
    

Plugins can be internal (built into the tool) or external (executables). Each plugin exposes a map of functions that are registered in the template engine with the format `fn:<alias>:<function_name>`.

Example:

```yaml
plugin:
  - dart_freeze:df      # internal plugin with alias 'df'
  - my-utils:mu         # external plugin (executable) with alias 'mu'
  - strings             # internal plugin without alias (the name is used as prefix)
```

#### `languages` (or `lang`) Section

A list of languages. Each element is a map with a single key which is the language name (e.g., `dart`, `go`, `ts`). The value is a map with the output configuration for that language:

*   `filepath` (required): path where generated files will be saved. It can be absolute or relative to the configuration file. If it ends in `/`, it is considered a directory; otherwise, a single file with that name is generated.
    
*   `suffix` (optional): suffix to be added to the base filename (only if `filepath` is a directory).
    
*   `format` (optional): filename format: `snake`, `pascal`, or `camel`. By default, the schema name is used as is.
    
*   `template` (optional): path to the `.mrr` template file that will be used to generate the code. If not specified, the internal generator uses its default (embedded) template. If the generator is external, it must have its own mechanism to use a default template or handle the omission.
    
Example:

```yaml
languages:
  - dart:
      filepath: './lib/models'
      suffix: '_dart'
      format: snake
  - go:
      filepath: './internal/models'
      format: pascal
      template: 'templates/go_struct.mrr'
  - ts:
      filepath: './src/models'
      template: 'templates/ts_interface.mrr'
```

#### `schemas` Section

Defines data models. It's a list that can contain:

*   **Inline schemas**: objects with keys `name` (schema name), `fields` (list of fields), and optionally `meta` (language-specific schema-level metadata).
    
*   **Inclusions**: objects with the `include` key and the path to another YAML file containing more schemas (imported recursively).
    

Each field has:

*   `name`: field name.
    
*   `type`: field type. It can be a primitive type (`string`, `int`, `float`, `bool`), a reference to another schema (`object:<name>`), a list (`list:<type>`), or a type with language-specific tags (e.g., `int go:float32`). Tags are written after the base type, separated by spaces. **Note**: It is recommended to use the new `meta` field for structured data, but inline tag syntax is maintained for compatibility in simple cases.
    
*   `meta` (optional): a map where each key is a language name (or `"common"` for all) and the value is another map with key-value pairs accessible from the template.
    
    Special keys exist in `meta` with predefined behavior:
    *   `filepath`: Allows specifying a sub-path for the schema, relative to the `filepath` defined in the `languages` section. Example: `meta: go: filepath: "auth/"`.
    
    This metadata can be any YAML type (string, number, boolean, list, map) and allows for extensible generation customization per field and language.
    

Example with `meta`:

```yaml
schemas:
  - name: user
    meta:
      dart:
        filePrefix: 'user_'
      go:
        package: 'models'
    fields:
      - name: id
        type: int
        meta:
          dart:
            jsonKey: 'id'
            ignore: false
          go:
            tag: 'json:"id"'
      - name: name
        type: string
        meta:
          dart:
            jsonKey: 'name'
          go:
            tag: 'json:"name" gorm:"column:name"'
      - name: email
        type: string
      - name: profile
        type: object:profile
        meta:
          dart:
            isLate: true
      - name: friends
        type: list:user
        meta:
          go:
            tag: 'json:"friends,omitempty"'
  - name: profile
    fields:
      - name: bio
        type: string
      - name: avatar
        type: string
  - include: 'common/address.yml'
```

### 2.2. Schema Binding (`binding`)

The `binding` property allows a schema to inherit all fields from one or more other schemas. This is extremely useful when multiple schemas share similar properties or when a specific language requires a unified object representing multiple related models.

#### Usage
Add the `binding` key followed by an array of schema names:

```yaml
schemas:
  - name: login
    fields:
      - name: email
        type: string
      - name: password
        type: string

  - name: signup
    binding: [login]
    fields:
      - name: username
        type: string
```

In this example, `signup` will effectively have three fields: `email`, `password`, and `username`.

#### Overriding Fields
If a schema defines a field with the same name as a field in a bound schema, the local definition takes precedence. This allows customizing types or metadata for specific fields while inheriting the rest.

#### Recursive Binding
Bindings are resolved recursively. If `Schema C` binds `Schema B`, and `Schema B` binds `Schema A`, then `Schema C` will inherit fields from both `A` and `B`.

#### Language-Specific Omission (`omit`)
You can omit certain inherited fields for a specific language using the `meta` configuration. This is done via `meta.<language>.binding.omit`.

Example:
```yaml
schemas:
  - name: signup
    binding: [login]
    meta:
      go:
        binding:
          omit: [password] # Omit password for Go structs
    fields:
      - name: username
        type: string
```

In the generated Go code for `signup`, the `password` field will be excluded, while in other languages it will still be present.

### 2.3. Import Management (`import`)

The `import` field allows managing external dependencies (packages) and internal references between schemas. It is a fundamental piece in languages like Dart (where classes usually reside in separate files) and allows enriching Go files with necessary imports automatically or manually.

#### Configuration Formats
Mirror supports three ways to define imports in a schema, offering configuration flexibility:

1.  **Boolean Format**: Allows globally disabling all imports for a schema.
    ```yaml
    import: false  # Disables all automatic and manual imports
    ```

2.  **Map Format (Recommended)**: Allows granular control per language.
    ```yaml
    import:
      disable: false   # Optional, defaults to false
      go: 
        - "fmt"
        - "os"
      dart: 
        - "package:flutter/material.dart"
        - "auto:profile"
    ```

3.  **List Format**: Allows grouping definitions from multiple languages sequentially.
    ```yaml
    import:
      - go: ["fmt", "os"]
      - dart: ["package:my_app/utils.dart"]
      - disable: true  # Can appear at any point in the list
    ```

#### Technical Operation and Automation
Mirror's logic (`internal/parser/yaml.go`) automates much of the work:

*   **Automatic Reference (`auto:`)**: When a schema defines a field of object type (`object:Name`), Mirror detects this dependency and automatically adds an `auto:Name` entry to the list of imports for all active languages.
    *   **Resolution**: When generating code, the language plugin translates `auto:Name` into the actual file path according to conventions (e.g., `name.dart` in snake_case or `Name.go` in PascalCase).
*   **`disable` Priority**: If `disable` is `true`, the generator will skip the imports section in the resulting file, regardless of how many automatic or manual dependencies exist.
*   **Structural Metadata**: Internally, this translates to the `ImportConfig` structure defined in `internal/model/types.go`:
    ```go
    type ImportConfig struct {
        Disable bool                // Activation state
        Langs   map[string][]string // Map of Language -> Dependency list (strings)
    }
    ```

### 2.4. Inclusions (`include`)

Allows splitting the configuration into multiple files. Paths are relative to the file performing the inclusion.

```yaml
schemas:
  - include: 'common/base_models.yml'
```

## 3. Template Files (`.mrr`) with `text/template`

Templates use standard Go `text/template` syntax. This provides a wide range of capabilities, including variables, conditionals, loops, custom functions, and pipelines. Additionally, plugins can add functions using the format `fn:<alias>:<name>`.

### 3.1. Basic Syntax

*   **Data Access**: `{{ .Name }}` – accesses the `Name` field of the current context.
    
*   **Conditionals**:
    
    ```text
    {{ if .IsActive }}...{{ else }}...{{ end }}
    ```

*   **Loops**:
    
    ```text
    {{ range .Fields }}
      {{ .Name }}: {{ .Type }}
    {{ end }}
    ```

*   **Variables**: `{{ $field := .Field }}`
    
*   **Functions**: `{{ .Name | toUpper }}`, `{{ eq .Type "string" }}`, etc.
    
*   **Plugin Functions**: `{{ fn:df:freeze .Name }}` (where `df` is the alias for the `dart_freeze` plugin and `freeze` is a function it exposes).
    
*   **Comments**: `{{/* comment */}}`
    

### 3.2. Data Context

Each template is executed with a context object containing all information for the current schema. This context includes:

*   **Schema Name**: `{{ .Name }}`
    
*   **Fields**: `{{ .Fields }}` – a list of structures with `Name`, `Type`, and `Meta` fields (the latter being nested maps).
    
*   **Schema Metadata**: `{{ .Meta.go.package }}` (if it exists).
    
*   **Per-field Metadata**: `{{ range .Fields }}{{ .Meta.dart.jsonKey }}{{ end }}`.
    

Metadata is passed as maps of `interface{}`. It can be accessed using successive dots, but it's recommended to verify its existence with `{{ if .Meta.dart }}...{{ end }}`.

### 3.3. Helper and Plugin Functions

The template engine registers a base set of functions, plus those provided by plugins. Base functions may include:

*   `toUpper`, `toLower`, `toPascal`, `toSnake`, `toCamel` – string transformations.
    
*   `type` – maps primitive types to specific language types (e.g., `int` → `int64` in Go).
    
*   `escape` – escapes special characters for the output language.
    
*   `eq`, `ne`, `lt`, `gt` – comparisons.
    

Plugin functions are registered in the format `fn:<alias>:<name>`. For example, if the `dart_freeze` plugin is defined with alias `df` and exposes a `freeze` function, it's used as `{{ fn:df:freeze .Name }}`.

### 3.4. Template Examples

**Default Dart Template** (embedded):

```text
class {{ .Name }} {
{{ range .Fields }}
  final {{ .Type }} {{ .Name }};
{{ end }}
}
```

**Custom Go Template** (`templates/go_struct.mrr`) using a plugin function:

```text
package {{ .Meta.go.package }}
type {{ .Name }} struct {
{{ range .Fields }}
  {{ .Name }} {{ .Type }} `{{ .Meta.go.tag | fn:mu:escapeJSON }}`
{{ end }}
}
```

**Template with Conditional and Plugin Function**:

```text
{{ if fn:df:shouldGenerate .Name }}
type {{ .Name }} struct {
  {{ range .Fields }}
    {{ .Name }} {{ .Type }} `json:"{{ .Name | toSnake }}"`
  {{ end }}
}
{{ end }}
```

### 3.5. Compilation and Execution

The template engine (integrated into the language generator) performs the following steps:

1.  **Register Base Functions**: Basic functions are added to the template.
    
2.  **Register Plugins**: For each plugin defined in the `plugin` section, its functions are loaded (either from internal code or by executing the external plugin to get the function map) and registered with the corresponding alias.
    
3.  **Parsing**: The `.mrr` file is read and the template is parsed using `template.ParseFiles` (or `template.New` + `Parse`).
    
4.  **Execution**: For each schema, the template is executed using the schema data as context, writing the output to a buffer or directly to a file.
    

This approach leverages the standard Go engine, which is extensible via functions.

## 4. Function Plugins

Plugins are components that provide additional functions to templates. They can be internal (built into the tool) or external (executables).

### 4.1. Internal Plugins

Internal plugins are compiled with the tool and registered directly. Each internal plugin has a name (the same used in configuration) and optionally an alias. Upon loading, it injects a map of functions (name → Go function) into the template engine.

Example of internal `dart_freeze` plugin providing a `freeze` function:

```go
var pluginFunctions = map[string]interface{}{
    "freeze": func(s string) string { return "frozen_" + s },
}
```

### 4.2. External Plugins

External plugins are independent executables that must follow a defined interface. When registered, the tool invokes them to obtain the map of functions they expose. The plugin must respond to a JSON request via stdin and return a function map (as JSON) via stdout.

#### JSON Interface (Function Request):

```json
{
  "command": "list_functions"
}
```

#### JSON Interface (Response):

```json
{
  "functions": {
    "freeze": "function signature description (optional)",
    "shouldGenerate": "description"
  }
}
```

The tool does not execute external plugin functions on every template invocation; instead, the plugin must implement a server (or evaluation mode) so the tool can send it function execution requests with arguments. Alternatively, the plugin can be executed every time a function is invoked (simple mode). The design can be flexible.

**Simple Mode**: For each use of `fn:alias:func` in the template, the tool executes the plugin with a JSON describing the function and its arguments, and the plugin returns the result.

**Server Mode**: The plugin remains running, and the tool communicates via a socket or persistent stdin/stdout.

For initial simplicity, simple mode can be chosen (executing the plugin each time a function is needed), although it is less efficient. For internal plugins, the call is direct.

### 4.3. Plugin Resolution

When a plugin is specified in the `plugin` section, the tool first checks if it's an internal plugin. If not, it attempts to execute a command with that name (e.g., `dart_freeze`), assuming it's in the PATH or in the plugins directory.

## 5. Language Generators

Generators are responsible for producing the files. They can be internal (using the `text/template` engine) or external (independent executables). Each generator is associated with a specific language.

### 5.1. Internal Generators

The tool includes internal generators for common languages (Dart, Go, TypeScript) that use `text/template`. These generators:

*   Have an embedded default template.
    
*   Receive schemas and output configuration.
    
*   If a `template` is specified, that external template is loaded; otherwise, the default template is used.
    
*   Compile the template with base functions and plugin functions.
    
*   Write files to the indicated paths.
    

### 5.2. External Generators

External generators are executables that must follow the interface defined above. They can implement their own generation logic without using `text/template`. They can also leverage plugins if provided with the appropriate information.

### 5.3. Interaction Between Plugins and Generators

When an internal generator is invoked, it already has access to all registered functions (base + plugins). For external generators, the list of available plugins can be passed via the input JSON, allowing them to execute plugin functions if desired (e.g., using an invocation mechanism similar to that of external plugins).

## 6. Application Flow

1.  **Read YAML File**: The configuration file is parsed (default `mirror.yml` or the one specified).
    
2.  **Resolve Imports**: All YAML files referenced in `include` within `schemas` are loaded recursively, merging their schemas.
    
3.  **Load Plugins**: Plugins defined in the `plugin` section are initialized, registering their functions (directly for internal ones, via interface for external ones).
    
4.  **Validation**:
    
    *   Verify that all listed languages exist as generators.
        
    *   Verify that output paths are accessible (or can be created).
        
    *   Verify that field types reference existing schemas.
        
    *   Verify that indicated templates (if any) are readable.
        
5.  **Context Preparation**: For each language, determine which schemas should be generated (all defined). Prepare the output configuration.
    
6.  **Invoke Generators**: For each language, execute the corresponding one (internal or external), passing schemas and configuration, as well as the list of available plugins (so external generators can use them).
    
    *   If internal, the template is compiled with registered functions and code is generated.
        
    *   If external, the process is launched and JSON is exchanged.
        
7.  **Write Files**: Returned files are written to the filesystem.
    
8.  **Report**: A summary of generated files or errors is displayed.
    

## 7. Error Handling

*   Generator not found: error and stop.
    
*   Plugin not found: error and stop (if listed in config but cannot be loaded).
    
*   Specified template not found or contains syntax errors: error indicating the line.
    
*   Variable not defined in template: error (captured at runtime).
    
*   Reference to non-existent schema: error.
    
*   Duplicate schema names: error.
    
*   Error in external generator (non-zero exit code): its stderr is shown.
    
*   Error in external plugin when listing functions or invoking them: error is shown.
    

## 8. Complete Example

`mirror.yml` file:

```yaml
plugin:
  - dart_freeze:df
  - strings
languages:
  - dart:
      filepath: './lib/models'
      suffix: '_dart'
      format: snake
  - go:
      filepath: './internal/models'
      format: pascal
      template: 'templates/go_struct.mrr'
schemas:
  - name: user
    meta:
      go:
        package: 'models'
    fields:
      - name: id
        type: int
        meta:
          dart:
            jsonKey: 'id'
          go:
            tag: 'json:"id"'
      - name: name
        type: string
        meta:
          dart:
            jsonKey: 'name'
          go:
            tag: 'json:"name"'
      - name: email
        type: string
      - name: friends
        type: list:user
        meta:
          go:
            tag: 'json:"friends,omitempty"'
  - include: 'common/address.yml'
```

`templates/go_struct.mrr` file:

```text
package {{ .Meta.go.package }}
type {{ .Name }} struct {
{{ range .Fields }}
  {{ .Name }} {{ .Type }} `{{ .Meta.go.tag | fn:strings:toUpper }}`
{{ end }}
}
```

In this example, the `strings` plugin (internal) provides a `toUpper` function. The template uses it to transform the tag to uppercase.

## 9. Tool Usage in Terminal

The tool is invoked with the `mirror` command. It supports several subcommands to generate code and manage generators and plugins.

### 9.1. Generation and Initialization

#### Initialization (`init` subcommand)

```bash
mirror init [options]
```

This command helps configure a new `mirror.yml` project by analyzing existing code files:

1.  **Recursive Scanning**: Searches for code files (Go, Dart, etc.) in the specified directory using optional patterns.
2.  **Model Extraction**: Uses language-specific analyzers to detect structures/classes and extract their fields automatically.
3.  **Configuration Generation**: Creates a `mirror.yml` file with detected schemas and language settings.

Options:

*   `--directory <dir>`: Directory to scan (default: current directory `.`).
*   `--pattern <pattern>`: File pattern to match (e.g., `*_model.go`, `**/*.dart`). Supports glob patterns with `**` for recursion. If not specified, scans all supported files.
*   `--languages <list>`: Comma-separated list of languages to generate for (e.g., `go,dart`). If not specified, detects the predominant language automatically.
*   `--include-paths`: Include source file paths in schema metadata (default: true).
*   `--split`: Create a `mirror/` directory and save schemas in `mirror/schemas.yml`, using `include` in the main file (default: false).
*   `--help`: Show help for the init command.

Examples:

```bash
mirror init                                    # With all default values
mirror init --help                             # Show help
mirror init --pattern "*_model.go"             # Scan for Go model files
mirror init --directory src --languages go,dart # Scan src/ for Go and Dart
mirror init --pattern "**/*.dart" --include-paths=false  # Scan Dart files without paths
mirror init --split                             # Create mirror/ with separate schema file```

#### Code Generation (`generate` subcommand)

```bash
mirror [generate] [file.yml]
```

*   `generate` is optional; if omitted, generation is assumed (if it's not another known command).
*   If no file is specified, it looks for `mirror.yml` in the current directory.

Options for generation:

*   `--watch`: monitors changes in `.yml` files and regenerates automatically.
*   `--verbose`: shows detailed information about the process (e.g., which plugin was used, files written).
*   `--lang-dir`: additional directory to search for external language generators (plugins).

Examples:

```bash
mirror                     # Generates using mirror.yml
mirror myconfig.yml        # Generates using myconfig.yml
mirror generate --watch    # Generates and watches for changes
```

### 9.2. Inspecting Languages, Templates, and Plugins

*   `mirror ls [languages|plugins]`: lists all available generators and plugins (internal and external). Recognizes aliases like `list-lang`, `list-plugins`, etc.
*   `mirror show-template <language>`: shows the default template that the specified language will use, obtaining it directly from the plugin. (Alias: `st`).
*   `mirror help`: shows dynamic help with all registered commands and aliases. (Alias: `--help`, `-h`).

Example:

```bash
mirror ls
```

Example:

```bash
mirror list-plugins
```

Expected output of `mirror ls`:

```text
Available languages (internal):
  dart
  go

Available plugins (external):
  (none found)
```

### 9.3. Managing External Generators and Plugins

The tool provides subcommands to install, update, and remove external generators and plugins.

#### For Generators (`lang` subcommand):

*   `mirror lang install <name> [--version <version>] [--source <url>]`
    
*   `mirror lang uninstall <name>`
    
*   `mirror lang update [<name>]`
    
*   `mirror lang list`
    
*   `mirror lang create <name>`
    
*   `mirror lang upload <name>`
    

#### For Plugins (`plugin` subcommand):

*   `mirror plugin install <name> [--version <version>] [--source <url>] [--alias <alias>]` – allows assigning an alias during installation.
    
*   `mirror plugin uninstall <name>`
    
*   `mirror plugin update [<name>]`
    
*   `mirror plugin list`
    
*   `mirror plugin create <name>`
    
*   `mirror plugin upload <name>`
    

The structure of an external plugin can be similar to that of a generator: an executable that, when receiving the `list_functions` command via stdin, returns a JSON with the functions it exposes. For function execution, a simple mode can be implemented where the plugin receives a JSON with the function and arguments and returns the result.