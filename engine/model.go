// Package engine implements the durable, fault-tolerant workflow execution
// core of TPT Conduit. It is inspired by Temporal: workflow state is never
// stored directly, instead it is derived by replaying an append-only event
// log. This makes the engine crash-safe and deterministic.
package engine

import (
	"encoding/json"
	"time"
)

// RunStatus is the high-level lifecycle status of a workflow run.
type RunStatus string

const (
	RunStatusActive    RunStatus = "ACTIVE"
	RunStatusWaiting   RunStatus = "WAITING" // parked on a timer or human approval
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusCancelled RunStatus = "CANCELLED"
)

// StepStatus is the per-step status within a run.
type StepStatus string

const (
	StepPending   StepStatus = "PENDING"
	StepRunning   StepStatus = "RUNNING"
	StepCompleted StepStatus = "COMPLETED"
	StepFailed    StepStatus = "FAILED"
	StepWaiting   StepStatus = "WAITING"
	StepSkipped   StepStatus = "SKIPPED"
)

// StepKind enumerates the primitive building blocks of a workflow.
type StepKind string

const (
	// KindTask is an automated/worker-executed task.
	KindTask StepKind = "task"
	// KindApproval is a multi-step human-in-the-loop approval chain.
	KindApproval StepKind = "approval"
	// KindDelay is a durable timer that pauses the run for a duration.
	KindDelay StepKind = "delay"
)

// Ticket is the generic, domain-agnostic unit of work. All structured data
// lives in Fields so that one engine serves helpdesk, HR, asset tracking, etc.
type Ticket struct {
	ID          string         `json:"id"`
	Workflow    string         `json:"workflow"`
	WorkflowVer string         `json:"workflow_version"`
	Title       string         `json:"title"`
	Fields      map[string]any `json:"fields"`
	Assignee    string         `json:"assignee"`
	Queue       string         `json:"queue"`
	Priority    string         `json:"priority"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// RetryPolicy governs automatic retries of an automated task step.
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts,omitempty"` // total attempts (1 = no retry)
	Delay       time.Duration `json:"delay,omitempty"`        // backoff between attempts
}

// StepDef is the static declaration of a single workflow step.
type StepDef struct {
	Name     string       `json:"name"`
	Kind     StepKind     `json:"kind"`
	Task     string       `json:"task,omitempty"`     // handler key for KindTask
	Approval *ApprovalDef `json:"approval,omitempty"` // for KindApproval
	Delay    *DelayDef    `json:"delay,omitempty"`    // for KindDelay
	Next     string       `json:"next,omitempty"`     // step to run on success
	OnError  string       `json:"on_error,omitempty"` // step to run on failure
	AssignTo string       `json:"assign_to,omitempty"`
	Retry    RetryPolicy  `json:"retry,omitempty"`
}

// ApprovalDef describes a sequential chain of human approvers.
type ApprovalDef struct {
	Chain []Approver `json:"chain"`
}

// Approver identifies a single link in an approval chain.
type Approver struct {
	Role string `json:"role"`
	User string `json:"user,omitempty"`
}

// DelayDef is a durable timer configuration.
type DelayDef struct {
	Duration time.Duration `json:"duration"`
}

// SLADef escalates (or flags) a run if it is not completed within a window.
type SLADef struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	OnBreach string        `json:"on_breach,omitempty"` // step to jump to on breach
}

// RoutingRule assigns a ticket to a queue/assignee based on field values.
type RoutingRule struct {
	If       map[string]any `json:"if"`    // field equality match
	Queue    string         `json:"queue"` // assign to queue
	Assignee string         `json:"assignee,omitempty"`
	Priority string         `json:"priority,omitempty"`
}

// WorkflowDef is the shared internal representation (IR) that both the YAML
// and TypeScript DSLs compile to. The engine consumes only this.
type WorkflowDef struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Initial     string        `json:"initial"`
	Steps       []StepDef     `json:"steps"`
	SLAs        []SLADef      `json:"slas,omitempty"`
	Routing     []RoutingRule `json:"routing,omitempty"`
}

// stepMap indexes steps by name for O(1) lookups.
func (w *WorkflowDef) stepMap() map[string]StepDef {
	m := make(map[string]StepDef, len(w.Steps))
	for _, s := range w.Steps {
		m[s.Name] = s
	}
	return m
}

func (w *WorkflowDef) Step(name string) (StepDef, bool) {
	for _, s := range w.Steps {
		if s.Name == name {
			return s, true
		}
	}
	return StepDef{}, false
}

// Event is a single immutable fact appended to a run's history log.
type Event struct {
	ID       int64           `json:"id"` // monotonic sequence within run
	RunID    string          `json:"run_id"`
	TicketID string          `json:"ticket_id"`
	Type     EventType       `json:"type"`
	Seq      int64           `json:"seq"`
	At       time.Time       `json:"at"`
	Payload  json.RawMessage `json:"payload,omitempty"`
	// ScheduleAt, when set, marks a durable timer that fires at that time.
	ScheduleAt *time.Time `json:"schedule_at,omitempty"`
}

// EventType enumerates every kind of fact the engine can record.
type EventType string

const (
	EventTicketCreated     EventType = "TICKET_CREATED"
	EventRunStarted        EventType = "RUN_STARTED"
	EventStepScheduled     EventType = "STEP_SCHEDULED"
	EventStepStarted       EventType = "STEP_STARTED"
	EventStepCompleted     EventType = "STEP_COMPLETED"
	EventStepFailed        EventType = "STEP_FAILED"
	EventTimerFired        EventType = "TIMER_FIRED"
	EventApprovalRequested EventType = "APPROVAL_REQUESTED"
	EventApprovalGranted   EventType = "APPROVAL_GRANTED"
	EventApprovalRejected  EventType = "APPROVAL_REJECTED"
	EventSLABreached       EventType = "SLA_BREACHED"
	EventTicketAssigned    EventType = "TICKET_ASSIGNED"
	EventRunCompleted      EventType = "RUN_COMPLETED"
	EventRunFailed         EventType = "RUN_FAILED"
	EventRunCancelled      EventType = "RUN_CANCELLED"
	EventRunFailing        EventType = "RUN_FAILING"
)

// --- Event payloads (kept as plain structs, serialized into Event.Payload) ---

type TicketCreatedPayload struct {
	Ticket Ticket `json:"ticket"`
}

type RunStartedPayload struct {
	RunID     string `json:"run_id"`
	FirstStep string `json:"first_step"`
}

type StepScheduledPayload struct {
	Step  string    `json:"step"`
	Kind  StepKind  `json:"kind"`
	DueAt time.Time `json:"due_at,omitempty"`
}

type StepStartedPayload struct {
	Step string `json:"step"`
}

type StepCompletedPayload struct {
	Step   string         `json:"step"`
	Output map[string]any `json:"output,omitempty"`
}

type StepFailedPayload struct {
	Step    string `json:"step"`
	Reason  string `json:"reason"`
	Attempt int    `json:"attempt"`
}

type TimerFiredPayload struct {
	Step string `json:"step"`
}

type ApprovalRequestedPayload struct {
	Step     string `json:"step"`
	Index    int    `json:"index"` // which link in the chain
	Role     string `json:"role"`
	User     string `json:"user,omitempty"`
	ChainLen int    `json:"chain_len"`
}

type ApprovalGrantedPayload struct {
	Step  string `json:"step"`
	Index int    `json:"index"`
	By    string `json:"by"`
}

type ApprovalRejectedPayload struct {
	Step   string `json:"step"`
	Index  int    `json:"index"`
	By     string `json:"by"`
	Reason string `json:"reason,omitempty"`
}

type SLABreachedPayload struct {
	Name string `json:"name"`
	Step string `json:"step,omitempty"`
}

type TicketAssignedPayload struct {
	Queue    string `json:"queue"`
	Assignee string `json:"assignee,omitempty"`
	Priority string `json:"priority,omitempty"`
	Rule     int    `json:"rule"`
}

type RunCompletedPayload struct {
	LastStep string `json:"last_step"`
}

type RunFailedPayload struct {
	LastStep string `json:"last_step"`
	Reason   string `json:"reason,omitempty"`
}

type RunCancelledPayload struct {
	Reason string `json:"reason,omitempty"`
}

// StepState is the derived (replayed) state of a single step.
type StepState struct {
	Name     string         `json:"name"`
	Kind     StepKind       `json:"kind"`
	Status   StepStatus     `json:"status"`
	Attempt  int            `json:"attempt"`
	Output   map[string]any `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	DueAt    *time.Time     `json:"due_at,omitempty"`
	Approval *ApprovalState `json:"approval,omitempty"`
}

// ApprovalState tracks progress through an approval chain.
type ApprovalState struct {
	Chain     []Approver `json:"chain"`
	Index     int        `json:"index"`
	Status    string     `json:"status"` // pending | granted | rejected
	DecidedBy string     `json:"decided_by,omitempty"`
}

// WorkflowRun is the derived (replayed) state of a workflow execution.
type WorkflowRun struct {
	ID          string                `json:"id"`
	TicketID    string                `json:"ticket_id"`
	Workflow    string                `json:"workflow"`
	WorkflowVer string                `json:"workflow_version"`
	Status      RunStatus             `json:"status"`
	Current     string                `json:"current_step"`
	Steps       map[string]*StepState `json:"steps"`
	Output      map[string]any        `json:"output,omitempty"`
	Failed      bool                  `json:"failed,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}
