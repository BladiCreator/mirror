package test

import (
	"os"
	"testing"

	"github.com/BladiCreator/mirror/internal/parser"
)

func TestParseLanguageConfig(t *testing.T) {
	content := `
plugin: []
languages:
  - legacy:
      filepath: "./legacy/path"
      suffix: "_legacy"
      format: snake
  - single:
      output:
        filepath: "./single/path"
        format: pascal
  - multiple:
      output:
        filepath:
          - "./path1"
          - "./path2"
        suffix: "_multi"
        format: camel
`
	tmpfile, err := os.CreateTemp("", "mirror_test_*.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	mrr, err := parser.ParseYAMLFile(tmpfile.Name(), map[string]bool{}, true)
	if err != nil {
		t.Fatalf("parseYAMLFile failed: %v", err)
	}

	// Test Legacy
	legacy, ok := mrr.Languages["legacy"]
	if !ok {
		t.Fatal("legacy config missing")
	}
	paths := legacy.GetFilepaths()
	if len(paths) != 1 || paths[0] != "./legacy/path" {
		t.Errorf("legacy paths = %v, want ['./legacy/path']", paths)
	}
	if legacy.GetSuffix() != "_legacy" {
		t.Errorf("legacy suffix = %q, want '_legacy'", legacy.GetSuffix())
	}
	if legacy.GetFormat() != "snake" {
		t.Errorf("legacy format = %q, want 'snake'", legacy.GetFormat())
	}

	// Test Single Output
	single, ok := mrr.Languages["single"]
	if !ok {
		t.Fatal("single config missing")
	}
	paths = single.GetFilepaths()
	if len(paths) != 1 || paths[0] != "./single/path" {
		t.Errorf("single paths = %v, want ['./single/path']", paths)
	}
	if single.GetFormat() != "pascal" {
		t.Errorf("single format = %q, want 'pascal'", single.GetFormat())
	}

	// Test Multiple Output
	multiple, ok := mrr.Languages["multiple"]
	if !ok {
		t.Fatal("multiple config missing")
	}
	paths = multiple.GetFilepaths()
	if len(paths) != 2 || paths[0] != "./path1" || paths[1] != "./path2" {
		t.Errorf("multiple paths = %v, want ['./path1', './path2']", paths)
	}
	if multiple.GetSuffix() != "_multi" {
		t.Errorf("multiple suffix = %q, want '_multi'", multiple.GetSuffix())
	}
	if multiple.GetFormat() != "camel" {
		t.Errorf("multiple format = %q, want 'camel'", multiple.GetFormat())
	}
}
