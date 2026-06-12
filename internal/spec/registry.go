package spec

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// SpecRegistry maps version key (directory name) to its loaded Spec.
type SpecRegistry map[string]*Spec

// LoadRegistry scans dir for subdirectories, loads the first *.yaml in each,
// and returns a populated registry. Returns an error if any spec fails to load.
func LoadRegistry(dir string) (SpecRegistry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("spec registry: read dir %s: %w", dir, err)
	}

	registry := make(SpecRegistry)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		versionDir := filepath.Join(dir, version)

		yamlFiles, err := filepath.Glob(filepath.Join(versionDir, "*.yaml"))
		if err != nil || len(yamlFiles) == 0 {
			continue
		}

		specPath := yamlFiles[0]
		s, err := Load(specPath)
		if err != nil {
			return nil, fmt.Errorf("spec registry: failed to load %s: %w", specPath, err)
		}
		registry[version] = s
	}

	if len(registry) == 0 {
		return nil, fmt.Errorf("spec registry: no valid spec versions found in %s", dir)
	}

	versions := make([]string, 0, len(registry))
	for v := range registry {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	log.Printf("loaded spec versions: %v", versions)

	return registry, nil
}

// SpecDirFromEnv returns ENGINE_SPEC_DIR env var, defaulting to "spec/".
func SpecDirFromEnv() string {
	if dir := os.Getenv("ENGINE_SPEC_DIR"); dir != "" {
		return dir
	}
	return "spec"
}
