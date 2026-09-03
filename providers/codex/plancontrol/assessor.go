package codexplancontrol

import (
	"context"
	"errors"
	"reflect"

	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/providers/codex/appserver"
)

var (
	errInvalidConfiguration      = errors.New("codex plan control: invalid configuration")
	errInvalidClient             = errors.New("codex plan control: invalid app-server Client")
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
	client codexappserver.Client,
	configuration Configuration,
	encode ObservationEncoder[O],
) (plancontrol.Assessor[O], error) {
	if client == nil {
		return nil, errInvalidClient
	}
	clientValue := reflect.ValueOf(client)
	switch clientValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if clientValue.IsNil() {
			return nil, errInvalidClient
		}
	}
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
		return assessWithCodex(ctx, client, configuration, current, encoded)
	}, nil
}
