package test

import (
	"strings"
	"testing"

	"github.com/BladiCreator/mirror/internal/languages/builtin"
	"github.com/BladiCreator/mirror/internal/languages/builtin/dart"
	"github.com/BladiCreator/mirror/internal/languages/builtin/golang"
	"github.com/BladiCreator/mirror/internal/model"
)

func TestGenerationWithImports(t *testing.T) {
	langs := builtin.InternalLanguage()

	schemas := []*model.Schema{
		{
			Name: "usuario",
			Import: &model.ImportConfig{
				Disable: false,
				Langs: map[string][]string{
					"go":   {"fmt"},
					"dart": {"package:flutter/material.dart", "auto:profile"},
				},
			},
			Fields: []*model.Field{
				{Name: "id", Type: "int"},
			},
		},
	}

	cfg := model.OutputConfig{Filepath: "./models", Format: "pascal"}

	// Test Go generation
	var goPlg *golang.GoLanguage
	for _, l := range langs {
		if l.Name() == "go" {
			goPlg = l.(*golang.GoLanguage)
		}
	}

	goFiles, _ := goPlg.Generate(schemas, cfg)

	if !strings.Contains(goFiles[0].Content, "import (\n  \"fmt\"\n)") {
		t.Errorf("Go output missing expected import, got:\n%s", goFiles[0].Content)
	}
	if strings.Contains(goFiles[0].Content, "auto:profile") {
		t.Error("Go output should not contain auto:profile")
	}

	// Test Dart generation
	var dartPlg *dart.DartLanguage
	for _, l := range langs {
		if l.Name() == "dart" {
			dartPlg = l.(*dart.DartLanguage)
		}
	}

	dartFiles, _ := dartPlg.Generate(schemas, model.OutputConfig{Filepath: "./lib/models", Format: "snake"})

	if !strings.Contains(dartFiles[0].Content, "package:flutter/material.dart") {
		t.Errorf("Dart output missing manual import, got:\n%s", dartFiles[0].Content)
	}
	if !strings.Contains(dartFiles[0].Content, "import 'profile.dart';") {
		t.Errorf("Dart output missing auto import, got:\n%s", dartFiles[0].Content)
	}
}
