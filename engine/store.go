package engine

import (
	"context"
	"errors"
	"sync"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("engine: not found")

// InMemoryStore is a thread-safe, non-durable Store for tests and local dev.
type InMemoryStore struct {
	mu        sync.RWMutex
	tickets   map[string]Ticket
	workflows map[string]map[string]WorkflowDef // name -> version -> def
}

// NewInMemoryStore creates an empty in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tickets:   map[string]Ticket{},
		workflows: map[string]map[string]WorkflowDef{},
	}
}

func (s *InMemoryStore) SaveTicket(ctx context.Context, t Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickets[t.ID] = t
	return nil
}

func (s *InMemoryStore) GetTicket(ctx context.Context, id string) (Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tickets[id]
	if !ok {
		return Ticket{}, ErrNotFound
	}
	return t, nil
}

func (s *InMemoryStore) RegisterWorkflow(ctx context.Context, w WorkflowDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions, ok := s.workflows[w.Name]
	if !ok {
		versions = map[string]WorkflowDef{}
		s.workflows[w.Name] = versions
	}
	versions[w.Version] = w
	return nil
}

func (s *InMemoryStore) GetWorkflow(ctx context.Context, name, version string) (WorkflowDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.workflows[name]
	if !ok {
		return WorkflowDef{}, ErrNotFound
	}
	w, ok := versions[version]
	if !ok {
		return WorkflowDef{}, ErrNotFound
	}
	return w, nil
}

func (s *InMemoryStore) ListTickets(ctx context.Context) ([]Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Ticket, 0, len(s.tickets))
	for _, t := range s.tickets {
		out = append(out, t)
	}
	return out, nil
}

func (s *InMemoryStore) ListWorkflows(ctx context.Context) ([]WorkflowDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]WorkflowDef, 0)
	for _, versions := range s.workflows {
		for _, w := range versions {
			out = append(out, w)
		}
	}
	return out, nil
}

func (s *InMemoryStore) WorkflowVersions(ctx context.Context, name string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.workflows[name]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]string, 0, len(versions))
	for v := range versions {
		out = append(out, v)
	}
	return out, nil
}
