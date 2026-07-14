package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TaskHandler executes an automated task step. It must be deterministic with
// respect to its inputs (ticket fields + prior step outputs) so that replays
// are reproducible. Side effects should be performed via the engine, not here.
type TaskHandler func(ctx context.Context, run *WorkflowRun, step string, fields map[string]any) (map[string]any, error)

// Engine is the durable workflow execution core. It is safe for concurrent use.
type Engine struct {
	log    EventLog
	store  Store
	router *Router

	mu        sync.RWMutex
	workflows map[string]WorkflowDef // key: name@version
	handlers  map[string]TaskHandler

	pool     *WorkerPool
	now      func() time.Time
	closed   bool
	closeCh  chan struct{}
	once     sync.Once
	tick     time.Duration
}

// NewEngine constructs an engine with the given event log and store. The
// WorkerPool is started with the provided number of workers; pass 0 to run
// without an internal worker (e.g. for tests that drive steps manually).
func NewEngine(log EventLog, store Store, workers int) *Engine {
	e := &Engine{
		log:       log,
		store:     store,
		router:    NewRouter(),
		workflows: map[string]WorkflowDef{},
		handlers:  map[string]TaskHandler{},
		now:       time.Now,
		closeCh:   make(chan struct{}),
		tick:     100 * time.Millisecond,
	}
	e.pool = NewWorkerPool(workers, e.executeTask)
	go e.timerLoop()
	return e
}

// SetClock overrides the engine's notion of "now". It exists primarily for
// deterministic tests (timers, SLA deadlines) and should not be called
// concurrently with execution.
func (e *Engine) SetClock(fn func() time.Time) {
	e.mu.Lock()
	e.now = fn
	e.mu.Unlock()
}

func wfKey(name, version string) string { return name + "@" + version }

// RegisterWorkflow makes a workflow definition available to the engine.
func (e *Engine) RegisterWorkflow(def WorkflowDef) error {
	if def.Name == "" || def.Version == "" {
		return fmt.Errorf("workflow name and version are required")
	}
	if def.Initial == "" && len(def.Steps) > 0 {
		def.Initial = def.Steps[0].Name
	}
	if _, ok := def.Step(def.Initial); !ok {
		return fmt.Errorf("initial step %q not found in workflow %s", def.Initial, def.Name)
	}
	e.mu.Lock()
	e.workflows[wfKey(def.Name, def.Version)] = def
	e.mu.Unlock()
	return e.store.RegisterWorkflow(context.Background(), def)
}

// RegisterTask binds a TaskHandler to a task key used by task steps.
func (e *Engine) RegisterTask(key string, h TaskHandler) {
	e.mu.Lock()
	e.handlers[key] = h
	e.mu.Unlock()
}

func (e *Engine) workflow(name, version string) (WorkflowDef, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	w, ok := e.workflows[wfKey(name, version)]
	if ok {
		return w, true
	}
	// Fall back to store (e.g. after a restart).
	return WorkflowDef{}, false
}

func (e *Engine) loadWorkflow(ctx context.Context, name, version string) (WorkflowDef, error) {
	if w, ok := e.workflow(name, version); ok {
		return w, nil
	}
	w, err := e.store.GetWorkflow(ctx, name, version)
	if err != nil {
		return WorkflowDef{}, err
	}
	e.mu.Lock()
	e.workflows[wfKey(name, version)] = w
	e.mu.Unlock()
	return w, nil
}

// CreateTicket creates a ticket, starts a workflow run, and schedules the first
// step. It is fully atomic from the caller's perspective: every effect is an
// append-only event plus a single ticket row.
func (e *Engine) CreateTicket(ctx context.Context, workflowName, version, title string, fields map[string]any) (*Ticket, *WorkflowRun, error) {
	def, err := e.loadWorkflow(ctx, workflowName, version)
	if err != nil {
		return nil, nil, fmt.Errorf("load workflow: %w", err)
	}
	if fields == nil {
		fields = map[string]any{}
	}
	now := e.now().UTC()
	id := newID("tkt")
	t := Ticket{
		ID:          id,
		Workflow:    def.Name,
		WorkflowVer: def.Version,
		Title:       title,
		Fields:      fields,
		Priority:    "normal",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// Rule-based routing.
	e.router.Apply(&t, def.Routing)

	runID := newID("run")
	if err := e.store.SaveTicket(ctx, t); err != nil {
		return nil, nil, fmt.Errorf("save ticket: %w", err)
	}
	// Emit ticket-created (history) and start the run.
	if _, err := e.append(ctx, Event{TicketID: id, RunID: runID, Type: EventTicketCreated, Payload: mustJSON(TicketCreatedPayload{Ticket: t})}); err != nil {
		return nil, nil, err
	}
	if _, err := e.append(ctx, Event{TicketID: id, RunID: runID, Type: EventRunStarted, Payload: mustJSON(RunStartedPayload{RunID: runID, FirstStep: def.Initial})}); err != nil {
		return nil, nil, err
	}
	// Schedule first step.
	if _, err := e.scheduleStep(ctx, runID, id, def, def.Initial); err != nil {
		return nil, nil, err
	}
	// Persist routing result as an event too (for timeline clarity).
	if t.Assignee != "" || t.Queue != "" || t.Priority != "" {
		if _, err := e.append(ctx, Event{TicketID: id, RunID: runID, Type: EventTicketAssigned,
			Payload: mustJSON(TicketAssignedPayload{Queue: t.Queue, Assignee: t.Assignee, Priority: t.Priority})}); err != nil {
			return nil, nil, err
		}
	}
	run, err := e.GetRun(ctx, runID)
	if err != nil {
		return nil, nil, err
	}
	e.dispatch(ctx, run, def)
	return &t, run, nil
}

// append writes an event to the log and returns the persisted copy.
func (e *Engine) append(ctx context.Context, ev Event) (Event, error) {
	return e.log.Append(ctx, ev)
}

// scheduleStep appends a STEP_SCHEDULED event for the named step, configuring
// any timer/approval scaffolding the step needs.
func (e *Engine) scheduleStep(ctx context.Context, runID, ticketID string, def WorkflowDef, stepName string) (Event, error) {
	sd, ok := def.Step(stepName)
	if !ok {
		return Event{}, fmt.Errorf("step %q not found", stepName)
	}
	now := e.now()
	payload := StepScheduledPayload{Step: stepName, Kind: sd.Kind}
	ev := Event{TicketID: ticketID, RunID: runID, Type: EventStepScheduled, At: now, Payload: mustJSON(payload)}

	switch sd.Kind {
	case KindDelay:
		if sd.Delay == nil {
			return Event{}, fmt.Errorf("delay step %q missing delay config", stepName)
		}
		due := now.Add(sd.Delay.Duration)
		ev.ScheduleAt = &due
		payload.DueAt = due
		ev.Payload = mustJSON(payload)
	case KindApproval:
		if sd.Approval == nil || len(sd.Approval.Chain) == 0 {
			return Event{}, fmt.Errorf("approval step %q missing chain", stepName)
		}
		// The first link is requested immediately on scheduling.
		first := sd.Approval.Chain[0]
		if _, err := e.append(ctx, Event{TicketID: ticketID, RunID: runID, Type: EventApprovalRequested,
			Payload: mustJSON(ApprovalRequestedPayload{Step: stepName, Index: 0, Role: first.Role, User: first.User, ChainLen: len(sd.Approval.Chain)})}); err != nil {
			return Event{}, err
		}
	}
	return e.append(ctx, ev)
}

// GetRun replays the full event history for a run and returns derived state.
func (e *Engine) GetRun(ctx context.Context, runID string) (*WorkflowRun, error) {
	events, err := e.log.History(ctx, runID)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, ErrNotFound
	}
	ticketID := events[0].TicketID
	def, err := e.defForRun(ctx, events)
	if err != nil {
		return nil, err
	}
	run := &WorkflowRun{
		ID:       runID,
		TicketID: ticketID,
		Steps:    map[string]*StepState{},
	}
	e.applyAll(run, def, events)
	return run, nil
}

// defForRun resolves the workflow definition referenced by the first events.
func (e *Engine) defForRun(ctx context.Context, events []Event) (WorkflowDef, error) {
	// Derive workflow name/version from the ticket if possible.
	t, err := e.store.GetTicket(ctx, events[0].TicketID)
	if err == nil {
		return e.loadWorkflow(ctx, t.Workflow, t.WorkflowVer)
	}
	return WorkflowDef{}, ErrNotFound
}

// applyAll replays a slice of events onto the run state.
func (e *Engine) applyAll(run *WorkflowRun, def WorkflowDef, events []Event) {
	for _, ev := range events {
		e.applyOne(run, def, ev)
	}
}

// applyOne mutates run state for a single event. It is pure: given the same
// event stream it always produces the same state.
func (e *Engine) applyOne(run *WorkflowRun, def WorkflowDef, ev Event) {
	switch ev.Type {
	case EventTicketCreated:
		var p TicketCreatedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		run.TicketID = p.Ticket.ID
		run.Workflow = p.Ticket.Workflow
		run.WorkflowVer = p.Ticket.WorkflowVer
		if run.CreatedAt.IsZero() || ev.At.Before(run.CreatedAt) {
			run.CreatedAt = ev.At
		}
	case EventRunStarted:
		var p RunStartedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		run.Status = RunStatusActive
		run.Current = p.FirstStep
		run.ensureStep(p.FirstStep, def, StepPending)
	case EventStepScheduled:
		var p StepScheduledPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepPending)
		switch p.Kind {
		case KindApproval:
			st.Status = StepWaiting
		case KindDelay:
			st.Status = StepWaiting
		default:
			st.Status = StepPending
		}
		if p.DueAt != (time.Time{}) {
			d := p.DueAt
			st.DueAt = &d
		}
	case EventStepStarted:
		var p StepStartedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepRunning)
		st.Status = StepRunning
		st.Attempt++
	case EventStepCompleted:
		var p StepCompletedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepCompleted)
		st.Status = StepCompleted
		if p.Output != nil {
			st.Output = p.Output
		}
		if run.Output == nil {
			run.Output = map[string]any{}
		}
		for k, v := range p.Output {
			run.Output[k] = v
		}
	case EventStepFailed:
		var p StepFailedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepFailed)
		st.Status = StepFailed
		st.Error = p.Reason
		st.Attempt = p.Attempt
	case EventTimerFired:
		// handled by completion transition; no separate state needed.
	case EventApprovalRequested:
		var p ApprovalRequestedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepWaiting)
		if st.Approval == nil {
			chain := def.approvalChain(p.Step)
			st.Approval = &ApprovalState{Chain: chain, Index: p.Index, Status: "pending"}
		} else {
			st.Approval.Index = p.Index
			st.Approval.Status = "pending"
		}
		st.Status = StepWaiting
	case EventApprovalGranted:
		var p ApprovalGrantedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepWaiting)
		if st.Approval != nil {
			st.Approval.Status = "granted"
			st.Approval.DecidedBy = p.By
		}
	case EventApprovalRejected:
		var p ApprovalRejectedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		st := run.ensureStep(p.Step, def, StepWaiting)
		if st.Approval != nil {
			st.Approval.Status = "rejected"
			st.Approval.DecidedBy = p.By
		}
		st.Status = StepFailed
		st.Error = "rejected: " + p.Reason
	case EventSLABreached:
		// informational; state change handled by transition logic.
	case EventTicketAssigned:
		// informational for timeline.
	case EventRunCompleted:
		run.Status = RunStatusCompleted
	case EventRunFailed:
		run.Status = RunStatusFailed
	case EventRunCancelled:
		run.Status = RunStatusCancelled
	}
	if ev.At.After(run.UpdatedAt) {
		run.UpdatedAt = ev.At
	}
}

func (w WorkflowDef) approvalChain(step string) []Approver {
	if sd, ok := w.Step(step); ok && sd.Approval != nil {
		return sd.Approval.Chain
	}
	return nil
}

// ensureStep returns the step state, creating it with the given status if new.
func (r *WorkflowRun) ensureStep(name string, def WorkflowDef, status StepStatus) *StepState {
	if r.Steps == nil {
		r.Steps = map[string]*StepState{}
	}
	st, ok := r.Steps[name]
	if !ok {
		kind := KindTask
		if sd, ok := def.Step(name); ok {
			kind = sd.Kind
		}
		st = &StepState{Name: name, Kind: kind, Status: status}
		r.Steps[name] = st
	}
	return st
}

// dispatch inspects current run state and enqueues any work that is ready:
// pending automated tasks go to the worker, due timers/approvals/SLA are fired.
func (e *Engine) dispatch(ctx context.Context, run *WorkflowRun, def WorkflowDef) {
	if run.Status == RunStatusCompleted || run.Status == RunStatusFailed || run.Status == RunStatusCancelled {
		return
	}
	e.CheckSLA(ctx, run, def)
	for name, st := range run.Steps {
		switch st.Kind {
		case KindTask:
			if st.Status == StepPending {
				e.pool.Submit(taskItem{RunID: run.ID, Step: name, TicketID: run.TicketID})
			}
		case KindDelay:
			if st.Status == StepWaiting && st.DueAt != nil && !st.DueAt.After(e.now()) {
				e.fireTimer(ctx, run, def, name)
			}
		}
	}
	// Approval steps waiting on a granted link: advance if last link granted.
	for name, st := range run.Steps {
		if st.Kind == KindApproval && st.Status == StepWaiting && st.Approval != nil {
			e.advanceApproval(ctx, run, def, name)
		}
	}
}

// executeTask is the WorkerPool handler: it runs a task handler and records the
// result, then advances the workflow.
func (e *Engine) executeTask(ctx context.Context, t taskItem) error {
	run, err := e.GetRun(ctx, t.RunID)
	if err != nil {
		return err
	}
	def, err := e.loadWorkflow(ctx, run.Workflow, run.WorkflowVer)
	if err != nil {
		return err
	}
	sd, ok := def.Step(t.Step)
	if !ok {
		return fmt.Errorf("step %q not found", t.Step)
	}
	ticket, err := e.store.GetTicket(ctx, run.TicketID)
	if err != nil {
		return err
	}
	h, ok := e.handler(sd.Task)
	if !ok {
		// No handler registered: fail the step.
		return e.failStep(ctx, run, def, t.Step, fmt.Sprintf("no handler for task %q", sd.Task))
	}
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepStarted, Payload: mustJSON(StepStartedPayload{Step: t.Step})}); err != nil {
		return err
	}
	out, herr := h(ctx, run, t.Step, ticket.Fields)
	if herr != nil {
		return e.handleTaskFailure(ctx, run, def, t.Step, sd, herr)
	}
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepCompleted, Payload: mustJSON(StepCompletedPayload{Step: t.Step, Output: out})}); err != nil {
		return err
	}
	return e.advance(ctx, run, def, t.Step, true)
}

func (e *Engine) handler(key string) (TaskHandler, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	h, ok := e.handlers[key]
	return h, ok
}

// handleTaskFailure applies the step retry policy or fails the step.
func (e *Engine) handleTaskFailure(ctx context.Context, run *WorkflowRun, def WorkflowDef, step string, sd StepDef, herr error) error {
	st := run.Steps[step]
	maxAttempts := sd.Retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if st.Attempt < maxAttempts {
		delay := sd.Retry.Delay
		if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepFailed,
			Payload: mustJSON(StepFailedPayload{Step: step, Reason: herr.Error(), Attempt: st.Attempt})}); err != nil {
			return err
		}
		// Reschedule with a timer for the backoff (durable retry).
		now := e.now()
		due := now.Add(delay)
		if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepScheduled,
			At: now, ScheduleAt: &due,
			Payload: mustJSON(StepScheduledPayload{Step: step, Kind: KindTask, DueAt: due})}); err != nil {
			return err
		}
		return nil
	}
	return e.failStep(ctx, run, def, step, herr.Error())
}

func (e *Engine) failStep(ctx context.Context, run *WorkflowRun, def WorkflowDef, step, reason string) error {
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepFailed,
		Payload: mustJSON(StepFailedPayload{Step: step, Reason: reason, Attempt: run.Steps[step].Attempt})}); err != nil {
		return err
	}
	return e.advance(ctx, run, def, step, false)
}

// advance moves the workflow to the next step (or completes/fails it) after a
// step finishes. succeeded selects Next vs OnError.
func (e *Engine) advance(ctx context.Context, run *WorkflowRun, def WorkflowDef, completedStep string, succeeded bool) error {
	sd, ok := def.Step(completedStep)
	var next string
	if ok {
		if succeeded {
			next = sd.Next
		} else {
			next = sd.OnError
		}
	}
	if next == "" {
		// No further step: the run terminates.
		if succeeded {
			_, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventRunCompleted,
				Payload: mustJSON(RunCompletedPayload{LastStep: completedStep})})
			return err
		}
		_, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventRunFailed,
			Payload: mustJSON(RunFailedPayload{LastStep: completedStep, Reason: run.Steps[completedStep].Error})})
		return err
	}
	if _, err := e.scheduleStep(ctx, run.ID, run.TicketID, def, next); err != nil {
		return err
	}
	nr, err := e.GetRun(ctx, run.ID)
	if err == nil {
		e.dispatch(ctx, nr, def)
	}
	return err
}

// fireTimer records a delay step's timer firing and completes the step.
func (e *Engine) fireTimer(ctx context.Context, run *WorkflowRun, def WorkflowDef, step string) error {
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventTimerFired,
		Payload: mustJSON(TimerFiredPayload{Step: step})}); err != nil {
		return err
	}
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepCompleted,
		Payload: mustJSON(StepCompletedPayload{Step: step})}); err != nil {
		return err
	}
	return e.advance(ctx, run, def, step, true)
}

// advanceApproval progresses an approval chain after the current link is
// granted, requesting the next link or completing the step.
func (e *Engine) advanceApproval(ctx context.Context, run *WorkflowRun, def WorkflowDef, step string) error {
	st := run.Steps[step]
	if st.Approval == nil || st.Approval.Status != "granted" {
		return nil
	}
	if st.Approval.Index+1 < len(st.Approval.Chain) {
		next := st.Approval.Index + 1
		link := st.Approval.Chain[next]
		if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventApprovalRequested,
			Payload: mustJSON(ApprovalRequestedPayload{Step: step, Index: next, Role: link.Role, User: link.User, ChainLen: len(st.Approval.Chain)})}); err != nil {
			return err
		}
		return nil
	}
	// All links granted.
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventStepCompleted,
		Payload: mustJSON(StepCompletedPayload{Step: step})}); err != nil {
		return err
	}
	return e.advance(ctx, run, def, step, true)
}

// Approve records an approval decision for the currently-pending link.
func (e *Engine) Approve(ctx context.Context, runID, step, by string, comment string) error {
	run, err := e.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	def, err := e.loadWorkflow(ctx, run.Workflow, run.WorkflowVer)
	if err != nil {
		return err
	}
	st, ok := run.Steps[step]
	if !ok || st.Approval == nil {
		return fmt.Errorf("step %q is not a pending approval", step)
	}
	if st.Approval.Status != "pending" {
		return fmt.Errorf("approval %q is not awaiting a decision (status=%s)", step, st.Approval.Status)
	}
	idx := st.Approval.Index
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventApprovalGranted,
		Payload: mustJSON(ApprovalGrantedPayload{Step: step, Index: idx, By: by})}); err != nil {
		return err
	}
	nr, err := e.GetRun(ctx, runID)
	if err == nil {
		e.advanceApproval(ctx, nr, def, step)
		// Re-dispatch in case a subsequent task step is now ready.
		e.dispatchAfter(ctx, nr, def)
	}
	return nil
}

// Reject rejects the current approval link, failing the step.
func (e *Engine) Reject(ctx context.Context, runID, step, by, reason string) error {
	run, err := e.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	def, err := e.loadWorkflow(ctx, run.Workflow, run.WorkflowVer)
	if err != nil {
		return err
	}
	st, ok := run.Steps[step]
	if !ok || st.Approval == nil {
		return fmt.Errorf("step %q is not a pending approval", step)
	}
	if st.Approval.Status != "pending" {
		return fmt.Errorf("approval %q is not awaiting a decision", step)
	}
	if _, err := e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventApprovalRejected,
		Payload: mustJSON(ApprovalRejectedPayload{Step: step, Index: st.Approval.Index, By: by, Reason: reason})}); err != nil {
		return err
	}
	return e.advance(ctx, run, def, step, false)
}

// Cancel terminates a run.
func (e *Engine) Cancel(ctx context.Context, runID, reason string) error {
	run, err := e.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.Status == RunStatusCompleted || run.Status == RunStatusFailed || run.Status == RunStatusCancelled {
		return nil
	}
	_, err = e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventRunCancelled,
		Payload: mustJSON(RunCancelledPayload{Reason: reason})})
	return err
}

// dispatchAfter re-runs dispatch without re-firing approvals (used after a
// human decision unblocks downstream task steps).
func (e *Engine) dispatchAfter(ctx context.Context, run *WorkflowRun, def WorkflowDef) {
	if run.Status == RunStatusCompleted || run.Status == RunStatusFailed || run.Status == RunStatusCancelled {
		return
	}
	for name, st := range run.Steps {
		if st.Kind == KindTask && st.Status == StepPending {
			e.pool.Submit(taskItem{RunID: run.ID, Step: name, TicketID: run.TicketID})
		}
		if st.Kind == KindDelay && st.Status == StepWaiting && st.DueAt != nil && !st.DueAt.After(e.now()) {
			e.fireTimer(ctx, run, def, name)
		}
	}
}

// Recover replays every run and re-arms pending work after a process restart.
// This is what makes execution durable: no in-memory queue survives a crash,
// but the event log lets us reconstruct and resume exactly where we left off.
func (e *Engine) Recover(ctx context.Context) error {
	runIDs, err := e.log.Runs(ctx)
	if err != nil {
		return err
	}
	for _, id := range runIDs {
		run, err := e.GetRun(ctx, id)
		if err != nil {
			return err
		}
		def, err := e.loadWorkflow(ctx, run.Workflow, run.WorkflowVer)
		if err != nil {
			// Workflow definition missing (e.g. not yet deployed); skip.
			continue
		}
		e.dispatch(ctx, run, def)
	}
	return nil
}

// Close stops the worker pool.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()
	return e.pool.Stop()
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func newID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback (extremely unlikely).
		return prefix + hex.EncodeToString([]byte{byte(time.Now().UnixNano())})
	}
	return prefix + "_" + hex.EncodeToString(b)
}
