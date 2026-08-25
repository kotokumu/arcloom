// Package plansnapshot contains provider-independent current Plan snapshots
// and the progress evidence produced alongside them.
package plansnapshot

import (
	"errors"

	"github.com/kotokumu/arcloom/plan"
)

// ProgressState is the observed state of a representation or member.
type ProgressState uint8

const (
	Open ProgressState = iota + 1
	Closed
	Unknown
)

var (
	errInvalidProgressState = errors.New("invalid progress state")
	errInvalidTaskProgress  = errors.New("invalid task progress")
	errInvalidProgress      = errors.New("invalid progress evidence")
)

// TaskProgress is immutable provider-independent progress for one named task.
type TaskProgress struct {
	name  string
	state ProgressState
	valid bool
}

// NewTaskProgress constructs progress for one valid Plan task name. Duplicate
// names are intentionally permitted here so observations preserve every
// member in exact order.
func NewTaskProgress(name string, state ProgressState) (TaskProgress, error) {
	if _, err := plan.NewTask(name); err != nil {
		return TaskProgress{}, err
	}
	if !validProgressState(state) {
		return TaskProgress{}, errInvalidProgressState
	}
	return TaskProgress{name: name, state: state, valid: true}, nil
}

func (p TaskProgress) Name() string         { return p.name }
func (p TaskProgress) State() ProgressState { return p.state }

// ProgressEvidence is immutable overall and member progress for one
// observation. Membership completeness is independent of the observed state.
type ProgressEvidence struct {
	overall  ProgressState
	tasks    []TaskProgress
	complete bool
	valid    bool
}

// CompleteProgress constructs evidence after all current members were
// established.
func CompleteProgress(overall ProgressState, tasks []TaskProgress) (ProgressEvidence, error) {
	return newProgress(overall, tasks, true)
}

// IncompleteProgress constructs evidence when current membership could not be
// fully established.
func IncompleteProgress(overall ProgressState, tasks []TaskProgress) (ProgressEvidence, error) {
	return newProgress(overall, tasks, false)
}

func newProgress(overall ProgressState, tasks []TaskProgress, complete bool) (ProgressEvidence, error) {
	if !validProgressState(overall) {
		return ProgressEvidence{}, errInvalidProgressState
	}
	for _, task := range tasks {
		if !task.valid || !validProgressState(task.state) {
			return ProgressEvidence{}, errInvalidTaskProgress
		}
	}
	return ProgressEvidence{overall: overall, tasks: cloneTasks(tasks), complete: complete, valid: true}, nil
}

func validProgressState(state ProgressState) bool {
	return state == Open || state == Closed || state == Unknown
}

func cloneTasks(tasks []TaskProgress) []TaskProgress {
	if tasks == nil {
		return nil
	}
	return append([]TaskProgress{}, tasks...)
}

func (p ProgressEvidence) OverallState() ProgressState { return p.overall }
func (p ProgressEvidence) Tasks() []TaskProgress       { return cloneTasks(p.tasks) }
func (p ProgressEvidence) MembershipComplete() bool    { return p.valid && p.complete }
