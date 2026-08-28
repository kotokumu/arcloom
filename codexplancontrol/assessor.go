package codexplancontrol

import (
	"context"
	"errors"

	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

var (
	errInvalidConfiguration      = errors.New("codex plan control: invalid configuration")
	errInvalidObservationEncoder = errors.New("codex plan control: invalid observation encoder")
	errAssessmentUnavailable     = errors.New("codex plan control: assessment unavailable")
)

// ObservationEncoder preserves caller-owned observation vocabulary while
// making it available to one Codex assessment interaction. It must not mutate
// or retain state reachable from the supplied observation, and must return
// after ctx is cancelled.
type ObservationEncoder[O any] func(context.Context, O) (string, error)

// NewAssessor validates local configuration without starting Codex. Each call
// to the returned Assessor encodes the supplied Plan and observation for one
// disposable read-only Codex interaction. Provider transmission, retention,
// telemetry, token use, and cost remain properties of the Host-installed
// Codex service. The interaction neither authorizes nor applies a Plan change.
func NewAssessor[O any](
	configuration Configuration,
	encode ObservationEncoder[O],
) (plancontrol.Assessor[O], error) {
	if !configuration.isValid() {
		return nil, errInvalidConfiguration
	}
	if encode == nil {
		return nil, errInvalidObservationEncoder
	}
	return func(ctx context.Context, current plan.Plan, observations O) (plancontrol.AssessorResponse, error) {
		if ctx == nil {
			return plancontrol.AssessorResponse{}, errAssessmentUnavailable
		}
		if err := ctx.Err(); err != nil {
			return plancontrol.AssessorResponse{}, err
		}
		if !current.IsValid() {
			return plancontrol.AssessorResponse{}, errAssessmentUnavailable
		}
		encoded, err := encode(ctx, observations)
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return plancontrol.AssessorResponse{}, contextErr
			}
			return plancontrol.AssessorResponse{}, errAssessmentUnavailable
		}
		if err := ctx.Err(); err != nil {
			return plancontrol.AssessorResponse{}, err
		}
		return assessWithCodex(ctx, configuration, current, encoded)
	}, nil
}
