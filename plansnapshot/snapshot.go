package plansnapshot

import (
	"errors"

	"github.com/kotokumu/arcloom/plan"
)

var (
	errInvalidCurrentPlan    = errors.New("invalid current plan")
	errIncompleteCurrentPlan = errors.New("incomplete progress cannot accompany current plan")
	errTaskProgressMismatch  = errors.New("plan and progress tasks do not correspond")
)

// Snapshot is one coherent current observation. It may contain progress
// without a current Plan when Plan facts are unavailable or invalid.
type Snapshot struct {
	current     plan.Plan
	hasCurrent  bool
	progress    ProgressEvidence
	hasProgress bool
	valid       bool
}

// New requires a valid current Plan, complete progress, and exact ordered
// correspondence between Plan task names and progress member names.
func New(current plan.Plan, progress ProgressEvidence) (Snapshot, error) {
	if !current.IsValid() {
		return Snapshot{}, errInvalidCurrentPlan
	}
	if !progress.valid {
		return Snapshot{}, errInvalidProgress
	}
	if !progress.complete {
		return Snapshot{}, errIncompleteCurrentPlan
	}
	planTasks := current.Tasks()
	progressTasks := progress.tasks
	if len(planTasks) != len(progressTasks) {
		return Snapshot{}, errTaskProgressMismatch
	}
	for i := range planTasks {
		if planTasks[i].Name() != progressTasks[i].name {
			return Snapshot{}, errTaskProgressMismatch
		}
	}
	return Snapshot{current: current, hasCurrent: true, progress: cloneProgress(progress), hasProgress: true, valid: true}, nil
}

// WithoutCurrent preserves valid progress when a coherent current Plan cannot
// be established.
func WithoutCurrent(progress ProgressEvidence) (Snapshot, error) {
	if !progress.valid {
		return Snapshot{}, errInvalidProgress
	}
	return Snapshot{progress: cloneProgress(progress), hasProgress: true, valid: true}, nil
}

func cloneProgress(progress ProgressEvidence) ProgressEvidence {
	progress.tasks = cloneTasks(progress.tasks)
	return progress
}

func (s Snapshot) CurrentPlan() (plan.Plan, bool) {
	if !s.valid || !s.hasCurrent {
		return plan.Plan{}, false
	}
	return s.current, true
}

func (s Snapshot) Progress() (ProgressEvidence, bool) {
	if !s.valid || !s.hasProgress {
		return ProgressEvidence{}, false
	}
	return cloneProgress(s.progress), true
}
