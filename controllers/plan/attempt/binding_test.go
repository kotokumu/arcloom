package planattempt_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

// Table shape comes from gotests; assertions use public binding accessors.
func TestNewPlanAttemptBindingRejectsInvalidConfiguration(t *testing.T) {
	type args struct {
		target   planattempt.PlanTarget[string]
		assessor plancontrol.Assessor[string]
	}
	tests := []struct {
		name    string
		args    args
		want    planattempt.PlanAttemptBinding
		wantErr bool
	}{
		{name: "zero target and nil assessor", want: planattempt.PlanAttemptBinding{}, wantErr: true},
		{name: "zero target", args: args{assessor: func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
			t.Fatal("construction assessed")
			return plancontrol.AssessorResponse{}, nil
		}}, want: planattempt.PlanAttemptBinding{}, wantErr: true},
		{name: "nil assessor", args: args{target: must(planattempt.NewPlanTarget(must(controlruntime.NewTargetIdentity("plan", "one")), func(context.Context) (plansnapshot.Snapshot, error) {
			t.Fatal("construction observed")
			return plansnapshot.Snapshot{}, nil
		}, func(context.Context) (string, error) { t.Fatal("construction delivery observed"); return "", nil }))}, want: planattempt.PlanAttemptBinding{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planattempt.NewPlanAttemptBinding(tt.args.target, tt.args.assessor)
			if (err != nil) != tt.wantErr || !errors.Is(err, planattempt.ErrInvalidPlanAttemptBinding) {
				t.Fatalf("error=%v", err)
			}
			if diff := cmp.Diff(tt.want.Identity(), got.Identity(), cmp.AllowUnexported(controlruntime.TargetIdentity{})); diff != "" {
				t.Fatal(diff)
			}
			if got.Attempt() != nil {
				t.Fatal("invalid binding has executable")
			}
		})
	}
}

func TestPlanAttemptBindingPreservesExactTarget(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "one"))
	calls := 0
	snapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { calls++; return snapshot, nil }, func(context.Context) (string, error) { t.Fatal("unexpected delivery observation"); return "", nil }))
	binding := must(planattempt.NewPlanAttemptBinding(target, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
		t.Fatal("unexpected assessment")
		return plancontrol.AssessorResponse{}, nil
	}))
	if calls != 0 {
		t.Fatal("construction observed")
	}
	if diff := cmp.Diff(identity, binding.Identity(), cmp.AllowUnexported(controlruntime.TargetIdentity{})); diff != "" {
		t.Fatal(diff)
	}
	for _, other := range []controlruntime.TargetIdentity{must(controlruntime.NewTargetIdentity("plan", "two")), must(controlruntime.NewTargetIdentity("other", "one"))} {
		_, _, err := binding.Attempt()(context.Background(), other)
		var failure *planattempt.FailureError
		if !errors.As(err, &failure) || failure.Code() != planattempt.TargetBindingUnavailable || calls != 0 {
			t.Fatalf("mismatch err=%v calls=%d", err, calls)
		}
	}
	result, directive, err := binding.Attempt()(context.Background(), identity)
	if err != nil || result.Kind() != planattempt.CurrentPlanNotEstablished || directive.Kind() != controlruntime.AwaitRequest || calls != 1 {
		t.Fatalf("result=%v directive=%v err=%v calls=%d", result.Kind(), directive.Kind(), err, calls)
	}
}
