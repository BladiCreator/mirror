package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/mirror/mirror/internal/model"
)

func parseMRRFile(path string, visited map[string]bool, schemaOnly bool) (*model.MRRFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	mrr := &model.MRRFile{Filepath: path, Schemas: map[string]*model.Schema{}}
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
					importPath, err := normalizeImportPath(item, path)
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
	return strings.HasSuffix(trim, ".mrr") || strings.HasSuffix(trim, ".yml") || strings.HasSuffix(trim, ".yaml")
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
