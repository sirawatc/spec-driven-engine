# Story: E1-S3 — Multi-Version Spec Registry

**Epic:** E1 — Engine Foundation  
**Status:** Ready for Development  
**Depends on:** E1-S2  
**Blocks:** E2-S1, E6-S1

---

## Summary

The engine must serve multiple spec versions simultaneously. This story adds `SpecRegistry` — a map of version key → `*Spec` — populated by scanning a directory tree at startup. Routes and dashboard pages are all keyed off the registry.

---

## Acceptance Criteria

- [ ] Spec files live under `spec/<version>/` directories (e.g. `spec/v1/`, `spec/v2/`)
- [ ] Engine scans directory from env var `ENGINE_SPEC_DIR` (default: `spec/`)
- [ ] Each subdirectory containing at least one `*.yaml` file is treated as one version
- [ ] All discovered specs are loaded; any single load failure exits with a clear error naming the failing file
- [ ] `SpecRegistry` type: `map[string]*Spec` keyed by directory name (e.g. `"v1"`)
- [ ] Startup log: `loaded spec versions: [v1 v2]` (sorted alphabetically)
- [ ] Unit tests: single version loads, two versions load independently, one bad file causes failure with the file path in the error message

---

## Implementation (`internal/spec/registry.go`)

```go
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
            continue // no yaml files — skip silently
        }

        specPath := yamlFiles[0] // first yaml file in the directory
        spec, err := Load(specPath)
        if err != nil {
            return nil, fmt.Errorf("spec registry: failed to load %s: %w", specPath, err)
        }
        registry[version] = spec
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
```

---

## Helper: Read from Env (`internal/spec/registry.go`, append)

```go
// SpecDirFromEnv returns ENGINE_SPEC_DIR env var, defaulting to "spec/".
func SpecDirFromEnv() string {
    if dir := os.Getenv("ENGINE_SPEC_DIR"); dir != "" {
        return dir
    }
    return "spec"
}
```

---

## Unit Tests (`internal/spec/registry_test.go`)

Test cases required:

1. **Single version** — directory with `v1/spec.yaml` loads `registry["v1"]` correctly
2. **Two versions** — `v1/` and `v2/` both load; registry has two entries with correct names
3. **Bad file** — one valid `v1/` and one invalid `v2/spec.yaml` (bad YAML) → error mentions `v2` path
4. **Empty directory** — no subdirectories → error
5. **Subdirectory with no yaml** — skipped silently; other valid versions still load
6. **Versions sorted in log** — verify `LoadRegistry` produces sorted version list (check log output or sort order)

Use `t.TempDir()` to create real temporary directories for each test case.

---

## Verification

```bash
go test ./internal/spec/...    # all tests pass including registry tests
go build ./...                 # compiles
```
