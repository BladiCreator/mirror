package languages

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mirror/mirror/internal/languages/internal"
	pm "github.com/mirror/mirror/internal/languages/model"
)

// Registry holds known languages.
type Registry struct {
	internal    map[string]pm.Language
	languageDir string
}

// NewRegistry creates a language registry.
func NewRegistry(languagesDir string) *Registry {
	reg := &Registry{internal: map[string]pm.Language{}, languageDir: languagesDir}
	for _, p := range internal.InternalLanguage() {
		reg.internal[p.Name()] = p
	}
	return reg
}

func (r *Registry) Get(name string) (pm.Language, bool) {
	if p, ok := r.internal[name]; ok {
		return p, true
	}
	if externalPath, err := r.findExternalBinary(name); err == nil {
		return pm.NewExternalLanguage(name, externalPath), true
	}
	return nil, false
}

func (r *Registry) findExternalBinary(name string) (string, error) {
	if r.languageDir != "" {
		candidate := filepath.Join(r.languageDir, name)
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

// SetInternal adds or replaces an internal language (for tests).
func (r *Registry) SetInternal(p pm.Language) {
	r.internal[p.Name()] = p
}

// Analyzers returns all available analyzers from internal languages.
func (r *Registry) Analyzers() map[string]pm.Analyzer {
	res := make(map[string]pm.Analyzer)
	for name, lang := range r.internal {
		if a := lang.Analyzer(); a != nil {
			res[name] = a
		}
	}
	return res
}
