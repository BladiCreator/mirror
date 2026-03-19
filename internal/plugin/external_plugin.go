package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mirror/mirror/internal/model"
)

// ExternalPlugin executes an external binary plugin via JSON stdin/stdout.
type ExternalPlugin struct {
	name string
	path string
}

func (p *ExternalPlugin) Name() string { return p.name }

func (p *ExternalPlugin) Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error) {
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
