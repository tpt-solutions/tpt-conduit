// Package dsl provides the YAML workflow definition surface for TPT Conduit.
// Both the YAML and (future) TypeScript DSLs compile to the same engine.WorkflowDef
// internal representation (IR); the engine only ever consumes that IR.
package dsl

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"tptconduit/engine"
)

// Workflow is the YAML authoring shape. Durations are written as Go duration
// strings (e.g. "24h", "30m") for human readability.
type Workflow struct {
	Name        string        `yaml:"name"`
	Version     string        `yaml:"version"`
	Description string        `yaml:"description"`
	Initial     string        `yaml:"initial"`
	Steps       []Step        `yaml:"steps"`
	SLAs        []SLA         `yaml:"slas,omitempty"`
	Routing     []RoutingRule `yaml:"routing,omitempty"`
}

// Step is the YAML step shape.
type Step struct {
	Name     string    `yaml:"name"`
	Kind     string    `yaml:"kind"` // task | approval | delay
	Task     string    `yaml:"task,omitempty"`
	Next     string    `yaml:"next,omitempty"`
	OnError  string    `yaml:"on_error,omitempty"`
	AssignTo string    `yaml:"assign_to,omitempty"`
	Approval *Approval `yaml:"approval,omitempty"`
	Delay    string    `yaml:"delay,omitempty"` // Go duration string
	Retries  int       `yaml:"retries,omitempty"`
	Retry    string    `yaml:"retry_delay,omitempty"`
}

// Approval is a YAML approval chain.
type Approval struct {
	Chain []Approver `yaml:"chain"`
}

// Approver is one link in an approval chain.
type Approver struct {
	Role string `yaml:"role"`
	User string `yaml:"user,omitempty"`
}

// SLA is a YAML SLA definition.
type SLA struct {
	Name     string `yaml:"name"`
	Duration string `yaml:"duration"` // Go duration string
	OnBreach string `yaml:"on_breach,omitempty"`
}

// RoutingRule is the YAML routing shape.
type RoutingRule struct {
	If       map[string]any `yaml:"if"`
	Queue    string         `yaml:"queue"`
	Assignee string         `yaml:"assignee,omitempty"`
	Priority string         `yaml:"priority,omitempty"`
}

// ParseYAML decodes a workflow definition from YAML bytes and compiles it to the
// engine IR, validating as it goes.
func ParseYAML(data []byte) (engine.WorkflowDef, error) {
	var w Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return engine.WorkflowDef{}, fmt.Errorf("dsl: parse yaml: %w", err)
	}
	return w.Compile()
}

// Compile converts the YAML authoring shape into the engine IR.
func (w *Workflow) Compile() (engine.WorkflowDef, error) {
	if w.Name == "" || w.Version == "" {
		return engine.WorkflowDef{}, fmt.Errorf("dsl: name and version are required")
	}
	out := engine.WorkflowDef{
		Name:        w.Name,
		Version:     w.Version,
		Description: w.Description,
		Initial:     w.Initial,
	}
	if len(w.Steps) == 0 {
		return engine.WorkflowDef{}, fmt.Errorf("dsl: workflow %q has no steps", w.Name)
	}
	if out.Initial == "" {
		out.Initial = w.Steps[0].Name
	}
	seen := map[string]bool{}
	for _, s := range w.Steps {
		if s.Name == "" {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: step with empty name")
		}
		if seen[s.Name] {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: duplicate step %q", s.Name)
		}
		seen[s.Name] = true
		sd, err := compileStep(s)
		if err != nil {
			return engine.WorkflowDef{}, err
		}
		out.Steps = append(out.Steps, sd)
	}
	// Validate next/on_error/on_breach references.
	valid := func(name string) bool { _, ok := stepByName(out.Steps, name); return ok }
	for _, s := range out.Steps {
		if s.Next != "" && !valid(s.Next) {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: step %q next=%q not found", s.Name, s.Next)
		}
		if s.OnError != "" && !valid(s.OnError) {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: step %q on_error=%q not found", s.Name, s.OnError)
		}
	}
	for _, sla := range w.SLAs {
		d, err := time.ParseDuration(sla.Duration)
		if err != nil {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: sla %q bad duration: %w", sla.Name, err)
		}
		if sla.OnBreach != "" && !valid(sla.OnBreach) {
			return engine.WorkflowDef{}, fmt.Errorf("dsl: sla %q on_breach=%q not found", sla.Name, sla.OnBreach)
		}
		out.SLAs = append(out.SLAs, engine.SLADef{Name: sla.Name, Duration: d, OnBreach: sla.OnBreach})
	}
	for _, r := range w.Routing {
		out.Routing = append(out.Routing, engine.RoutingRule{If: r.If, Queue: r.Queue, Assignee: r.Assignee, Priority: r.Priority})
	}
	return out, nil
}

func compileStep(s Step) (engine.StepDef, error) {
	var kind engine.StepKind
	switch s.Kind {
	case "task", "":
		kind = engine.KindTask
		if s.Task == "" {
			return engine.StepDef{}, fmt.Errorf("dsl: task step %q requires task", s.Name)
		}
	case "approval":
		kind = engine.KindApproval
		if s.Approval == nil || len(s.Approval.Chain) == 0 {
			return engine.StepDef{}, fmt.Errorf("dsl: approval step %q requires approval.chain", s.Name)
		}
	case "delay":
		kind = engine.KindDelay
		if s.Delay == "" {
			return engine.StepDef{}, fmt.Errorf("dsl: delay step %q requires delay", s.Name)
		}
		d, err := time.ParseDuration(s.Delay)
		if err != nil {
			return engine.StepDef{}, fmt.Errorf("dsl: step %q bad delay: %w", s.Name, err)
		}
		sd := engine.StepDef{Name: s.Name, Kind: kind, Next: s.Next, OnError: s.OnError, AssignTo: s.AssignTo, Delay: &engine.DelayDef{Duration: d}}
		return sd, nil
	default:
		return engine.StepDef{}, fmt.Errorf("dsl: step %q unknown kind %q", s.Name, s.Kind)
	}
	sd := engine.StepDef{
		Name:     s.Name,
		Kind:     kind,
		Task:     s.Task,
		Next:     s.Next,
		OnError:  s.OnError,
		AssignTo: s.AssignTo,
		Retry:    engine.RetryPolicy{MaxAttempts: s.Retries, Delay: parseDelay(s.Retry)},
	}
	if s.Approval != nil {
		chain := make([]engine.Approver, 0, len(s.Approval.Chain))
		for _, a := range s.Approval.Chain {
			chain = append(chain, engine.Approver{Role: a.Role, User: a.User})
		}
		sd.Approval = &engine.ApprovalDef{Chain: chain}
	}
	return sd, nil
}

func parseDelay(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

// StepByName finds a step by name (helper for validation).
func stepByName(steps []engine.StepDef, name string) (engine.StepDef, bool) {
	for _, s := range steps {
		if s.Name == name {
			return s, true
		}
	}
	return engine.StepDef{}, false
}
