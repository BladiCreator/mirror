package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BladiCreator/mirror/internal/parser"
)

func TestParseYaml(t *testing.T) {
	tmp := t.TempDir()
	samplePath := filepath.Join(tmp, "sample.yml")
	sample := `languages:
  - dart:
      filepath: "./lib/models"
      suffix: _dart
      format: snake
  - go:
      filepath: "./internal/models"
      format: pascal

schemas:
  - name: usuario
    fields:
      - name: id
        type: int
      - name: nombre
        type: string
  - name: profile
    fields:
      - name: bio
        type: string
`
	if err := os.WriteFile(samplePath, []byte(sample), 0644); err != nil {
		t.Fatalf("write sample yml: %v", err)
	}

	mrr, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("parse yml failed: %v", err)
	}
	if len(mrr.Languages) != 2 || len(mrr.Schemas) != 2 {
		t.Fatalf("unexpected parse counts for yaml: languages=%d schemas=%d", len(mrr.Languages), len(mrr.Schemas))
	}
	if _, ok := mrr.Schemas["usuario"]; !ok {
		t.Fatal("usuario schema missing yaml")
	}
	if _, ok := mrr.Schemas["profile"]; !ok {
		t.Fatal("profile schema missing yaml")
	}
}

func TestParseYamlInclude(t *testing.T) {
	tmp := t.TempDir()
	commonPath := filepath.Join(tmp, "common.yml")
	common := `schemas:
  - name: direccion
    fields:
      - name: calle
        type: string
      - name: ciudad
        type: string
`
	if err := os.WriteFile(commonPath, []byte(common), 0644); err != nil {
		t.Fatalf("write common yaml: %v", err)
	}

	samplePath := filepath.Join(tmp, "sample.yml")
	sample := `languages:
  - go:
      filepath: "./internal/models"

schemas:
  - name: usuario
    fields:
      - name: id
        type: int
  - include: 'common.yml'
`
	if err := os.WriteFile(samplePath, []byte(sample), 0644); err != nil {
		t.Fatalf("write sample yaml include: %v", err)
	}

	mrr, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("parse yaml include failed: %v", err)
	}
	if _, ok := mrr.Schemas["direccion"]; !ok {
		t.Fatal("direccion schema missing yaml include from", commonPath)
	}
}

func TestParseYamlImport(t *testing.T) {
	tmp := t.TempDir()
	samplePath := filepath.Join(tmp, "sample.yml")
	sample := `languages:
  - dart:
      filepath: "./lib/models"
  - go:
      filepath: "./internal/models"

schemas:
  - name: usuario
    import:
      disable: false
      go: ["fmt", "os"]
      dart: ["package:flutter/material.dart"]
    fields:
      - name: id
        type: int
      - name: perfil
        type: object:profile
  - name: profile
    fields:
      - name: bio
        type: string
`
	if err := os.WriteFile(samplePath, []byte(sample), 0644); err != nil {
		t.Fatalf("write sample yml: %v", err)
	}

	mrr, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("parse yml failed: %v", err)
	}

	u, ok := mrr.Schemas["usuario"]
	if !ok {
		t.Fatal("usuario schema missing")
	}

	if u.Import == nil {
		t.Fatal("import config missing for usuario")
	}

	if len(u.Import.Langs["go"]) != 3 {
		t.Errorf("expected 3 go imports (fmt, os, auto:profile), got %d", len(u.Import.Langs["go"]))
	}

	// Check auto import in dart
	dartImps := u.Import.Langs["dart"]
	foundAuto := false
	for _, imp := range dartImps {
		if imp == "auto:profile" {
			foundAuto = true
			break
		}
	}
	if !foundAuto {
		t.Error("auto:profile import missing for dart")
	}
}
