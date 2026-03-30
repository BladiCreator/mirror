package model

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/BladiCreator/mirror/internal/model"
)

// Language defines generation behavior.
type Language interface {
	Name() string
	Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error)
	ResolveType(typeStr string) string
	Analyzer() Analyzer
	Template() (string, error)
}

// Analyzer defines how to detect and extract schemas from source files.
type Analyzer interface {
	Detect(dir string) (int, error)
	Extract(dir string) ([]*model.Schema, error)
}

// ExternalLanguage executes an external binary plugin via JSON stdin/stdout.
type ExternalLanguage struct {
	name string
	path string
}

func NewExternalLanguage(name string, path string) Language {
	return &ExternalLanguage{name: name, path: path}
}

func (p *ExternalLanguage) Name() string { return p.name }

func (p *ExternalLanguage) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
	in := struct {
		Schemas      []*model.Schema    `json:"schemas"`
		OutputConfig model.OutputConfig `json:"output_config"`
	}{Schemas: schemas, OutputConfig: cfg}

	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.path)
	cmd.Stdin = strings.NewReader(string(b))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("external plugin %s failed: %w", p.name, err)
	}

	var resp struct {
		Files []model.GeneratedFile `json:"files"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("invalid plugin output from %s: %w", p.name, err)
	}

	return resp.Files, nil
}

func (p *ExternalLanguage) ResolveType(t string) string {
	return t
}

func (p *ExternalLanguage) Analyzer() Analyzer {
	return nil
}

func (p *ExternalLanguage) Template() (string, error) {
	return "Template showing for external plugins is not yet implemented", nil
}
