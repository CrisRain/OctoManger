package webui

import (
	"os"
	"testing"
	"testing/fstest"
)

func TestOpenUsesFilesystemWhenAvailable(t *testing.T) {
	fsys, source := Open()
	if fsys == nil || source == "" {
		t.Fatalf("expected filesystem assets, got source=%q", source)
	}
	if source != "filesystem" && source != "embedded" {
		t.Fatalf("unexpected source %q", source)
	}
}

func TestOpenUsesEmbeddedWhenFilesystemMissing(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	temp := t.TempDir()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	fsys, source := Open()
	if fsys == nil || source != "embedded" {
		t.Fatalf("expected embedded assets, got source=%q", source)
	}
}

func TestOpenEmbeddedAssetsFromFailure(t *testing.T) {
	if fsys, ok := openEmbeddedAssetsFrom(fstest.MapFS{}); ok || fsys != nil {
		t.Fatalf("expected embedded assets to fail with empty fs")
	}
}
