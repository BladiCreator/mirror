package parser

import (
	"fmt"
	"path/filepath"

	lm "github.com/BladiCreator/mirror/internal/languages/model"
	"github.com/BladiCreator/mirror/internal/model"
)

var extensionToLanguage = map[string]string{
	".go":   "go",
	".dart": "dart",
	// Add more as needed
}

// DetectPredominantLanguage returns the language code with the highest detection score.
func DetectPredominantLanguage(dir string, pattern string, analyzers map[string]lm.Analyzer) (string, error) {
	// If pattern contains extension, infer language
	if pattern != "" {
		ext := filepath.Ext(pattern)
		if lang, ok := extensionToLanguage[ext]; ok {
			return lang, nil
		}
	}

	bestLang := ""
	maxCount := -1

	for lang, a := range analyzers {
		count, err := a.Detect(dir, pattern)
		if err != nil {
			continue
		}
		if count > maxCount {
			maxCount = count
			bestLang = lang
		}
	}

	if maxCount <= 0 {
		if pattern != "" {
			return "", fmt.Errorf("no supported source files detected in %s with pattern %s", dir, pattern)
		} else {
			return "", fmt.Errorf("no supported source files detected in %s", dir)
		}
	}

	return bestLang, nil
}

// ExtractSchemas uses the specified language's analyzer to extract schemas.
func ExtractSchemas(lang, dir, pattern string, analyzers map[string]lm.Analyzer) ([]*model.Schema, error) {
	a, ok := analyzers[lang]
	if !ok {
		return nil, fmt.Errorf("no analyzer found for %s", lang)
	}
	return a.Extract(dir, pattern)
}

// InitialSetup interactively creates the mirror.yml.
// Since we are in an AI context, we might need a non-interactive way or clear instructions.
func InitialSetup(scanDir string, detectedLang string, schemas []*model.Schema, chosenLangs []string) (*model.MirrorFile, error) {
	mrr := &model.MirrorFile{
		Languages: make(map[string]model.LanguageConfig),
		Schemas:   make(map[string]*model.Schema),
	}

	basePath := "."
	if scanDir != "." {
		basePath = "./" + filepath.Base(scanDir)
	}

	for _, l := range chosenLangs {
		config := model.LanguageConfig{
			Output: &model.OutputSettings{
				Filepath: basePath,
				Format:   "pascal",
			},
		}
		if l == "dart" {
			config.Output.Format = "snake"
		} else if l == "go" {
			config.Output.Format = "pascal"
		}
		mrr.Languages[l] = config
	}

	for _, s := range schemas {
		mrr.Schemas[s.Name] = s
	}

	return mrr, nil
}
