package functions

import (
	"strings"
	"text/template"
)

// Registry contains all available internal function plugins.
var Registry = map[string]map[string]any{
	"strings": {
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"toTitle": strings.Title,
		// the previous `model.ToPascal` could be moved here but strings.Title is enough for demonstration.
		// adding strings package utilities mappings
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace": strings.ReplaceAll,
		"split": strings.Split,
	},
}

// ResolveFuncs takes a list of plugin declarations (e.g. "strings" or "strings:st")
// and returns a template.FuncMap with the functions properly namespaced with "fn:alias:func".
func ResolveFuncs(plugins []string) template.FuncMap {
	funcs := template.FuncMap{}

	for _, p := range plugins {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		parts := strings.SplitN(p, ":", 2)
		pluginName := parts[0]
		alias := pluginName // default alias is the plugin name
		if len(parts) > 1 {
			alias = parts[1]
		}

		// Look up in internal registry
		if pFuncs, ok := Registry[pluginName]; ok {
			for name, fn := range pFuncs {
				key := "fn_" + alias + "_" + name
				funcs[key] = fn
			}
		}
		// In the future, this is where external plugin functions would be loaded or proxied.
	}

	return funcs
}
