package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	agentapp "octomanger/internal/domains/agents/app"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	jobapp "octomanger/internal/domains/jobs/app"
	jobdomain "octomanger/internal/domains/jobs/domain"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/domains/plugins/grpclauncher"
	"octomanger/internal/platform/config"
	platformruntime "octomanger/internal/platform/runtime"
	"octomanger/internal/testutil"
)

type stubPluginService struct {
	plugins   []plugindomain.Plugin
	listErr   error
	listFn    func(ctx context.Context) ([]plugindomain.Plugin, error)
	execFn    func(ctx context.Context, pluginKey string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error
	syncFn    func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error
	execCalls int32
}

func (s *stubPluginService) Execute(ctx context.Context, pluginKey string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
	atomic.AddInt32(&s.execCalls, 1)
	if s.execFn != nil {
		return s.execFn(ctx, pluginKey, req, onEvent)
	}
	if onEvent != nil {
		onEvent(plugindomain.ExecutionEvent{Type: "result", Data: map[string]any{"ok": true}})
	}
	return nil
}

func (s *stubPluginService) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.plugins, nil
}

func (s *stubPluginService) Get(ctx context.Context, key string) (*plugindomain.Plugin, error) {
	for _, plugin := range s.plugins {
		if plugin.Manifest.Key == key {
			item := plugin
			return &item, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *stubPluginService) SyncAccountTypes(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
	if s.syncFn != nil {
		return s.syncFn(ctx, fn)
	}
	return nil
}

func newTestApp(t *testing.T, pluginSvc plugins.PluginService) *platformruntime.App {
	t.Helper()

	db := testutil.NewTestDB(t)
	logger := zap.NewNop()

	cfg := config.Config{
		Plugins: config.PluginsConfig{
			Services: map[string]config.PluginServiceEntry{
				"demo": {Address: "127.0.0.1:50051"},
			},
		},
		Worker: config.WorkerConfig{
			ID:                 "worker-1",
			PollInterval:       2 * time.Millisecond,
			ExecutionPollLimit: 2,
			SchedulePollLimit:  1,
			AgentScanInterval:  2 * time.Millisecond,
		},
	}

	accountSvc := accounttypeapp.New(accounttypepostgres.New(db))
	jobRepo := jobpostgres.New(db)
	jobsSvc := jobapp.New(logger, jobRepo, pluginSvc, cfg.Worker.ID)
	agentRepo := agentpostgres.New(db)
	agentsSvc := agentapp.New(logger, agentRepo, pluginSvc, nil, cfg.Worker.ID, time.Millisecond, time.Millisecond)

	return &platformruntime.App{
		Config:       cfg,
		Logger:       logger,
		DB:           db,
		Plugins:      pluginSvc,
		AccountTypes: accountSvc,
		Jobs:         jobsSvc,
		Agents:       agentsSvc,
	}
}

func TestRunBootstrapError(t *testing.T) {
	origBootstrap := bootstrapApp
	bootstrapApp = func(ctx context.Context) (*platformruntime.App, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { bootstrapApp = origBootstrap })

	if err := Run(context.Background()); err == nil {
		t.Fatalf("expected bootstrap error")
	}
}

func TestRunSuccessUsesHooks(t *testing.T) {
	origBootstrap := bootstrapApp
	origStartLauncher := startPluginLauncherFn
	origStartSync := startPluginAccountTypeSyncFn
	origStartAgent := startAgentSupervisorFn
	origRunLoop := runJobLoopFn
	t.Cleanup(func() {
		bootstrapApp = origBootstrap
		startPluginLauncherFn = origStartLauncher
		startPluginAccountTypeSyncFn = origStartSync
		startAgentSupervisorFn = origStartAgent
		runJobLoopFn = origRunLoop
	})

	pluginSvc := &stubPluginService{}
	app := newTestApp(t, pluginSvc)
	bootstrapApp = func(ctx context.Context) (*platformruntime.App, error) {
		return app, nil
	}

	var called int32
	startPluginLauncherFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
		atomic.AddInt32(&called, 1)
	}
	startPluginAccountTypeSyncFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger, interval time.Duration) {
		atomic.AddInt32(&called, 1)
	}
	startAgentSupervisorFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
		atomic.AddInt32(&called, 1)
	}
	runJobLoopFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
		atomic.AddInt32(&called, 1)
	}

	if err := Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if atomic.LoadInt32(&called) != 4 {
		t.Fatalf("expected all hooks to be called, got %d", called)
	}
}

func TestToAddressMap(t *testing.T) {
	in := map[string]config.PluginServiceEntry{
		"a": {Address: "127.0.0.1:50051"},
		"b": {Address: "127.0.0.1:50052"},
	}
	out := toAddressMap(in)
	if out["a"] != "127.0.0.1:50051" || out["b"] != "127.0.0.1:50052" {
		t.Fatalf("unexpected map %#v", out)
	}
}

func TestStartPluginAccountTypeSyncSkipsWhenNoServices(t *testing.T) {
	origSync := syncPluginAccountTypesFn
	defer func() { syncPluginAccountTypesFn = origSync }()

	called := int32(0)
	syncPluginAccountTypesFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger, interval time.Duration) {
		atomic.AddInt32(&called, 1)
	}

	app := &platformruntime.App{
		Config: config.Config{
			Plugins: config.PluginsConfig{Services: map[string]config.PluginServiceEntry{}},
		},
	}
	startPluginAccountTypeSync(context.Background(), app, zap.NewNop(), time.Millisecond)
	if atomic.LoadInt32(&called) != 0 {
		t.Fatalf("expected sync not to be scheduled")
	}
}

func TestStartPluginAccountTypeSyncSchedules(t *testing.T) {
	origSync := syncPluginAccountTypesFn
	t.Cleanup(func() { syncPluginAccountTypesFn = origSync })

	called := make(chan struct{}, 1)
	syncPluginAccountTypesFn = func(ctx context.Context, application *platformruntime.App, logger *zap.Logger, interval time.Duration) {
		called <- struct{}{}
	}

	app := &platformruntime.App{
		Config: config.Config{
			Plugins: config.PluginsConfig{
				Services: map[string]config.PluginServiceEntry{
					"demo": {Address: "addr"},
				},
			},
		},
	}
	startPluginAccountTypeSync(context.Background(), app, zap.NewNop(), time.Millisecond)

	select {
	case <-called:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected sync to be scheduled")
	}
}

func TestStartPluginLauncherRunsManager(t *testing.T) {
	origDiscover := discoverLaunchesFn
	origNewLauncher := newLauncherFn
	t.Cleanup(func() {
		discoverLaunchesFn = origDiscover
		newLauncherFn = origNewLauncher
	})

	discoverLaunchesFn = func(modulesDir string, services map[string]string) []grpclauncher.ProcessConfig {
		if modulesDir != "modules" || services["demo"] != "addr" {
			t.Fatalf("unexpected discover inputs")
		}
		return nil
	}

	called := make(chan struct{}, 1)
	newLauncherFn = func(logger *zap.Logger, pythonBin, sdkDir string, launches []grpclauncher.ProcessConfig) launcherRunner {
		return launcherRunnerFunc(func(ctx context.Context) error {
			called <- struct{}{}
			return errors.New("boom")
		})
	}

	app := &platformruntime.App{
		Config: config.Config{
			Plugins: config.PluginsConfig{
				ModulesDir: "modules",
				Services: map[string]config.PluginServiceEntry{
					"demo": {Address: "addr"},
				},
			},
		},
	}
	startPluginLauncher(context.Background(), app, zap.NewNop())

	select {
	case <-called:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected launcher to run")
	}
}

type launcherRunnerFunc func(context.Context) error

func (f launcherRunnerFunc) Run(ctx context.Context) error {
	return f(ctx)
}

func TestSyncPluginAccountTypesSuccess(t *testing.T) {
	pluginSvc := &stubPluginService{
		plugins: []plugindomain.Plugin{
			{Manifest: plugindomain.Manifest{Key: "demo", Name: "Demo"}, Healthy: true},
		},
		syncFn: func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
			return fn(ctx, pluginapp.AccountTypeSpec{Key: "demo", Name: "Demo"})
		},
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("sync did not complete")
	}
}

func TestSyncPluginAccountTypesStopsOnContext(t *testing.T) {
	pluginSvc := &stubPluginService{
		listErr: errors.New("list failed"),
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("sync did not stop")
	}
}

func TestSyncPluginAccountTypesSyncError(t *testing.T) {
	pluginSvc := &stubPluginService{
		plugins: []plugindomain.Plugin{
			{Manifest: plugindomain.Manifest{Key: "demo", Name: "Demo"}, Healthy: true},
		},
		syncFn: func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
			return errors.New("sync failed")
		},
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("sync did not stop on error")
	}
}

func TestSyncPluginAccountTypesNoHealthy(t *testing.T) {
	pluginSvc := &stubPluginService{
		plugins: []plugindomain.Plugin{
			{Manifest: plugindomain.Manifest{Key: "demo", Name: "Demo"}, Healthy: false},
		},
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("sync did not stop with no healthy plugins")
	}
}

func TestSyncPluginAccountTypesNoPlugins(t *testing.T) {
	pluginSvc := &stubPluginService{}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("sync did not stop with no plugins")
	}
}

func TestSyncPluginAccountTypesPendingZeroNotComplete(t *testing.T) {
	pluginSvc := &stubPluginService{
		plugins: []plugindomain.Plugin{
			{Manifest: plugindomain.Manifest{Key: "healthy", Name: "Healthy"}, Healthy: true},
			{Manifest: plugindomain.Manifest{Key: "unhealthy", Name: "Unhealthy"}, Healthy: false},
		},
		syncFn: func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
			return fn(ctx, pluginapp.AccountTypeSpec{Key: "healthy", Name: "Healthy"})
		},
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	<-ctx.Done()
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("sync did not stop")
	}
}

func TestSyncPluginAccountTypesCompletesOnTick(t *testing.T) {
	var calls int32
	pluginSvc := &stubPluginService{
		listFn: func(ctx context.Context) ([]plugindomain.Plugin, error) {
			if atomic.AddInt32(&calls, 1) == 1 {
				return []plugindomain.Plugin{
					{Manifest: plugindomain.Manifest{Key: "healthy", Name: "Healthy"}, Healthy: true},
					{Manifest: plugindomain.Manifest{Key: "unhealthy", Name: "Unhealthy"}, Healthy: false},
				}, nil
			}
			return []plugindomain.Plugin{
				{Manifest: plugindomain.Manifest{Key: "healthy", Name: "Healthy"}, Healthy: true},
			}, nil
		},
		syncFn: func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
			return fn(ctx, pluginapp.AccountTypeSpec{Key: "healthy", Name: "Healthy"})
		},
	}
	app := newTestApp(t, pluginSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncPluginAccountTypes(ctx, app, zap.NewNop(), 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("sync did not complete on tick")
	}
}

func TestStartAgentSupervisorLogsWhenRunnerErrors(t *testing.T) {
	origRunner := runAgentSupervisorFn
	t.Cleanup(func() { runAgentSupervisorFn = origRunner })

	done := make(chan struct{}, 1)
	runAgentSupervisorFn = func(ctx context.Context, svc *agentapp.Service, interval time.Duration) error {
		done <- struct{}{}
		return errors.New("boom")
	}

	app := newTestApp(t, &stubPluginService{})
	startAgentSupervisor(context.Background(), app, zap.NewNop())
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected supervisor to run")
	}
}

func TestStartAgentSupervisorNoError(t *testing.T) {
	origRunner := runAgentSupervisorFn
	t.Cleanup(func() { runAgentSupervisorFn = origRunner })

	done := make(chan struct{}, 1)
	runAgentSupervisorFn = func(ctx context.Context, svc *agentapp.Service, interval time.Duration) error {
		done <- struct{}{}
		return nil
	}

	app := newTestApp(t, &stubPluginService{})
	startAgentSupervisor(context.Background(), app, zap.NewNop())
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected supervisor to run")
	}
}

func TestRunJobLoopStopsOnContext(t *testing.T) {
	app := newTestApp(t, &stubPluginService{})
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		runJobLoop(ctx, app, zap.NewNop())
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("worker loop did not stop")
	}
}

func TestRunWorkerTickHandlesErrors(t *testing.T) {
	pluginSvc := &stubPluginService{}
	app := newTestApp(t, pluginSvc)
	if sqlDB, err := app.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}

	runWorkerTick(context.Background(), context.Background(), app, zap.NewNop())
}

func TestProcessQueuedExecutionsStopsOnContext(t *testing.T) {
	app := newTestApp(t, &stubPluginService{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processQueuedExecutions(ctx, context.Background(), app, zap.NewNop())
}

func TestProcessQueuedExecutionsHandlesError(t *testing.T) {
	app := newTestApp(t, &stubPluginService{})
	if sqlDB, err := app.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}

	processQueuedExecutions(context.Background(), context.Background(), app, zap.NewNop())
}

func TestProcessQueuedExecutionsProcessesExecutions(t *testing.T) {
	pluginSvc := &stubPluginService{}
	app := newTestApp(t, pluginSvc)
	repo := jobpostgres.New(app.DB)

	ctx := context.Background()
	def, err := repo.CreateDefinition(ctx, jobdomain.CreateDefinitionInput{
		Key:       "demo",
		Name:      "Demo",
		PluginKey: "demo",
		Action:    "RUN",
		Input:     map[string]any{"a": 1},
	}, nil)
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}
	if _, err := repo.EnqueueExecution(ctx, def.ID, "tester", "manual", map[string]any{"b": 2}); err != nil {
		t.Fatalf("enqueue execution: %v", err)
	}

	processQueuedExecutions(context.Background(), context.Background(), app, zap.NewNop())

	execs, err := repo.ListExecutions(ctx)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execs))
	}
	if execs[0].Status != jobdomain.StatusSucceeded {
		t.Fatalf("expected execution succeeded, got %q", execs[0].Status)
	}
}
