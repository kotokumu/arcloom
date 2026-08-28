// Package codexappserver defines the contract implemented by the separately
// developed Codex app-server Go SDK.
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

// WorkingDirectory returns the absolute directory that bounds the turn.
func (r ReadOnlyTurnRequest) WorkingDirectory() string { return r.workingDirectory }

// DeveloperInstructions returns the instructions for the isolated turn.
func (r ReadOnlyTurnRequest) DeveloperInstructions() string { return r.developerInstructions }

// Input returns the complete user-input material for the isolated turn.
func (r ReadOnlyTurnRequest) Input() string { return r.input }

// OutputSchema returns the JSON object constraining the final output.
func (r ReadOnlyTurnRequest) OutputSchema() string { return r.outputSchema }

// CompletedTurn contains the single final output established by a successful
// read-only Codex turn.
type CompletedTurn struct {
	FinalOutput string
}

// Client is implemented by the Codex app-server Go SDK.
//
// CompleteReadOnlyTurn owns one isolated app-server interaction. It permits no
// approval, network, tool, or mutation capability; returns only one completed
// final output; honors cancellation; and completes shutdown within the SDK's
// accepted finite bound. The same Client accepts concurrent calls; each call
// keeps material, session, result, and cancellation state isolated, and one
// call's cancellation does not affect another call. Protocol, lifecycle,
// shutdown, or invalid-request failure returns an error and no CompletedTurn.
type Client interface {
	CompleteReadOnlyTurn(context.Context, ReadOnlyTurnRequest) (CompletedTurn, error)
}
