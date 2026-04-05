package yamlconfig

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// Loader loads service definitions from a YAML file (implements ports.ServiceLoader).
type Loader struct {
	Path string
}

var _ ports.ServiceLoader = (*Loader)(nil)

// NewLoader returns a file-based catalog loader.
func NewLoader(path string) *Loader {
	return &Loader{Path: path}
}

// LoadDefinitions implements ports.ServiceLoader.
func (l *Loader) LoadDefinitions(_ context.Context) ([]domain.ServiceDefinition, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var raw domain.RawFileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return domain.ResolveServiceDefinitions(&raw)
}
