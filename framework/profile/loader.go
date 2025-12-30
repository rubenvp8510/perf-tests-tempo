package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// Load reads a profile from a YAML file
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file %s: %w", path, err)
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile %s: %w", path, err)
	}

	if err := Validate(&profile); err != nil {
		return nil, fmt.Errorf("invalid profile %s: %w", path, err)
	}

	return &profile, nil
}

// LoadAll reads all YAML profiles from a directory
func LoadAll(dir string) ([]*Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory %s: %w", dir, err)
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		profile, err := Load(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// LoadByNames loads specific profiles by name from a directory
func LoadByNames(dir string, names []string) ([]*Profile, error) {
	var profiles []*Profile
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Try with .yaml extension first, then .yml
		path := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = filepath.Join(dir, name+".yml")
		}

		profile, err := Load(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load profile %q: %w", name, err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// Validate checks that a profile has all required fields
func Validate(p *Profile) error {
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	// Validate Tempo config
	if p.Tempo.Variant == "" {
		return fmt.Errorf("tempo.variant is required (monolithic or stack)")
	}
	if p.Tempo.Variant != "monolithic" && p.Tempo.Variant != "stack" {
		return fmt.Errorf("tempo.variant must be 'monolithic' or 'stack', got %q", p.Tempo.Variant)
	}
	// Resources are optional, but if specified both memory and CPU must be set
	if p.Tempo.Resources != nil {
		if p.Tempo.Resources.Memory == "" && p.Tempo.Resources.CPU != "" {
			return fmt.Errorf("tempo.resources.memory is required when cpu is specified")
		}
		if p.Tempo.Resources.CPU == "" && p.Tempo.Resources.Memory != "" {
			return fmt.Errorf("tempo.resources.cpu is required when memory is specified")
		}
	}

	// Validate K6 config
	if p.K6.Duration == "" {
		return fmt.Errorf("k6.duration is required")
	}
	if p.K6.VUs.Min <= 0 {
		return fmt.Errorf("k6.vus.min must be positive")
	}
	if p.K6.VUs.Max <= 0 {
		return fmt.Errorf("k6.vus.max must be positive")
	}
	if p.K6.VUs.Min > p.K6.VUs.Max {
		return fmt.Errorf("k6.vus.min cannot be greater than k6.vus.max")
	}
	if p.K6.Ingestion.MBPerSecond <= 0 {
		return fmt.Errorf("k6.ingestion.mbPerSecond must be positive")
	}
	if p.K6.Ingestion.TraceProfile == "" {
		return fmt.Errorf("k6.ingestion.traceProfile is required")
	}
	if p.K6.Query.QueriesPerSecond <= 0 {
		return fmt.Errorf("k6.query.queriesPerSecond must be positive")
	}

	return nil
}

// ListProfileNames returns the names of all profiles in a directory
func ListProfileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory %s: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		} else if strings.HasSuffix(name, ".yml") {
			names = append(names, strings.TrimSuffix(name, ".yml"))
		}
	}

	return names, nil
}
