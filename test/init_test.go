package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BladiCreator/mirror/internal/languages"
	"github.com/BladiCreator/mirror/internal/model"
	"github.com/BladiCreator/mirror/internal/parser"
)

func TestGoAnalyzer(t *testing.T) {
	tmp := t.TempDir()

	goCode := `package models
type User struct {
	ID int
	Name string
}
`
	if err := os.WriteFile(filepath.Join(tmp, "user.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmp, "inner")
	os.Mkdir(subDir, 0755)
	innerCode := `package inner
type Profile struct {
	Bio string
}
`
	if err := os.WriteFile(filepath.Join(subDir, "profile.go"), []byte(innerCode), 0644); err != nil {
		t.Fatal(err)
	}

	reg := languages.NewRegistry("")
	analyzers := reg.Analyzers()
	goAnalyzer, ok := analyzers["go"]
	if !ok {
		t.Fatal("go analyzer not found")
	}

	count, err := goAnalyzer.Detect(tmp, "")
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 go files, got %d", count)
	}

	schemas, err := goAnalyzer.Extract(tmp, "")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}
}

func TestDartAnalyzer(t *testing.T) {
	tmp := t.TempDir()

	dartCode := `class Item {
  final int id;
  final String title;
  Item({required this.id, required this.title});
}
`
	if err := os.WriteFile(filepath.Join(tmp, "item.dart"), []byte(dartCode), 0644); err != nil {
		t.Fatal(err)
	}

	reg := languages.NewRegistry("")
	analyzers := reg.Analyzers()
	dartAnalyzer, ok := analyzers["dart"]
	if !ok {
		t.Fatal("dart analyzer not found")
	}

	count, _ := dartAnalyzer.Detect(tmp, "")
	if count != 1 {
		t.Errorf("expected 1 dart file, got %d", count)
	}

	schemas, err := dartAnalyzer.Extract(tmp, "")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if len(schemas) != 1 || schemas[0].Name != "Item" {
		t.Fatalf("expected schema Item, got %v", schemas)
	}
}

func TestPatternDetection(t *testing.T) {
	tmp := t.TempDir()

	// Create a file matching pattern
	goCode := `package models
type UserModel struct {
	ID int
}
`
	if err := os.WriteFile(filepath.Join(tmp, "user_model.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	reg := languages.NewRegistry("")
	analyzers := reg.Analyzers()

	// Test pattern detection
	detected, err := parser.DetectPredominantLanguage(tmp, "*_model.go", analyzers)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if detected != "go" {
		t.Errorf("expected go, got %s", detected)
	}

	// Test extension inference
	detected2, err := parser.DetectPredominantLanguage(tmp, "*.go", analyzers)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if detected2 != "go" {
		t.Errorf("expected go, got %s", detected2)
	}
}

func TestInitialSetup(t *testing.T) {
	schemas := []*model.Schema{
		{Name: "User", Fields: []*model.Field{{Name: "id", Type: "int"}}},
	}
	chosen := []string{"go", "dart"}

	mrr, err := parser.InitialSetup(".", "go", schemas, chosen)
	if err != nil {
		t.Fatalf("InitialSetup failed: %v", err)
	}

	if len(mrr.Languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(mrr.Languages))
	}
	if _, ok := mrr.Schemas["User"]; !ok {
		t.Error("schema User missing from setup")
	}
}
