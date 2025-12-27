package scenario

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Registry holds all available scenarios
type Registry struct {
	scenarios map[string]*Scenario
}

// NewRegistry creates a new scenario registry
func NewRegistry() *Registry {
	return &Registry{
		scenarios: make(map[string]*Scenario),
	}
}

// LoadFromFile loads a scenario from a YAML file
func (r *Registry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read scenario file: %w", err)
	}

	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return fmt.Errorf("failed to parse scenario YAML: %w", err)
	}

	r.scenarios[scenario.Name] = &scenario
	return nil
}

// LoadFromDir loads all scenarios from a directory
func (r *Registry) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := r.LoadFromFile(path); err != nil {
			return fmt.Errorf("failed to load scenario from %s: %w", path, err)
		}
	}

	return nil
}

// LoadFromEmbedded loads scenarios from embedded filesystem
func (r *Registry) LoadFromEmbedded(fs embed.FS, dir string) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read embedded scenarios: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := fs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		var scenario Scenario
		if err := yaml.Unmarshal(data, &scenario); err != nil {
			return fmt.Errorf("failed to parse scenario YAML from %s: %w", path, err)
		}

		r.scenarios[scenario.Name] = &scenario
	}

	return nil
}

// Get retrieves a scenario by name
func (r *Registry) Get(name string) (*Scenario, error) {
	scenario, ok := r.scenarios[name]
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found", name)
	}
	return scenario, nil
}

// List returns all scenario names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.scenarios))
	for name := range r.scenarios {
		names = append(names, name)
	}
	return names
}

// ListWithDescriptions returns all scenarios with their descriptions
func (r *Registry) ListWithDescriptions() map[string]string {
	result := make(map[string]string)
	for name, scenario := range r.scenarios {
		result[name] = scenario.Description
	}
	return result
}
