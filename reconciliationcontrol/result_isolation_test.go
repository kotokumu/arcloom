package reconciliationcontrol_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/reconciliationcontrol"
)

func TestControllerResultIsolation(t *testing.T) {
	t.Run("unrelated target values use the same controller implementation", func(t *testing.T) {
		type snapshotWithoutCurrentPlan struct{ Progress string }
		type assessedPlan struct{ Assessment string }

		snapshotContext, cancelSnapshot := context.WithCancel(context.Background())
		snapshotAttempt := reconciliationcontrol.Attempt[snapshotWithoutCurrentPlan](func(context.Context, reconciliationcontrol.TargetIdentity) (snapshotWithoutCurrentPlan, reconciliationcontrol.Directive, error) {
			return snapshotWithoutCurrentPlan{Progress: "milestone observed"}, reconciliationcontrol.AwaitAnotherRequest(), nil
		})
		snapshotController := must(reconciliationcontrol.Start(snapshotContext, 1, snapshotAttempt))
		snapshotTarget := must(reconciliationcontrol.NewTargetIdentity("plan", "snapshot-only"))
		if err := snapshotController.Request(context.Background(), snapshotTarget); err != nil {
			t.Fatalf("snapshot Request() error = %v", err)
		}

		assessmentContext, cancelAssessment := context.WithCancel(context.Background())
		assessmentAttempt := reconciliationcontrol.Attempt[assessedPlan](func(context.Context, reconciliationcontrol.TargetIdentity) (assessedPlan, reconciliationcontrol.Directive, error) {
			return assessedPlan{Assessment: "retain"}, reconciliationcontrol.AwaitAnotherRequest(), nil
		})
		assessmentController := must(reconciliationcontrol.Start(assessmentContext, 1, assessmentAttempt))
		assessmentTarget := must(reconciliationcontrol.NewTargetIdentity("plan", "assessed"))
		if err := assessmentController.Request(context.Background(), assessmentTarget); err != nil {
			t.Fatalf("assessment Request() error = %v", err)
		}

		select {
		case report := <-snapshotController.Reports():
			if diff := cmp.Diff(snapshotTarget.Kind(), report.Target().Kind()); diff != "" {
				t.Errorf("snapshot target kind mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(snapshotTarget.Key(), report.Target().Key()); diff != "" {
				t.Errorf("snapshot target key mismatch (-want +got):\n%s", diff)
			}
			got, directive, ok := report.Completion()
			if diff := cmp.Diff(snapshotWithoutCurrentPlan{Progress: "milestone observed"}, got); diff != "" {
				t.Errorf("snapshot value mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(reconciliationcontrol.AwaitRequest, directive.Kind()); diff != "" {
				t.Errorf("snapshot Directive mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, ok); diff != "" {
				t.Errorf("snapshot Completion presence mismatch (-want +got):\n%s", diff)
			}
		case <-time.After(time.Second):
			t.Fatal("snapshot Report was not published")
		}

		select {
		case report := <-assessmentController.Reports():
			got, directive, ok := report.Completion()
			if diff := cmp.Diff(assessedPlan{Assessment: "retain"}, got); diff != "" {
				t.Errorf("assessment value mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(reconciliationcontrol.AwaitRequest, directive.Kind()); diff != "" {
				t.Errorf("assessment Directive mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, ok); diff != "" {
				t.Errorf("assessment Completion presence mismatch (-want +got):\n%s", diff)
			}
		case <-time.After(time.Second):
			t.Fatal("assessment Report was not published")
		}

		cancelSnapshot()
		cancelAssessment()
		if err := snapshotController.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("snapshot Wait() error = %v, want context.Canceled", err)
		}
		if err := assessmentController.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("assessment Wait() error = %v, want context.Canceled", err)
		}
	})
}

func TestControllerFailureKinds(t *testing.T) {
	tests := []struct {
		name     string
		attempt  reconciliationcontrol.Attempt[string]
		wantKind reconciliationcontrol.FailureKind
		wantErr  error
	}{
		{
			name: "target attempt failed",
			attempt: func(context.Context, reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
				return "ignored", reconciliationcontrol.AwaitAnotherRequest(), reconciliationcontrol.ErrInvalidTargetIdentity
			},
			wantKind: reconciliationcontrol.TargetAttemptFailed,
			wantErr:  reconciliationcontrol.ErrInvalidTargetIdentity,
		},
		{
			name: "control directive rejected",
			attempt: func(context.Context, reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
				return "ignored", reconciliationcontrol.Directive{}, nil
			},
			wantKind: reconciliationcontrol.ControlDirectiveRejected,
			wantErr:  reconciliationcontrol.ErrInvalidDirective,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			controller := must(reconciliationcontrol.Start(ctx, 1, tt.attempt))
			target := must(reconciliationcontrol.NewTargetIdentity("test", tt.name))
			if err := controller.Request(context.Background(), target); err != nil {
				t.Fatalf("Request() error = %v", err)
			}
			select {
			case report := <-controller.Reports():
				failure, ok := report.Failure()
				if diff := cmp.Diff(true, ok); diff != "" {
					t.Errorf("Failure presence mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tt.wantKind, failure.Kind()); diff != "" {
					t.Errorf("Failure kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(true, errors.Is(failure, tt.wantErr)); diff != "" {
					t.Errorf("Failure cause mismatch (-want +got):\n%s", diff)
				}
				_, _, completed := report.Completion()
				if diff := cmp.Diff(false, completed); diff != "" {
					t.Errorf("Completion presence mismatch (-want +got):\n%s", diff)
				}
			case <-time.After(time.Second):
				t.Fatal("Failure Report was not published")
			}
			cancel()
			if err := controller.Wait(); !errors.Is(err, context.Canceled) {
				t.Errorf("Wait() error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestTargetAttemptFailureOutranksReturnedValueAndDirectiveWithoutRetry(t *testing.T) {
	tests := []struct {
		name      string
		directive reconciliationcontrol.Directive
	}{
		{name: "valid immediate directive", directive: reconciliationcontrol.ImmediateReevaluation()},
		{name: "invalid zero directive", directive: reconciliationcontrol.Directive{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cause := errors.New("attempt boundary failed")
			calls := 0
			attempt := reconciliationcontrol.Attempt[string](func(context.Context, reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
				calls++
				return "must not be published", tt.directive, cause
			})
			controller := must(reconciliationcontrol.Start(ctx, 1, attempt))
			target := must(reconciliationcontrol.NewTargetIdentity("test", tt.name))
			if err := controller.Request(context.Background(), target); err != nil {
				t.Fatalf("Request() error = %v", err)
			}
			report := <-controller.Reports()
			if report.Target() != target {
				t.Errorf("Report target = %v/%v, want %v/%v", report.Target().Kind(), report.Target().Key(), target.Kind(), target.Key())
			}
			if result, directive, ok := report.Completion(); ok || result != "" || directive.Kind() != 0 {
				t.Errorf("Completion() = (%q, %v, %v), want absent zero values", result, directive.Kind(), ok)
			}
			failure, ok := report.Failure()
			if !ok || failure.Kind() != reconciliationcontrol.TargetAttemptFailed || !errors.Is(failure, cause) {
				t.Errorf("Failure() = (%v, %v), want TargetAttemptFailed wrapping cause", failure, ok)
			}
			cancel()
			_ = controller.Wait()
			if calls != 1 {
				t.Errorf("Attempt calls = %d, want no implicit retry", calls)
			}
		})
	}
}
