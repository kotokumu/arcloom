//go:build linux || darwin

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOpenFIFOOutputRejectsInvalidDestination(t *testing.T) {
	directory := t.TempDir()
	regular := filepath.Join(directory, "existing")
	if err := os.WriteFile(regular, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(directory, "without-reader")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	// gotests scaffold; the want type and fields are unchanged.
	type args struct {
		ctx  context.Context
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *fifoOutput
		wantErr bool
	}{
		{name: "empty", args: args{context.Background(), ""}, want: nil, wantErr: true},
		{name: "missing", args: args{context.Background(), filepath.Join(directory, "missing")}, want: nil, wantErr: true},
		{name: "regular", args: args{context.Background(), regular}, want: nil, wantErr: true},
		{name: "directory", args: args{context.Background(), directory}, want: nil, wantErr: true},
		{name: "no reader", args: args{context.Background(), fifo}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := openFIFOOutput(tt.args.ctx, tt.args.path)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Fatal(diff)
			}
		})
	}
	contents, err := os.ReadFile(regular)
	if err != nil || string(contents) != "preserve" {
		t.Fatal("existing output file changed")
	}
	if _, err := os.Stat(filepath.Join(directory, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("output path created")
	}
}

func TestFIFOOutputPreservesCompleteRecordsAndDescriptorIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	readerFD, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := os.NewFile(uintptr(readerFD), path)
	defer func() { _ = reader.Close() }()
	original, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Close(original) }()
	before, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(original), syscall.F_GETFL, 0)
	if errno != 0 {
		t.Fatal(errno)
	}
	stdoutBefore, _, errno := syscall.Syscall(syscall.SYS_FCNTL, 1, syscall.F_GETFL, 0)
	if errno != 0 {
		t.Fatal(errno)
	}
	output, err := openFIFOOutput(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{"type": "assessment", "value": "日本語"}, {"type": "processed_report", "value": "same"}}
	for _, record := range want {
		if err := output.WriteRecord(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	after, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(original), syscall.F_GETFL, 0)
	if errno != 0 || before != after {
		t.Fatal("another writer's flags changed")
	}
	stdoutAfter, _, errno := syscall.Syscall(syscall.SYS_FCNTL, 1, syscall.F_GETFL, 0)
	if errno != 0 || stdoutBefore != stdoutAfter {
		t.Fatal("inherited stdout flags changed")
	}
	// Closing output must not close the independently owned writer.
	if _, err := syscall.Write(original, []byte("\n")); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(reader)
	for _, record := range want {
		var got map[string]string
		if err := decoder.Decode(&got); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(record, got); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestFIFOOutputCancellationInterruptsActiveWriteWithStoppedReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blocked")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	readerFD, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := os.NewFile(uintptr(readerFD), path)
	defer func() { _ = reader.Close() }()
	output, err := openFIFOOutput(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = output.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	// Exceeds the OS pipe buffer. Reading its first byte proves the real write
	// began; the reader then stops, with the rest of the record still pending.
	go func() { done <- output.WriteRecord(ctx, map[string]string{"value": strings.Repeat("x", 8<<20)}) }()
	first := make([]byte, 1)
	if _, err := io.ReadFull(reader, first); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("active write=%v", err)
	}
	// Return includes joining the cancellation callback, so Close is safe now.
}

func TestFIFOOutputCancelBeforeWriteAndReaderFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "closed-reader")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	readerFD, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := os.NewFile(uintptr(readerFD), path)
	output, err := openFIFOOutput(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = output.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := output.WriteRecord(ctx, map[string]string{"value": "not delivered"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.WriteRecord(context.Background(), map[string]string{"value": "unavailable"}); err == nil {
		t.Fatal("closed reader treated as success")
	}
}
