package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"tptconduit/api/proto"
	"tptconduit/engine"
)

func newGRPCServer(t *testing.T) (proto.ConduitClient, func()) {
	t.Helper()
	e := engine.NewEngine(engine.NewInMemoryEventLog(), engine.NewInMemoryStore(), 0)
	wf := engine.WorkflowDef{
		Name:    "signoff",
		Version: "v1",
		Initial: "approve",
		Steps: []engine.StepDef{{
			Name: "approve",
			Kind: engine.KindApproval,
			Approval: &engine.ApprovalDef{
				Chain: []engine.Approver{{Role: "manager"}, {Role: "director"}},
			},
		}},
	}
	if err := e.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register: %v", err)
	}
	srv := NewServer(e)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	gs := grpc.NewServer()
	srv.RegisterOn(gs)
	go func() { _ = gs.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	client := proto.NewConduitClient(conn)
	cleanup := func() {
		conn.Close()
		gs.Stop()
	}
	return client, cleanup
}

func TestGRPCFlow(t *testing.T) {
	client, cleanup := newGRPCServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.CreateTicket(ctx, &proto.CreateTicketRequest{
		Workflow: "signoff", Version: "v1", Title: "Approval needed",
		FieldsJson: `{"amount": 250}`,
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if resp.Ticket.Title != "Approval needed" {
		t.Fatalf("unexpected title: %v", resp.Ticket.Title)
	}

	list, err := client.ListTickets(ctx, &proto.ListTicketsRequest{})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(list.Tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(list.Tickets))
	}

	runs, err := client.ListRuns(ctx, &proto.ListRunsRequest{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs.Runs))
	}
	runID := runs.Runs[0].Id

	if _, err := client.Approve(ctx, &proto.ApproveRequest{RunId: runID, Step: "approve", By: "manager"}); err != nil {
		t.Fatalf("approve 1: %v", err)
	}
	if _, err := client.Approve(ctx, &proto.ApproveRequest{RunId: runID, Step: "approve", By: "director"}); err != nil {
		t.Fatalf("approve 2: %v", err)
	}
	got, err := client.GetRun(ctx, &proto.GetRunRequest{Id: runID})
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Run.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got.Run.Status)
	}
}
