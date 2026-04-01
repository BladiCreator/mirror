package model

import (
	"strings"
	"unicode"
)

// ApplyFormat styles a name according to a given format string.
func ApplyFormat(name string, format string) string {
	if format == "" || format == "pascal" {
		return ToPascal(name)
	}
	norm := ConvertName(name)
	switch format {
	case "snake":
		return strings.ReplaceAll(strings.ToLower(norm), " ", "_")
	case "camel":
		parts := strings.Fields(norm)
		if len(parts) == 0 {
			return name
		}
		var p strings.Builder
		p.WriteString(strings.ToLower(parts[0]))
		for i := 1; i < len(parts); i++ {
			p.WriteString(TitleCase(parts[i]))
		}
		return p.String()
	case "kebab":
		return strings.ReplaceAll(strings.ToLower(norm), " ", "-")
	case "pascal":
		return ToPascal(name)
	default:
		return name
	}
}

// ConvertName replaces separators with spaces.
func ConvertName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' {
			return ' '
		}
		return r
	}, name)
}

// ToPascal converts string space separated to PascalCase
func ToPascal(s string) string {
	parts := strings.Fields(ConvertName(s))
	for i, p := range parts {
		parts[i] = TitleCase(p)
	}
	return strings.Join(parts, "")
}

// TitleCase title cases a string.
func TitleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
