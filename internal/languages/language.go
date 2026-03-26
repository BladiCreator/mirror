package languages

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mirror/mirror/internal/languages/builtin"
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
	for _, l := range builtin.InternalLanguage() {
		reg.internal[l.Name()] = l
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
		candidate := filepath.Join(r.languageDir, "mirror-lang-"+name)
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return execLookPath("mirror-lang-" + name)
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

// ListInternal returns names of all internal languages.
func (r *Registry) ListInternal() []string {
	var res []string
	for name := range r.internal {
		res = append(res, name)
	}
	return res
}

// ListExternal returns names of all external plugins found in languageDir.
func (r *Registry) ListExternal() []string {
	var res []string
	if r.languageDir == "" {
		return res
	}
	files, err := os.ReadDir(r.languageDir)
	if err != nil {
		return res
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "mirror-lang-") {
			name := strings.TrimPrefix(f.Name(), "mirror-lang-")
			if runtime.GOOS == "windows" {
				name = strings.TrimSuffix(name, ".exe")
			}
			// Avoid duplicates if an internal language has a binary with the same name
			if _, ok := r.internal[name]; ok {
				continue
			}
			res = append(res, name)
		}
	}
	return res
}
