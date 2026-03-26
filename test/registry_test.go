package test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mirror/mirror/internal/languages"
)

func TestRegistryInternal(t *testing.T) {
	reg := languages.NewRegistry("")
	
	// Check if internal languages are present
	if _, ok := reg.Get("go"); !ok {
		t.Error("go internal language missing")
	}
	if _, ok := reg.Get("dart"); !ok {
		t.Error("dart internal language missing")
	}
	
	internal := reg.ListInternal()
	if len(internal) < 2 {
		t.Errorf("expected at least 2 internal languages, got %d", len(internal))
	}
}

func TestRegistryExternal(t *testing.T) {
	tmp := t.TempDir()
	
	// Create a fake plugin binary
	// On Windows, the registry looks for .exe or no extension
	// Let's create both to be safe or use a generic name
	pluginName := "mirror-lang-rust"
	if runtime.GOOS == "windows" {
		pluginName += ".exe"
	}
	pluginPath := filepath.Join(tmp, pluginName)
	if err := os.WriteFile(pluginPath, []byte(""), 0755); err != nil {
		t.Fatal(err)
	}
	
	reg := languages.NewRegistry(tmp)
	
	external := reg.ListExternal()
	found := false
	for _, p := range external {
		if p == "rust" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("external plugin 'rust' not found by registry in %s, found: %v", tmp, external)
	}
	
	l, ok := reg.Get("rust")
	if !ok {
		t.Fatal("could not get external plugin 'rust'")
	}
	if l.Name() != "rust" {
		t.Errorf("expected plugin name rust, got %s", l.Name())
	}
}
