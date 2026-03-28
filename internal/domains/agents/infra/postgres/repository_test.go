package agentpostgres

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "octomanger/internal/domains/agents/domain"
	"octomanger/internal/testutil"
)

func TestRepositoryCRUDAndCache(t *testing.T) {
	db := testutil.NewTestDB(t)
	rdb, _ := testutil.NewTestRedis(t)
	repo := New(db, rdb)
	ctx := context.Background()

	created, err := repo.Create(ctx, agentdomain.CreateAgentInput{
		Name:      "Agent",
		PluginKey: "demo",
		Action:    "RUN",
		Input:     map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.DesiredState != agentdomain.DesiredStateStopped || created.RuntimeState != agentdomain.RuntimeStateIdle {
		t.Fatalf("unexpected defaults %#v", created)
	}

	items, err := repo.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("list: %v len=%d", err, len(items))
	}
	// cache hit
	items, err = repo.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("list cache: %v len=%d", err, len(items))
	}

	pageItems, total, err := repo.ListPage(ctx, 10, 0)
	if err != nil || total != 1 || len(pageItems) != 1 {
		t.Fatalf("list page: %v total=%d len=%d", err, total, len(pageItems))
	}

	runnable, err := repo.ListDesiredRunning(ctx)
	if err != nil {
		t.Fatalf("list desired running: %v", err)
	}
	if len(runnable) != 0 {
		t.Fatalf("expected no running agents")
	}

	newName := "Agent 2"
	updated, err := repo.Patch(ctx, created.ID, agentdomain.PatchAgentInput{Name: &newName})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if updated.Name != "Agent 2" {
		t.Fatalf("unexpected name %q", updated.Name)
	}

	if err := repo.SetDesiredState(ctx, created.ID, agentdomain.DesiredStateRunning); err != nil {
		t.Fatalf("set desired state: %v", err)
	}
	runnable, _ = repo.ListDesiredRunning(ctx)
	if len(runnable) != 1 {
		t.Fatalf("expected running agent")
	}

	now := time.Now().UTC()
	if err := repo.UpdateRuntimeState(ctx, created.ID, agentdomain.RuntimeStateRunning, "", &now); err != nil {
		t.Fatalf("update runtime state: %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found delete error, got %v", err)
	}
}

func TestRepositoryLogsAndCache(t *testing.T) {
	db := testutil.NewTestDB(t)
	rdb, _ := testutil.NewTestRedis(t)
	repo := New(db, rdb)
	ctx := context.Background()

	created, err := repo.Create(ctx, agentdomain.CreateAgentInput{Name: "Agent", PluginKey: "demo", Action: "RUN"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.AppendLog(ctx, created.ID, "log", "msg", map[string]any{"n": 1}); err != nil {
		t.Fatalf("append log: %v", err)
	}

	logs, err := repo.ListLogsAfter(ctx, created.ID, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("list logs: %v len=%d", err, len(logs))
	}
	// cache hit
	logs, err = repo.ListLogsAfter(ctx, created.ID, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("list logs cache: %v len=%d", err, len(logs))
	}

	batch := []agentdomain.AgentLogEntry{{AgentID: created.ID, EventType: "log", Message: "batch"}}
	if err := repo.AppendLogBatch(ctx, batch); err != nil {
		t.Fatalf("append log batch: %v", err)
	}
}

func TestRepositoryErrorPaths(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	if _, err := repo.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := repo.Patch(ctx, 999, agentdomain.PatchAgentInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found patch, got %v", err)
	}
	if err := repo.SetDesiredState(ctx, 999, agentdomain.DesiredStateRunning); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found desired state, got %v", err)
	}
	if err := repo.UpdateRuntimeState(ctx, 999, agentdomain.RuntimeStateRunning, "", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found runtime state, got %v", err)
	}
}
