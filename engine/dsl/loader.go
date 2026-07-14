package dsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tptconduit/engine"
)

// Registry is an in-memory index of workflow definitions loaded from a
// repository directory (the git-friendly deployment model). The engine only
// ever consumes workflow definitions, so deploying a workflow is just: commit
// its YAML (or TS-compiled JSON) to the repo, load the directory on boot, and
// register each definition. Versions are explicit in the definition metadata;
// "latest" resolves to the highest semantic version of a given workflow name.
type Registry struct {
	defs map[string]engine.WorkflowDef // key: name@version
}

// LoadDir walks dir and parses every *.yaml / *.yml file as a workflow
// definition, returning a Registry. Non-workflow YAML in the tree should be
// kept out of the workflows directory to avoid parse errors.
func LoadDir(dir string) (*Registry, error) {
	r := &Registry{defs: map[string]engine.WorkflowDef{}}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isWorkflowFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		w, err := ParseYAML(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		r.defs[wfKey(w.Name, w.Version)] = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func isWorkflowFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func wfKey(name, version string) string { return name + "@" + version }

// RegisterAll deploys every definition in the registry to the engine.
func (r *Registry) RegisterAll(e *engine.Engine) error {
	for _, w := range r.defs {
		if err := e.RegisterWorkflow(w); err != nil {
			return fmt.Errorf("register %s@%s: %w", w.Name, w.Version, err)
		}
	}
	return nil
}

// Resolve returns the definition for a name and version. A version of "latest"
// or "" resolves to the highest semantic version available for that name.
func (r *Registry) Resolve(name, version string) (engine.WorkflowDef, error) {
	if version == "" || strings.EqualFold(version, "latest") {
		return r.Latest(name)
	}
	if w, ok := r.defs[wfKey(name, version)]; ok {
		return w, nil
	}
	return engine.WorkflowDef{}, fmt.Errorf("workflow %s@%s not found", name, version)
}

// Latest returns the highest semantic-versioned definition for a name.
func (r *Registry) Latest(name string) (engine.WorkflowDef, error) {
	var best engine.WorkflowDef
	bestVer := ""
	found := false
	for _, w := range r.defs {
		if w.Name != name {
			continue
		}
		if !found || compareVersions(w.Version, bestVer) > 0 {
			best = w
			bestVer = w.Version
			found = true
		}
	}
	if !found {
		return engine.WorkflowDef{}, fmt.Errorf("workflow %s not found", name)
	}
	return best, nil
}

// compareVersions does a numeric semantic-version comparison on dotted
// components (e.g. 1.2.10 > 1.10.0). Non-numeric components compare
// lexicographically as a fallback.
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		av, bv := "0", "0"
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		ai, aerr := strconv.Atoi(av)
		bi, berr := strconv.Atoi(bv)
		if aerr == nil && berr == nil {
			if ai != bi {
				if ai < bi {
					return -1
				}
				return 1
			}
			continue
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}
