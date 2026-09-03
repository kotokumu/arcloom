//go:build linux || darwin

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"
	"time"
)

// fifoOutput owns one independent descriptor, not an inherited stdout clone.
// The one-cycle command writes sequentially and closes only after writes return.
type fifoOutput struct{ file *os.File }

func openFIFOOutput(ctx context.Context, path string) (*fifoOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("output must name an existing FIFO")
	}
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.New("output FIFO unavailable; start its reader first")
	}
	// NewFile preserves this independently opened nonblocking descriptor and
	// enrolls the writer in the poller. On macOS OpenFile deliberately avoids
	// FIFO polling because its reader EOF notifications are unreliable; this
	// descriptor is write-only and never waits for a reader EOF notification.
	file := os.NewFile(uintptr(fd), path)
	after, err := file.Stat()
	if err != nil || after.Mode()&os.ModeNamedPipe == 0 || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, errors.New("output FIFO changed during open")
	}
	if err := file.SetWriteDeadline(time.Time{}); err != nil {
		_ = file.Close()
		return nil, errors.New("output FIFO does not support cancellation")
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &fifoOutput{file: file}, nil
}

func (out *fifoOutput) WriteRecord(ctx context.Context, value any) (result error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	callbackDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(callbackDone); _ = out.file.SetWriteDeadline(time.Now()) })
	defer func() {
		if !stop() {
			<-callbackDone
		}
		if err := out.file.SetWriteDeadline(time.Time{}); result == nil {
			result = err
		}
	}()
	written, err := out.file.Write(encoded)
	if ended := ctx.Err(); ended != nil {
		return ended
	}
	if err != nil {
		return err
	}
	if written != len(encoded) {
		return io.ErrShortWrite
	}
	return nil
}

func (out *fifoOutput) Close() error { return out.file.Close() }
