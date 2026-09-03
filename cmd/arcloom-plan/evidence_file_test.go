//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadOperatorEvidenceReacquiresRegularUTF8File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.txt")
	for _, want := range []string{"First acceptance evidence.\n", "更新された証拠\n"} {
		if err := os.WriteFile(path, []byte(want), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := readOperatorEvidence(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestReadOperatorEvidenceRejectsUnavailableAndNonRegularFiles(t *testing.T) {
	directory := t.TempDir()
	invalid := filepath.Join(directory, "invalid.txt")
	if err := os.WriteFile(invalid, []byte{0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(directory, "pipe")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(directory, "missing"), directory, invalid, fifo} {
		if value, err := readOperatorEvidence(context.Background(), path); err == nil || value != "" {
			t.Fatalf("accepted %s", path)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readOperatorEvidence(ctx, fifo); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
