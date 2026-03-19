package model

import "github.com/mirror/mirror/internal/model"

// Plugin defines generation behavior.
type Plugin interface {
	Name() string
	Generate(schemas []*model.Schema, cfg model.OutputConfig) ([]model.GeneratedFile, error)
}
