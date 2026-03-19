package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mirror/mirror/internal/plugin/internal"
	pm "github.com/mirror/mirror/internal/plugin/model"
)

// Registry holds known plugins.
type Registry struct {
	internal   map[string]pm.Plugin
	pluginsDir string
}

// NewRegistry creates a plugin registry.
func NewRegistry(pluginsDir string) *Registry {
	reg := &Registry{internal: map[string]pm.Plugin{}, pluginsDir: pluginsDir}
	for _, p := range internal.InternalPlugins() {
		reg.internal[p.Name()] = p
	}
	return reg
}

func (r *Registry) Get(name string) (pm.Plugin, bool) {
	if p, ok := r.internal[name]; ok {
		return p, true
	}
	if externalPath, err := r.findExternalBinary(name); err == nil {
		return &ExternalPlugin{name: name, path: externalPath}, true
	}
	return nil, false
}

func (r *Registry) findExternalBinary(name string) (string, error) {
	if r.pluginsDir != "" {
		candidate := filepath.Join(r.pluginsDir, name)
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return execLookPath(name)
}

func execLookPath(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return p, nil
}

// SetInternal adds or replaces an internal plugin (for tests).
func (r *Registry) SetInternal(p pm.Plugin) {
	r.internal[p.Name()] = p
}

// EnsurePluginDeclared check compares configuration plugin names with declared ones.
func EnsurePluginDeclared(declared []string, used []string) error {
	lookup := map[string]bool{}
	for _, d := range declared {
		lookup[d] = true
	}
	for _, u := range used {
		if !lookup[u] {
			return fmt.Errorf("plugin %q referenced by path is not declared in plugin section", u)
		}
	}
	return nil
}
