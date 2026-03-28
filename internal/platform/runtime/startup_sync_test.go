package runtime

import (
	"context"
	"testing"

	"go.uber.org/zap"

	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
)

type stubPluginService struct {
	listFn func(context.Context) ([]plugindomain.Plugin, error)
	syncFn func(context.Context, pluginapp.SyncAccountTypeFunc) error
}

func (s stubPluginService) Execute(context.Context, string, plugindomain.ExecutionRequest, func(plugindomain.ExecutionEvent)) error {
	return nil
}

func (s stubPluginService) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

func (s stubPluginService) Get(context.Context, string) (*plugindomain.Plugin, error) {
	return nil, nil
}

func (s stubPluginService) SyncAccountTypes(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
	if s.syncFn != nil {
		return s.syncFn(ctx, fn)
	}
	return nil
}

func TestSyncPluginAccountTypesOnStartupBranches(t *testing.T) {
	ctx := context.Background()
	app := &App{Logger: zap.NewNop()}

	app.Plugins = stubPluginService{listFn: func(context.Context) ([]plugindomain.Plugin, error) {
		return nil, context.Canceled
	}}
	syncPluginAccountTypesOnStartup(ctx, app)

	app.Plugins = stubPluginService{listFn: func(context.Context) ([]plugindomain.Plugin, error) {
		return []plugindomain.Plugin{{Manifest: plugindomain.Manifest{Key: "p"}, Healthy: false}}, nil
	}}
	syncPluginAccountTypesOnStartup(ctx, app)

	called := false
	app.Plugins = stubPluginService{
		listFn: func(context.Context) ([]plugindomain.Plugin, error) {
			return []plugindomain.Plugin{{Manifest: plugindomain.Manifest{Key: "p"}, Healthy: true}}, nil
		},
		syncFn: func(context.Context, pluginapp.SyncAccountTypeFunc) error {
			called = true
			return context.Canceled
		},
	}
	syncPluginAccountTypesOnStartup(ctx, app)
	if !called {
		t.Fatalf("expected sync to be attempted")
	}

	called = false
	app.Plugins = stubPluginService{
		listFn: func(context.Context) ([]plugindomain.Plugin, error) {
			return []plugindomain.Plugin{{Manifest: plugindomain.Manifest{Key: "p"}, Healthy: true}}, nil
		},
		syncFn: func(context.Context, pluginapp.SyncAccountTypeFunc) error {
			called = true
			return nil
		},
	}
	syncPluginAccountTypesOnStartup(ctx, app)
	if !called {
		t.Fatalf("expected sync to succeed")
	}
}

var _ plugins.PluginService = stubPluginService{}
