//go:build linux || darwin

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestApplicationRequiresOutputBeforeExternalWork(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		fixture := &commandFixture{}
		args := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir(), "--output-fifo", path}
		if code := runApplication(context.Background(), args, fixture.dependencies()); code != 1 {
			t.Fatal(code)
		}
		if len(fixture.requests) != 0 || len(fixture.turns) != 0 {
			t.Fatal("output rejection performed external work")
		}
	}
}

func TestApplicationWritesOneCycleThroughRealFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	readerFD, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := os.NewFile(uintptr(readerFD), path)
	defer func() { _ = reader.Close() }()
	keeper, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Close(keeper) }()
	fixture := &commandFixture{}
	args := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir(), "--output-fifo", path}
	done := make(chan int, 1)
	go func() { done <- runApplication(context.Background(), args, fixture.dependencies()) }()
	decoder := json.NewDecoder(reader)
	var handoff assessmentRecord
	var processed processedRecord
	if err := decoder.Decode(&handoff); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&processed); err != nil {
		t.Fatal(err)
	}
	if code := <-done; code != 0 {
		t.Fatal(code)
	}
	if handoff.Type != "assessment" || processed.Type != "processed_report" || handoff.Assessment.Outcome != "retain" || processed.Assessment.Outcome != "retain" {
		t.Fatal("invalid real output")
	}
	if fixture.outputCalls != 0 {
		t.Fatal("application bypassed configured FIFO")
	}
}

func TestApplicationHandlesOSSignals(t *testing.T) {
	for _, signal := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(signal.String(), func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "output")
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
			reader, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reader.Close() }()
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			child := exec.Command(binary, "-test.run=^TestApplicationSignalHelper$")
			child.Env = append(os.Environ(), "ARCLOOM_SIGNAL_HELPER=1", "ARCLOOM_TEST_FIFO="+path, "ARCLOOM_TEST_DIRECTORY="+directory)
			stdout, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if child.ProcessState == nil {
					_ = child.Process.Kill()
					_ = child.Wait()
				}
			})
			line, err := bufio.NewReader(stdout).ReadString('\n')
			if err != nil || line != "ready\n" {
				t.Fatalf("readiness=%q %v", line, err)
			}
			if err := child.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			var exited *exec.ExitError
			if err := child.Wait(); !errors.As(err, &exited) || exited.ExitCode() != 130 {
				t.Fatalf("signal exit=%v", err)
			}
		})
	}
}

// This subprocess uses the same signal-lifecycle entry as main, with injected
// Provider boundaries. No production flag or network endpoint is added for tests.
func TestApplicationSignalHelper(t *testing.T) {
	if os.Getenv("ARCLOOM_SIGNAL_HELPER") != "1" {
		return
	}
	fixture := &commandFixture{onHTTP: func(ctx context.Context) { _, _ = fmt.Fprintln(os.Stdout, "ready"); <-ctx.Done() }}
	args := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", os.Getenv("ARCLOOM_TEST_DIRECTORY"), "--output-fifo", os.Getenv("ARCLOOM_TEST_FIFO")}
	os.Exit(runInterrupted(args, fixture.dependencies()))
}
