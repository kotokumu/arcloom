package codexplancontrol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/codexappserver"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

type recordingClient struct {
	requests []codexappserver.ReadOnlyTurnRequest
	complete func(context.Context, codexappserver.ReadOnlyTurnRequest) (codexappserver.CompletedTurn, error)
}

func (c *recordingClient) CompleteReadOnlyTurn(
	ctx context.Context,
	request codexappserver.ReadOnlyTurnRequest,
) (codexappserver.CompletedTurn, error) {
	c.requests = append(c.requests, request)
	if c.complete == nil {
		return codexappserver.CompletedTurn{}, errors.New("unexpected Client invocation")
	}
	return c.complete(ctx, request)
}

func TestNewAssessor(t *testing.T) {
	type args struct {
		client        func() codexappserver.Client
		configuration Configuration
		encode        ObservationEncoder[int]
	}
	workingDirectory := must(NewWorkingDirectory(t.TempDir()))
	validConfiguration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		workingDirectory,
	)
	validEncoder := ObservationEncoder[int](func(context.Context, int) (string, error) { return "observation", nil })
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "valid inputs", args: args{client: func() codexappserver.Client { return &recordingClient{} }, configuration: validConfiguration, encode: validEncoder}},
		{name: "nil Client", args: args{client: func() codexappserver.Client { return nil }, configuration: validConfiguration, encode: validEncoder}, wantErr: true},
		{name: "typed nil Client", args: args{client: func() codexappserver.Client { return (*recordingClient)(nil) }, configuration: validConfiguration, encode: validEncoder}, wantErr: true},
		{name: "zero configuration", args: args{client: func() codexappserver.Client { return &recordingClient{} }, encode: validEncoder}, wantErr: true},
		{name: "invalid model", args: args{client: func() codexappserver.Client { return &recordingClient{} }, configuration: NewConfiguration(Model{}, must(NewReasoningEffort("high")), workingDirectory), encode: validEncoder}, wantErr: true},
		{name: "invalid reasoning effort", args: args{client: func() codexappserver.Client { return &recordingClient{} }, configuration: NewConfiguration(must(NewModel("gpt-5.6-sol")), ReasoningEffort{}, workingDirectory), encode: validEncoder}, wantErr: true},
		{name: "invalid working directory", args: args{client: func() codexappserver.Client { return &recordingClient{} }, configuration: NewConfiguration(must(NewModel("gpt-5.6-sol")), must(NewReasoningEffort("high")), WorkingDirectory{}), encode: validEncoder}, wantErr: true},
		{name: "nil observation encoder", args: args{client: func() codexappserver.Client { return &recordingClient{} }, configuration: validConfiguration}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAssessor[int](tt.args.client(), tt.args.configuration, tt.args.encode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewAssessor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got == nil {
				t.Fatal("NewAssessor() returned nil Assessor")
			}
		})
	}
}

func TestNewAssessor_invalidInvocationStartsNoInteraction(t *testing.T) {
	configuration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		must(NewWorkingDirectory(t.TempDir())),
	)
	current := must(plan.New(
		"Plan",
		must(plan.NewGoal("Goal")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Accepted"))},
		nil,
		nil,
	))
	encoderFailure := errors.New("private encoder failure")
	tests := []struct {
		name    string
		ctx     func() context.Context
		current plan.Plan
		encode  ObservationEncoder[string]
		wantErr error
	}{
		{name: "nil context", ctx: func() context.Context { return nil }, current: current, wantErr: errAssessmentUnavailable},
		{name: "invalid current Plan", ctx: context.Background, wantErr: errAssessmentUnavailable},
		{name: "pre-cancelled context", ctx: cancelledContext, current: current, wantErr: context.Canceled},
		{name: "expired deadline", ctx: expiredContext, current: current, wantErr: context.DeadlineExceeded},
		{name: "encoder failure", ctx: context.Background, current: current, encode: func(context.Context, string) (string, error) { return "", encoderFailure }, wantErr: errAssessmentUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &recordingClient{}
			encode := tt.encode
			if encode == nil {
				encode = func(context.Context, string) (string, error) { return "observation", nil }
			}
			assessor := must(NewAssessor(client, configuration, encode))
			got, err := assessor(tt.ctx(), tt.current, "observation")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Assessor() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(plancontrol.AssessorResponse{}, got); diff != "" {
				t.Errorf("Assessor() response mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(client.requests)); diff != "" {
				t.Errorf("Client invocation count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
