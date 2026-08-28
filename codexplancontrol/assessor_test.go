package codexplancontrol

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

func TestNewAssessor(t *testing.T) {
	type args struct {
		configuration Configuration
		encode        ObservationEncoder[int]
	}
	workingDirectory := must(NewWorkingDirectory(t.TempDir()))
	validConfiguration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		workingDirectory,
		must(NewShutdownGrace(time.Second)),
	)
	validEncoder := ObservationEncoder[int](func(context.Context, int) (string, error) { return "observation", nil })
	tests := []struct {
		name    string
		args    args
		want    plancontrol.Assessor[int]
		wantErr bool
	}{
		{
			name: "valid configuration starts no interaction",
			args: args{configuration: validConfiguration, encode: validEncoder},
			want: func(ctx context.Context, _ plan.Plan, _ int) (plancontrol.AssessorResponse, error) {
				return plancontrol.AssessorResponse{}, ctx.Err()
			},
		},
		{name: "zero configuration", args: args{encode: validEncoder}, wantErr: true},
		{name: "invalid model", args: args{configuration: NewConfiguration(Model{}, must(NewReasoningEffort("high")), workingDirectory, must(NewShutdownGrace(time.Second))), encode: validEncoder}, wantErr: true},
		{name: "invalid reasoning effort", args: args{configuration: NewConfiguration(must(NewModel("gpt-5.6-sol")), ReasoningEffort{}, workingDirectory, must(NewShutdownGrace(time.Second))), encode: validEncoder}, wantErr: true},
		{name: "invalid working directory", args: args{configuration: NewConfiguration(must(NewModel("gpt-5.6-sol")), must(NewReasoningEffort("high")), WorkingDirectory{}, must(NewShutdownGrace(time.Second))), encode: validEncoder}, wantErr: true},
		{name: "invalid shutdown grace", args: args{configuration: NewConfiguration(must(NewModel("gpt-5.6-sol")), must(NewReasoningEffort("high")), workingDirectory, ShutdownGrace{}), encode: validEncoder}, wantErr: true},
		{name: "nil observation encoder", args: args{configuration: validConfiguration}, wantErr: true},
	}
	t.Setenv("PATH", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAssessor[int](tt.args.configuration, tt.args.encode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewAssessor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			current := must(plan.New(
				"Plan",
				must(plan.NewGoal("Goal")),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Accepted"))},
				nil,
				nil,
			))
			wantResponse, wantCallErr := tt.want(ctx, current, 1)
			gotResponse, gotCallErr := got(ctx, current, 1)
			if diff := cmp.Diff(wantResponse, gotResponse); diff != "" {
				t.Errorf("NewAssessor() response mismatch (-want +got):\n%s", diff)
			}
			if !errors.Is(gotCallErr, wantCallErr) {
				t.Errorf("NewAssessor() call error = %v, want %v", gotCallErr, wantCallErr)
			}
		})
	}
}

func TestNewAssessor_invalidInvocationStartsNoInteraction(t *testing.T) {
	helperDirectory := t.TempDir()
	helperPath := filepath.Join(helperDirectory, "codex")
	build := exec.Command("go", "build", "-o", helperPath, "./testdata/codexstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex contract helper: %v\n%s", err, output)
	}
	t.Setenv("PATH", helperDirectory)
	configuration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		must(NewWorkingDirectory(t.TempDir())),
		must(NewShutdownGrace(time.Second)),
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
			launchPath := filepath.Join(t.TempDir(), "launched")
			t.Setenv("CODEX_STUB_LAUNCH", launchPath)
			encode := tt.encode
			if encode == nil {
				encode = func(context.Context, string) (string, error) { return "observation", nil }
			}
			assessor := must(NewAssessor(configuration, encode))
			_, err := assessor(tt.ctx(), tt.current, "observation")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Assessor() error = %v, want %v", err, tt.wantErr)
			}
			if _, err := os.Stat(launchPath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Codex process launch evidence = %v, want no launch", err)
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
