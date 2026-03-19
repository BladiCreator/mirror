package internal

import (
	"strings"

	pm "github.com/mirror/mirror/internal/plugin/model"
)

// InternalPlugins returns built-in plugins.
func InternalPlugins() []pm.Plugin {
	return []pm.Plugin{&GoPlugin{}, &DartPlugin{}}
}

func ApplyFormat(name string, format string) string {
	if format == "" || format == "pascal" {
		return strings.Title(name)
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
		p := strings.Title(parts[0])
		return p + strings.Join(parts[1:], "")
	case "kebab":
		return strings.ReplaceAll(strings.ToLower(norm), " ", "-")
	case "pascal":
		return strings.Title(norm)
	default:
		return name
	}
}

func ConvertName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' {
			return ' '
		}
		return r
	}, name)
}
