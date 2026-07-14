package dsl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tptconduit/engine"
)

func writeWF(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

const wfTmpl = `
name: signoff
version: %s
initial: approve
steps:
  - name: approve
    kind: approval
    approval:
      chain:
        - role: manager
`

func TestLoaderRegisterAndResolve(t *testing.T) {
	dir := t.TempDir()
	writeWF(t, dir, "signoff.v1.yaml", fmt.Sprintf(wfTmpl, "1.0.0"))
	writeWF(t, dir, "signoff.v2.yaml", fmt.Sprintf(wfTmpl, "1.2.0"))
	writeWF(t, dir, "signoff.v3.yaml", fmt.Sprintf(wfTmpl, "1.10.0"))

	reg, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("load dir: %v", err)
	}
	if len(reg.defs) != 3 {
		t.Fatalf("expected 3 defs, got %d", len(reg.defs))
	}

	// Semantic "latest" must pick 1.10.0, not 1.2.0.
	latest, err := reg.Latest("signoff")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.Version != "1.10.0" {
		t.Fatalf("expected latest 1.10.0, got %s", latest.Version)
	}

	resolved, err := reg.Resolve("signoff", "latest")
	if err != nil || resolved.Version != "1.10.0" {
		t.Fatalf("resolve latest failed: %v %s", err, resolved.Version)
	}
	resolved, err = reg.Resolve("signoff", "1.2.0")
	if err != nil || resolved.Version != "1.2.0" {
		t.Fatalf("resolve 1.2.0 failed: %v %s", err, resolved.Version)
	}

	// Deploy the registry into an engine and confirm it is queryable.
	e := engine.NewEngine(engine.NewInMemoryEventLog(), engine.NewInMemoryStore(), 0)
	if err := reg.RegisterAll(e); err != nil {
		t.Fatalf("register all: %v", err)
	}
	got, err := e.GetWorkflow(context.Background(), "signoff", "1.10.0")
	if err != nil {
		t.Fatalf("engine should have registered 1.10.0: %v", err)
	}
	if got.Name != "signoff" {
		t.Fatalf("unexpected workflow: %+v", got)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.10.0", -1},
		{"1.10.0", "1.2.0", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Fatalf("compareVersions(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}
