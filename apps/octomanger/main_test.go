package main

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	origRun := runEntrypoint
	runEntrypoint = func(args []string) error {
		if len(args) != 2 || args[0] != "a" || args[1] != "b" {
			t.Fatalf("unexpected args %#v", args)
		}
		return nil
	}
	t.Cleanup(func() { runEntrypoint = origRun })

	code := run([]string{"a", "b"})
	if code != 0 {
		t.Fatalf("expected zero exit code, got %d", code)
	}
}

func TestRunErrorWritesToStderr(t *testing.T) {
	origRun := runEntrypoint
	runEntrypoint = func(args []string) error {
		return errors.New("boom")
	}
	t.Cleanup(func() { runEntrypoint = origRun })

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
	}()

	code := run(nil)
	if code != 1 {
		t.Fatalf("expected error exit code, got %d", code)
	}

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if string(out) == "" {
		t.Fatalf("expected stderr output")
	}
}
