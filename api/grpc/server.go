// Package grpc exposes the TPT Conduit engine over gRPC, mirroring the core
// mutations and read APIs for high-performance, service-to-service use.
package grpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"tptconduit/api/proto"
	"tptconduit/engine"
)

// Server implements proto.ConduitServer on top of the durable engine.
type Server struct {
	proto.UnimplementedConduitServer
	engine *engine.Engine
}

// NewServer wraps an engine in a gRPC server.
func NewServer(e *engine.Engine) *Server { return &Server{engine: e} }

// RegisterOn attaches the service to a gRPC server.
func (s *Server) RegisterOn(gs *grpc.Server) {
	proto.RegisterConduitServer(gs, s)
}

func (s *Server) CreateTicket(ctx context.Context, req *proto.CreateTicketRequest) (*proto.CreateTicketResponse, error) {
	if req.Workflow == "" || req.Version == "" || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow, version and title are required")
	}
	fields := map[string]any{}
	if req.FieldsJson != "" {
		if err := json.Unmarshal([]byte(req.FieldsJson), &fields); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "fields_json is not valid JSON: %v", err)
		}
	}
	t, _, err := s.engine.CreateTicket(ctx, req.Workflow, req.Version, req.Title, fields)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "create ticket: %v", err)
	}
	return &proto.CreateTicketResponse{Ticket: toProtoTicket(t)}, nil
}

func (s *Server) Approve(ctx context.Context, req *proto.ApproveRequest) (*proto.Empty, error) {
	if err := s.engine.Approve(ctx, req.RunId, req.Step, req.By, req.Comment); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "approve: %v", err)
	}
	return &proto.Empty{}, nil
}

func (s *Server) Reject(ctx context.Context, req *proto.RejectRequest) (*proto.Empty, error) {
	if err := s.engine.Reject(ctx, req.RunId, req.Step, req.By, req.Reason); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "reject: %v", err)
	}
	return &proto.Empty{}, nil
}

func (s *Server) Cancel(ctx context.Context, req *proto.CancelRequest) (*proto.Empty, error) {
	if err := s.engine.Cancel(ctx, req.RunId, req.Reason); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cancel: %v", err)
	}
	return &proto.Empty{}, nil
}

func (s *Server) GetTicket(ctx context.Context, req *proto.GetTicketRequest) (*proto.Ticket, error) {
	t, err := s.engine.GetTicket(ctx, req.Id)
	if err != nil {
		if err == engine.ErrNotFound {
			return nil, status.Error(codes.NotFound, "ticket not found")
		}
		return nil, status.Errorf(codes.Internal, "get ticket: %v", err)
	}
	return toProtoTicket(&t), nil
}

func (s *Server) ListTickets(ctx context.Context, _ *proto.ListTicketsRequest) (*proto.ListTicketsResponse, error) {
	tickets, err := s.engine.ListTickets(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tickets: %v", err)
	}
	out := &proto.ListTicketsResponse{}
	for i := range tickets {
		out.Tickets = append(out.Tickets, toProtoTicket(&tickets[i]))
	}
	return out, nil
}

func (s *Server) GetRun(ctx context.Context, req *proto.GetRunRequest) (*proto.GetRunResponse, error) {
	run, err := s.engine.GetRun(ctx, req.Id)
	if err != nil {
		if err == engine.ErrNotFound {
			return nil, status.Error(codes.NotFound, "run not found")
		}
		return nil, status.Errorf(codes.Internal, "get run: %v", err)
	}
	return &proto.GetRunResponse{Run: toProtoRun(run)}, nil
}

func (s *Server) ListRuns(ctx context.Context, _ *proto.ListRunsRequest) (*proto.ListRunsResponse, error) {
	ids, err := s.engine.ListRuns(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list runs: %v", err)
	}
	out := &proto.ListRunsResponse{}
	for _, id := range ids {
		run, err := s.engine.GetRun(ctx, id)
		if err != nil {
			continue
		}
		out.Runs = append(out.Runs, toProtoRun(run))
	}
	return out, nil
}

func (s *Server) GetWorkflow(ctx context.Context, req *proto.GetWorkflowRequest) (*proto.WorkflowResponse, error) {
	w, err := s.engine.GetWorkflow(ctx, req.Name, req.Version)
	if err != nil {
		if err == engine.ErrNotFound {
			return nil, status.Error(codes.NotFound, "workflow not found")
		}
		return nil, status.Errorf(codes.Internal, "get workflow: %v", err)
	}
	return &proto.WorkflowResponse{Workflow: toProtoWorkflow(w)}, nil
}

func (s *Server) ListWorkflows(ctx context.Context, _ *proto.ListWorkflowsRequest) (*proto.ListWorkflowsResponse, error) {
	ws, err := s.engine.ListWorkflows(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list workflows: %v", err)
	}
	out := &proto.ListWorkflowsResponse{}
	for _, w := range ws {
		out.Workflows = append(out.Workflows, toProtoWorkflow(w))
	}
	return out, nil
}

func toProtoTicket(t *engine.Ticket) *proto.Ticket {
	fieldsJSON := ""
	if t.Fields != nil {
		if b, err := json.Marshal(t.Fields); err == nil {
			fieldsJSON = string(b)
		}
	}
	return &proto.Ticket{
		Id:             t.ID,
		Workflow:       t.Workflow,
		WorkflowVersion: t.WorkflowVer,
		Title:          t.Title,
		FieldsJson:     fieldsJSON,
		Assignee:       t.Assignee,
		Queue:          t.Queue,
		Priority:       t.Priority,
		CreatedAt:      t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoRun(r *engine.WorkflowRun) *proto.WorkflowRun {
	outputJSON := ""
	if r.Output != nil {
		if b, err := json.Marshal(r.Output); err == nil {
			outputJSON = string(b)
		}
	}
	return &proto.WorkflowRun{
		Id:             r.ID,
		TicketId:       r.TicketID,
		Workflow:       r.Workflow,
		WorkflowVersion: r.WorkflowVer,
		Status:         string(r.Status),
		CurrentStep:    r.Current,
		OutputJson:     outputJSON,
		Failed:         r.Failed,
	}
}

func toProtoWorkflow(w engine.WorkflowDef) *proto.WorkflowDef {
	return &proto.WorkflowDef{
		Name:        w.Name,
		Version:     w.Version,
		Description: w.Description,
		Initial:     w.Initial,
		StepCount:   int32(len(w.Steps)),
	}
}
