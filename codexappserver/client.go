// Package codexappserver implements a narrow Codex app-server Go SDK for one
// isolated read-only Turn.
package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
)

var errInvalidReadOnlyTurnRequest = errors.New("codex app-server: invalid read-only Turn Request")

// ReadOnlyTurnRequest contains the complete caller-supplied material for one
// isolated read-only Codex turn.
type ReadOnlyTurnRequest struct {
	model                 string
	reasoningEffort       string
	workingDirectory      string
	developerInstructions string
	input                 string
	outputSchema          string
}

// NewReadOnlyTurnRequest validates immutable inputs without beginning an
// app-server interaction.
func NewReadOnlyTurnRequest(
	model string,
	reasoningEffort string,
	workingDirectory string,
	developerInstructions string,
	input string,
	outputSchema string,
) (ReadOnlyTurnRequest, error) {
	var schema map[string]json.RawMessage
	if strings.TrimSpace(model) == "" ||
		strings.TrimSpace(reasoningEffort) == "" ||
		!filepath.IsAbs(workingDirectory) ||
		strings.TrimSpace(developerInstructions) == "" ||
		strings.TrimSpace(input) == "" ||
		validateNoDuplicateJSONKeys([]byte(outputSchema)) != nil ||
		json.Unmarshal([]byte(outputSchema), &schema) != nil ||
		schema == nil {
		return ReadOnlyTurnRequest{}, errInvalidReadOnlyTurnRequest
	}
	return ReadOnlyTurnRequest{
		model:                 model,
		reasoningEffort:       reasoningEffort,
		workingDirectory:      workingDirectory,
		developerInstructions: developerInstructions,
		input:                 input,
		outputSchema:          outputSchema,
	}, nil
}

// Model returns the caller-selected Codex model.
func (r ReadOnlyTurnRequest) Model() string { return r.model }

// ReasoningEffort returns the caller-selected reasoning effort.
func (r ReadOnlyTurnRequest) ReasoningEffort() string { return r.reasoningEffort }

// WorkingDirectory returns the Turn working directory used for project and
// configuration context. It does not bound filesystem reads.
func (r ReadOnlyTurnRequest) WorkingDirectory() string { return r.workingDirectory }

// DeveloperInstructions returns the instructions for the isolated turn.
func (r ReadOnlyTurnRequest) DeveloperInstructions() string { return r.developerInstructions }

// Input returns the complete user-input material for the isolated turn.
func (r ReadOnlyTurnRequest) Input() string { return r.input }

// OutputSchema returns the JSON object constraining the final output.
func (r ReadOnlyTurnRequest) OutputSchema() string { return r.outputSchema }

func (r ReadOnlyTurnRequest) valid() bool {
	request, err := NewReadOnlyTurnRequest(
		r.model,
		r.reasoningEffort,
		r.workingDirectory,
		r.developerInstructions,
		r.input,
		r.outputSchema,
	)
	return err == nil && request == r
}

// CompletedTurn contains the single final output established by a successful
// read-only Codex turn.
type CompletedTurn struct {
	FinalOutput string
}

// Client is implemented by the Codex app-server Go SDK.
//
// CompleteReadOnlyTurn owns one isolated app-server interaction. It fixes
// approval policy to never and the sandbox to read-only with agent-initiated
// network disabled. Read-only local tool activity may occur. The same Client
// accepts concurrent calls; each call keeps process, material, result, and
// cancellation state isolated, and one call's cancellation does not affect
// another. Cancellation before a correlated completed Turn and one final output
// returns an error matching the supplied context error after bounded shutdown.
// Established success is not replaced by later cancellation. Protocol,
// lifecycle, correlation, shutdown, or invalid-request failure returns an error
// and no CompletedTurn.
type Client interface {
	// CompleteReadOnlyTurn returns a zero CompletedTurn on every error. Caller
	// cancellation observed before success returns an error matching ctx.Err()
	// after shutdown completes within the Client's configured finite bound; a
	// concurrent shutdown failure remains in the returned error. Success is
	// established only by one correlated completed Turn and final output, and is
	// not replaced by later cancellation.
	CompleteReadOnlyTurn(context.Context, ReadOnlyTurnRequest) (CompletedTurn, error)
}
