package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/mirror/mirror/internal/model"
)

var validFormats = map[string]bool{"pascal": true, "snake": true, "camel": true, "kebab": true, "": true}

// ParseFile parses a .mrr file and all its imports.
func ParseFile(path string) (*model.MRRFile, error) {
	return parseFile(path, map[string]bool{}, false)
}

func parseFile(path string, visited map[string]bool, schemaOnly bool) (*model.MRRFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[abs] {
		return nil, fmt.Errorf("circular import detected: %s", abs)
	}
	visited[abs] = true

	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	mrr := &model.MRRFile{Filepath: abs, Schemas: map[string]*model.Schema{}}
	var currentSection string
	var currentSchema *model.Schema

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := leadingSpaces(line)
		trim := strings.TrimSpace(line)

		if indent == 0 {
			// section header
			switch strings.ToLower(trim) {
			case "plugin", "paths", "schemas":
				currentSection = strings.ToLower(trim)
				currentSchema = nil
				continue
			default:
				return nil, fmt.Errorf("unexpected top-level token '%s' in %s", trim, path)
			}
		}

		if !strings.HasPrefix(strings.TrimSpace(line), "-") {
			continue
		}

		item := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))

		switch currentSection {
		case "plugin":
			mrr.Plugins = append(mrr.Plugins, item)
		case "paths":
			p, err := parsePathEntry(item)
			if err != nil {
				return nil, fmt.Errorf("parse paths: %w", err)
			}
			mrr.Paths = append(mrr.Paths, p)
		case "schemas":
			if indent <= 2 {
				// schema declaration or import
				if isImportSyntax(item) {
					importPath, err := normalizeImportPath(item, abs)
					if err != nil {
						return nil, err
					}
					mrr.Imports = append(mrr.Imports, importPath)
					child, err := parseFile(importPath, visited, true)
					if err != nil {
						return nil, err
					}
					for k, v := range child.Schemas {
						if _, exists := mrr.Schemas[k]; exists {
							return nil, fmt.Errorf("schema duplicate %q from import %s", k, importPath)
						}
						mrr.Schemas[k] = v
					}
					continue
				}
				schema, err := parseSchema(item)
				if err != nil {
					return nil, fmt.Errorf("parse schema in %s: %w", path, err)
				}
				if _, exists := mrr.Schemas[schema.Name]; exists {
					return nil, fmt.Errorf("schema %q already defined in %s", schema.Name, path)
				}
				mrr.Schemas[schema.Name] = schema
				currentSchema = schema
			} else {
				if currentSchema == nil {
					return nil, fmt.Errorf("field without parent schema in %s", path)
				}
				field, err := parseField(item)
				if err != nil {
					return nil, fmt.Errorf("parse field in %s: %w", path, err)
				}
				currentSchema.Fields = append(currentSchema.Fields, field)
			}
		default:
			return nil, fmt.Errorf("line outside section in %s: %s", path, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !schemaOnly {
		if err := Validate(mrr); err != nil {
			return nil, err
		}
	}

	return mrr, nil
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r == ' ' || r == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

func toBacktickTokens(s string) []string {
	re := regexp.MustCompile("`([^`]*)`")
	matches := re.FindAllStringSubmatch(s, -1)
	var out []string
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

func splitTokens(raw string) []string {
	var out []string
	var sb strings.Builder
	inQuote := false
	for _, r := range raw {
		if r == '\'' {
			inQuote = !inQuote
			sb.WriteRune(r)
			continue
		}
		if unicode.IsSpace(r) && !inQuote {
			if sb.Len() > 0 {
				out = append(out, sb.String())
				sb.Reset()
			}
			continue
		}
		sb.WriteRune(r)
	}
	if sb.Len() > 0 {
		out = append(out, sb.String())
	}
	return out
}

func parsePathEntry(raw string) (*model.PathEntry, error) {
	if raw == "" {
		return nil, errors.New("empty path entry")
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil, errors.New("empty path entry")
	}
	ext := fields[0]
	if ext == "" {
		return nil, errors.New("path extension is required")
	}
	segments := toBacktickTokens(raw)
	if len(segments) == 0 {
		return nil, errors.New("path entry requires a plugin list in backticks")
	}

	pluginSegment := strings.TrimSpace(segments[0])
	var plugins []string
	var optionSegment string

	if strings.HasPrefix(pluginSegment, "p::") {
		withoutKey := strings.TrimSpace(strings.TrimPrefix(pluginSegment, "p::"))
		toks := splitTokens(withoutKey)
		if len(toks) == 0 {
			return nil, errors.New("path entry has empty p:: plugin specification")
		}
		plugins = strings.Split(strings.TrimSpace(toks[0]), ",")
		for i := range plugins {
			plugins[i] = strings.TrimSpace(plugins[i])
		}
		if len(toks) > 1 {
			optionSegment = strings.Join(toks[1:], " ")
		}
	} else {
		plugins = strings.Split(pluginSegment, ",")
		for i := range plugins {
			plugins[i] = strings.TrimSpace(plugins[i])
		}
		if len(segments) > 1 {
			optionSegment = segments[1]
		}
	}

	entry := &model.PathEntry{Ext: ext, Plugins: plugins, OutputDir: "", Suffix: "", Format: ""}

	if optionSegment != "" {
		opts, err := parseOptions(optionSegment)
		if err != nil {
			return nil, err
		}
		if v, ok := opts["f"]; ok {
			entry.OutputDir = v
		}
		if v, ok := opts["format"]; ok {
			entry.Format = v
		}
		if v, ok := opts["suffix"]; ok {
			entry.Suffix = v
		}
	}
	if entry.OutputDir == "" {
		entry.OutputDir = ext
	}
	return entry, nil
}

func parseOptions(raw string) (map[string]string, error) {
	optRe := regexp.MustCompile(`([a-zA-Z0-9_]+)(::?'([^']*)')?`)
	m := make(map[string]string)
	for _, part := range strings.Fields(raw) {
		match := optRe.FindStringSubmatch(part)
		if len(match) >= 2 {
			k := match[1]
			if len(match) >= 4 && match[3] != "" {
				m[k] = match[3]
			}
		}
	}
	return m, nil
}

func isImportSyntax(item string) bool {
	trim := strings.TrimSpace(item)
	if strings.HasPrefix(trim, "'") && strings.HasSuffix(trim, "'") {
		trim = strings.Trim(trim, "'")
	}
	return strings.HasSuffix(trim, ".mrr")
}

func normalizeImportPath(item, parent string) (string, error) {
	trim := strings.TrimSpace(item)
	if strings.HasPrefix(trim, "'") && strings.HasSuffix(trim, "'") {
		trim = strings.Trim(trim, "'")
	}
	if !filepath.IsAbs(trim) {
		return filepath.Join(filepath.Dir(parent), trim), nil
	}
	return trim, nil
}

func parseSchema(item string) (*model.Schema, error) {
	parts := strings.SplitN(item, ":", 2)
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return nil, errors.New("schema name is required")
	}
	tags := map[string]string{}
	if len(parts) > 1 {
		raw := parts[1]
		if back := toBacktickTokens(raw); len(back) > 0 {
			tags = parseTags(back[0])
		}
	}
	return &model.Schema{Name: name, Tags: tags, Fields: []*model.Field{}}, nil
}

func parseField(item string) (*model.Field, error) {
	tokenRe := regexp.MustCompile("`([^`]*)`")
	tokens := tokenRe.FindAllStringSubmatch(item, -1)
	if len(tokens) < 1 {
		return nil, fmt.Errorf("field must include type in backticks: %s", item)
	}

	namePart := strings.Fields(strings.Replace(item, tokens[0][0], "", 1))[0]
	if namePart == "" {
		return nil, errors.New("field name is required")
	}
	ftype := strings.TrimSpace(tokens[0][1])
	var tags map[string]string
	if len(tokens) > 1 {
		tags = parseTags(tokens[1][1])
	} else {
		tags = map[string]string{}
	}
	return &model.Field{Name: namePart, Type: ftype, Tags: tags}, nil
}

func parseTags(raw string) map[string]string {
	result := map[string]string{}
	for _, t := range strings.Fields(raw) {
		if strings.Contains(t, ":") {
			parts := strings.SplitN(t, ":", 2)
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

// Validate performs semantic checks.
func Validate(mrr *model.MRRFile) error {
	if len(mrr.Plugins) == 0 {
		return errors.New("plugin section is required and requires at least one plugin")
	}
	if len(mrr.Paths) == 0 {
		return errors.New("paths section is required and requires at least one path")
	}
	if len(mrr.Schemas) == 0 {
		return errors.New("schemas section is required and requires at least one schema")
	}

	pluginsMap := map[string]bool{}
	for _, p := range mrr.Plugins {
		pluginsMap[p] = true
	}

	for _, pathEntry := range mrr.Paths {
		if len(pathEntry.Plugins) == 0 {
			return fmt.Errorf("path entry extension %q has no plugins", pathEntry.Ext)
		}
		for _, pn := range pathEntry.Plugins {
			if !pluginsMap[pn] {
				return fmt.Errorf("path entry extension %q references plugin that is not declared: %s", pathEntry.Ext, pn)
			}
		}
		if !validFormats[pathEntry.Format] {
			return fmt.Errorf("invalid format %q in path %s", pathEntry.Format, pathEntry.Ext)
		}
	}

	for _, s := range mrr.Schemas {
		if s.Name == "" {
			return errors.New("schema with empty name")
		}
		for _, f := range s.Fields {
			if f.Name == "" || f.Type == "" {
				return fmt.Errorf("schema %s has invalid field", s.Name)
			}
			if strings.HasPrefix(f.Type, "object:") {
				ref := strings.TrimPrefix(f.Type, "object:")
				if _, ok := mrr.Schemas[ref]; !ok {
					return fmt.Errorf("schema %q references unknown object type %q", s.Name, ref)
				}
			}
		}
	}

	return nil
}
