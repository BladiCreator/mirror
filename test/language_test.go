package test

import (
	"testing"

	"github.com/BladiCreator/mirror/internal/languages"
)

func TestRegistryAliases(t *testing.T) {
	reg := languages.NewRegistry("")

	tests := []struct {
		name     string
		found    bool
		expected string
	}{
		{"typescript", true, "typescript"},
		{"ts", true, "typescript"},
		{"go", true, "go"},
		{"golang", true, "go"},
		{"surrealql", true, "surrealql"},
		{"surrql", true, "surrealql"},
		{"dart", true, "dart"},
		{"unknown", false, ""},
	}

	for _, tt := range tests {
		lang, ok := reg.Get(tt.name)
		if ok != tt.found {
			t.Errorf("Get(%s) ok = %v, want %v", tt.name, ok, tt.found)
		}
		if ok && lang.Name() != tt.expected {
			t.Errorf("Get(%s).Name() = %s, want %s", tt.name, lang.Name(), tt.expected)
		}
	}
}
