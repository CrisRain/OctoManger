package entrypoint

import (
	"context"
	"errors"
	"testing"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/logging"
	platformruntime "octomanger/internal/platform/runtime"
	"octomanger/internal/testutil"
)

func TestRunHelp(t *testing.T) {
	if err := Run([]string{"help"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if err := Run([]string{"unknown"}); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestRunMigrateCommand(t *testing.T) {
	called := struct{ migrate, rollback int }{}
	deps := runDeps{
		migrate:  func(context.Context) error { called.migrate++; return nil },
		rollback: func(context.Context) error { called.rollback++; return nil },
	}

	if err := runMigrateCommand([]string{}, deps); err != nil {
		t.Fatalf("migrate command: %v", err)
	}
	if err := runMigrateCommand([]string{"down"}, deps); err != nil {
		t.Fatalf("migrate down command: %v", err)
	}
	if err := runMigrateCommand([]string{"oops"}, deps); err == nil {
		t.Fatalf("expected error for unknown migrate command")
	}
	if called.migrate != 1 || called.rollback != 1 {
		t.Fatalf("unexpected migrate calls %+v", called)
	}
}

func TestRunDefaultPath(t *testing.T) {
	app := &platformruntime.App{
		Logger: logging.New(config.LoggingConfig{Level: "info"}),
		DB:     testutil.NewTestDB(t),
	}

	called := struct{ migrate, api, worker bool }{}
	deps := runDeps{
		migrate: func(context.Context) error {
			called.migrate = true
			return nil
		},
		bootstrap: func(context.Context) (*platformruntime.App, error) {
			return app, nil
		},
		apiRun: func(context.Context, *platformruntime.App) error {
			called.api = true
			return nil
		},
		workerRun: func(context.Context, *platformruntime.App) error {
			called.worker = true
			return nil
		},
		rollback: func(context.Context) error { return errors.New("unexpected rollback") },
	}

	if err := runWith([]string{}, deps); err != nil {
		t.Fatalf("run default path: %v", err)
	}
	if !called.migrate || !called.api || !called.worker {
		t.Fatalf("expected migrate/api/worker to be called: %+v", called)
	}
}
