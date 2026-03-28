package entrypoint

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"octomanger/internal/platform/apiserver"
	"octomanger/internal/platform/migrator"
	platformruntime "octomanger/internal/platform/runtime"
	platformworker "octomanger/internal/platform/worker"
)

const usageText = `Usage:
  octomanger
  octomanger migrate
  octomanger migrate down

The service starts the full stack by default:
  1. Run versioned database migrations
  2. Start the HTTP API
  3. Start the worker loop
  4. Serve the bundled web UI

Migration commands:
  octomanger migrate       Apply all pending migrations
  octomanger migrate down  Roll back the latest applied migration
`

func Run(args []string) error {
	return runWith(args, defaultRunDeps())
}

type runDeps struct {
	migrate   func(context.Context) error
	rollback  func(context.Context) error
	bootstrap func(context.Context) (*platformruntime.App, error)
	apiRun    func(context.Context, *platformruntime.App) error
	workerRun func(context.Context, *platformruntime.App) error
}

func defaultRunDeps() runDeps {
	return runDeps{
		migrate:   migrator.Run,
		rollback:  migrator.RollbackLast,
		bootstrap: platformruntime.Bootstrap,
		apiRun:    apiserver.RunWithApp,
		workerRun: platformworker.RunWithApp,
	}
}

func runWith(args []string, deps runDeps) error {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Fprint(os.Stdout, usageText)
			return nil
		case "migrate":
			return runMigrateCommand(args[1:], deps)
		default:
			return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	migrateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err := deps.migrate(migrateCtx)
	cancel()
	if err != nil {
		return err
	}

	application, err := deps.bootstrap(ctx)
	if err != nil {
		return err
	}
	defer application.Close()

	workerDone := make(chan error, 1)
	go func() {
		workerDone <- deps.workerRun(ctx, application)
	}()

	apiErr := deps.apiRun(ctx, application)
	stop()
	workerErr := <-workerDone

	if apiErr != nil {
		return apiErr
	}
	return workerErr
}

func runMigrateCommand(args []string, deps runDeps) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch len(args) {
	case 0:
		return deps.migrate(ctx)
	case 1:
		if args[0] == "down" {
			return deps.rollback(ctx)
		}
	}
	return fmt.Errorf("unknown migrate command: %v\n\n%s", args, usageText)
}
