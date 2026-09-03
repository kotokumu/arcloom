package githubplan

import (
	"context"
	"net/http"

	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

// MilestoneTarget is the immutable repository/number binding for a snapshot
// observation.
type MilestoneTarget struct {
	repository Repository
	number     ResourceNumber
	valid      bool
}

func NewMilestoneTarget(repository Repository, number ResourceNumber) (MilestoneTarget, error) {
	if !validRepository(repository) {
		return MilestoneTarget{}, &ValidationError{code: InvalidRepository, field: RepositoryOwnerField}
	}
	if number.value <= 0 {
		return MilestoneTarget{}, &ValidationError{code: InvalidResourceNumber, field: ResourceNumberField}
	}
	return MilestoneTarget{repository: repository, number: number, valid: true}, nil
}

// NewMilestoneSnapshotObserver validates local configuration without I/O.
func NewMilestoneSnapshotObserver(client *http.Client, target MilestoneTarget) (plansnapshot.Observer, error) {
	if client == nil || client.Jar != nil {
		return nil, &ValidationError{code: InvalidClient, field: ClientField}
	}
	if !target.valid || !validRepository(target.repository) {
		return nil, &ValidationError{code: InvalidRepository, field: RepositoryOwnerField}
	}
	if target.number.value <= 0 {
		return nil, &ValidationError{code: InvalidResourceNumber, field: ResourceNumberField}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	binding := observerBinding{repository: target.repository, scheme: milestoneScheme, number: target.number}
	access := restAccess{client: clientCopy}
	return func(ctx context.Context) (plansnapshot.Snapshot, error) {
		facts, err := access.observe(ctx, binding)
		if err != nil {
			if ctx != nil {
				if contextErr := ctx.Err(); contextErr != nil {
					return plansnapshot.Snapshot{}, contextErr
				}
			}
			if !facts.root.available {
				return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
			}
			facts.tasks.markIncomplete()
		}
		return binding.scheme.snapshot(facts)
	}, nil
}

func (s scheme) snapshot(facts githubFactSet) (plansnapshot.Snapshot, error) {
	if facts.binding.scheme != s || !facts.root.available || facts.root.fact.number != facts.binding.number.value {
		return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
	}
	state := progressState(facts.root.fact.state)
	titles, complete := s.taskTitles(facts.tasks)
	progressTasks := make([]plansnapshot.TaskProgress, 0, len(titles))
	for _, member := range s.taskMembersInPlanOrder(facts.tasks) {
		if member.pullRequest {
			continue
		}
		progress, err := plansnapshot.NewTaskProgress(member.title, progressState(member.state))
		if err != nil {
			complete = false
			continue
		}
		progressTasks = append(progressTasks, progress)
	}
	var progress plansnapshot.ProgressEvidence
	var err error
	if complete {
		progress, err = plansnapshot.CompleteProgress(state, progressTasks)
	} else {
		progress, err = plansnapshot.IncompleteProgress(state, progressTasks)
	}
	if err != nil {
		return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
	}

	current, eligible := s.currentPlan(facts)
	if !eligible {
		return plansnapshot.WithoutCurrent(progress)
	}
	snapshot, err := plansnapshot.New(current, progress)
	if err != nil {
		return plansnapshot.WithoutCurrent(progress)
	}
	return snapshot, nil
}

func progressState(state nativeState) plansnapshot.ProgressState {
	switch state {
	case nativeStateOpen:
		return plansnapshot.Open
	case nativeStateClosed:
		return plansnapshot.Closed
	default:
		return plansnapshot.Unknown
	}
}

func (s scheme) currentPlan(facts githubFactSet) (plan.Plan, bool) {
	root := facts.root.fact
	if !root.title.available {
		return plan.Plan{}, false
	}
	narrative := s.establishNarrativeFacts(root.content)
	if !narrative.goal.available || !narrative.conditions.complete {
		return plan.Plan{}, false
	}
	goal, err := plan.NewGoal(narrative.goal.value)
	if err != nil {
		return plan.Plan{}, false
	}
	conditions := make([]plan.AcceptanceCondition, 0, len(narrative.conditions.members))
	for _, member := range narrative.conditions.members {
		condition, err := plan.NewAcceptanceCondition(member)
		if err != nil {
			return plan.Plan{}, false
		}
		conditions = append(conditions, condition)
	}
	titles, complete := s.taskTitles(facts.tasks)
	if !complete {
		return plan.Plan{}, false
	}
	tasks := make([]plan.Task, 0, len(titles))
	for _, title := range titles {
		task, err := plan.NewTask(title)
		if err != nil {
			return plan.Plan{}, false
		}
		tasks = append(tasks, task)
	}
	var targetDate *plan.TargetDate
	if s == milestoneScheme {
		state, value := root.date.targetDateText()
		switch state {
		case dateAbsent:
		case datePresent:
			parsed, err := plan.ParseTargetDate(value)
			if err != nil {
				return plan.Plan{}, false
			}
			targetDate = &parsed
		default:
			return plan.Plan{}, false
		}
	}
	value, err := plan.New(root.title.value, goal, conditions, tasks, targetDate)
	if err != nil {
		return plan.Plan{}, false
	}
	return value, true
}
