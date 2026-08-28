// Package codexplancontrol adapts a trusted local Codex installation to the
// provider-independent Plan Control assessor contract.
package codexplancontrol

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maximumShutdownGrace = 30 * time.Second

var (
	errInvalidModel            = errors.New("codex plan control: invalid model")
	errInvalidReasoningEffort  = errors.New("codex plan control: invalid reasoning effort")
	errInvalidWorkingDirectory = errors.New("codex plan control: invalid working directory")
	errInvalidShutdownGrace    = errors.New("codex plan control: invalid shutdown grace")
)

// Model is a non-empty Codex model identifier accepted from the Host.
type Model struct {
	value string
	valid bool
}

// NewModel validates a Codex model identifier without starting Codex.
func NewModel(value string) (Model, error) {
	if strings.TrimSpace(value) == "" {
		return Model{}, errInvalidModel
	}
	return Model{value: value, valid: true}, nil
}

// ReasoningEffort is one reasoning effort supported by the compatible Codex
// installation targeted by this adapter.
type ReasoningEffort struct {
	value string
	valid bool
}

// NewReasoningEffort validates a supported reasoning effort without starting
// Codex.
func NewReasoningEffort(value string) (ReasoningEffort, error) {
	switch value {
	case "low", "medium", "high", "xhigh", "max", "ultra":
		return ReasoningEffort{value: value, valid: true}, nil
	default:
		return ReasoningEffort{}, errInvalidReasoningEffort
	}
}

// WorkingDirectory is an existing absolute local directory used for one
// assessment interaction.
type WorkingDirectory struct {
	value string
	valid bool
}

// NewWorkingDirectory validates a local working directory without starting
// Codex.
func NewWorkingDirectory(value string) (WorkingDirectory, error) {
	if !filepath.IsAbs(value) {
		return WorkingDirectory{}, errInvalidWorkingDirectory
	}
	info, err := os.Stat(value)
	if err != nil || !info.IsDir() {
		return WorkingDirectory{}, errInvalidWorkingDirectory
	}
	return WorkingDirectory{value: value, valid: true}, nil
}

// ShutdownGrace bounds termination of a cancelled Codex interaction.
type ShutdownGrace struct {
	value time.Duration
	valid bool
}

// NewShutdownGrace accepts a positive duration no greater than 30 seconds.
func NewShutdownGrace(value time.Duration) (ShutdownGrace, error) {
	if value <= 0 || value > maximumShutdownGrace {
		return ShutdownGrace{}, errInvalidShutdownGrace
	}
	return ShutdownGrace{value: value, valid: true}, nil
}

// Configuration is immutable Safe Host Configuration for Codex assessment.
type Configuration struct {
	model            Model
	reasoningEffort  ReasoningEffort
	workingDirectory WorkingDirectory
	shutdownGrace    ShutdownGrace
}

// NewConfiguration combines independently validated Host values without I/O.
func NewConfiguration(
	model Model,
	reasoningEffort ReasoningEffort,
	workingDirectory WorkingDirectory,
	shutdownGrace ShutdownGrace,
) Configuration {
	return Configuration{
		model:            model,
		reasoningEffort:  reasoningEffort,
		workingDirectory: workingDirectory,
		shutdownGrace:    shutdownGrace,
	}
}

func (c Configuration) isValid() bool {
	return c.model.valid &&
		c.reasoningEffort.valid &&
		c.workingDirectory.valid &&
		c.shutdownGrace.valid
}
