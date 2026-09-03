//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"unicode/utf8"
)

// Evidence is operator-owned regular-file input. Nonblocking open ensures a
// FIFO substituted at the path cannot wait forever before the type check.
// Regular-file storage must honor the caller's local I/O availability bound.
func readOperatorEvidence(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errors.New("operator evidence unavailable")
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("operator evidence must be a regular file")
	}
	encoded, err := io.ReadAll(file)
	if ended := ctx.Err(); ended != nil {
		return "", ended
	}
	if err != nil || !utf8.Valid(encoded) {
		return "", errors.New("operator evidence unavailable or not UTF-8")
	}
	return string(encoded), nil
}
