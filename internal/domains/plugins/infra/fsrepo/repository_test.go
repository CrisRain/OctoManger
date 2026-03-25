package fsrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryListLoadsManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.yaml"), []byte("key: demo\nname: Demo\nentrypoint: main.py\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "main.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}

	repo := New(root)
	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list plugins: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(items))
	}
	if items[0].Manifest.Key != "demo" {
		t.Fatalf("unexpected plugin key: %s", items[0].Manifest.Key)
	}
	if !items[0].Healthy {
		t.Fatalf("expected plugin to be healthy")
	}
}
