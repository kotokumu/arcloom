package controlruntime_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
)

func TestControllerResultIsolation(t *testing.T) {
	t.Run("unrelated target values use the same controller implementation", func(t *testing.T) {
		type snapshotWithoutCurrentPlan struct{ Progress string }
		type assessedPlan struct{ Assessment string }

		snapshotContext, cancelSnapshot := context.WithCancel(context.Background())
		snapshotAttempt := controlruntime.Attempt[snapshotWithoutCurrentPlan](func(context.Context, controlruntime.TargetIdentity) (snapshotWithoutCurrentPlan, controlruntime.Directive, error) {
			return snapshotWithoutCurrentPlan{Progress: "milestone observed"}, controlruntime.AwaitAnotherRequest(), nil
		})
		snapshotController := must(controlruntime.Start(snapshotContext, 1, snapshotAttempt))
		snapshotTarget := must(controlruntime.NewTargetIdentity("plan", "snapshot-only"))
		if err := snapshotController.Request(context.Background(), snapshotTarget); err != nil {
			t.Fatalf("snapshot Request() error = %v", err)
		}

		assessmentContext, cancelAssessment := context.WithCancel(context.Background())
		assessmentAttempt := controlruntime.Attempt[assessedPlan](func(context.Context, controlruntime.TargetIdentity) (assessedPlan, controlruntime.Directive, error) {
			return assessedPlan{Assessment: "retain"}, controlruntime.AwaitAnotherRequest(), nil
		})
		assessmentController := must(controlruntime.Start(assessmentContext, 1, assessmentAttempt))
		assessmentTarget := must(controlruntime.NewTargetIdentity("plan", "assessed"))
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
			if diff := cmp.Diff(controlruntime.AwaitRequest, directive.Kind()); diff != "" {
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
			if diff := cmp.Diff(controlruntime.AwaitRequest, directive.Kind()); diff != "" {
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
		attempt  controlruntime.Attempt[string]
		wantKind controlruntime.FailureKind
		wantErr  error
	}{
		{
			name: "target attempt failed",
			attempt: func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
				return "ignored", controlruntime.AwaitAnotherRequest(), controlruntime.ErrInvalidTargetIdentity
			},
			wantKind: controlruntime.TargetAttemptFailed,
			wantErr:  controlruntime.ErrInvalidTargetIdentity,
		},
		{
			name: "control directive rejected",
			attempt: func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
				return "ignored", controlruntime.Directive{}, nil
			},
			wantKind: controlruntime.ControlDirectiveRejected,
			wantErr:  controlruntime.ErrInvalidDirective,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			controller := must(controlruntime.Start(ctx, 1, tt.attempt))
			target := must(controlruntime.NewTargetIdentity("test", tt.name))
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
		directive controlruntime.Directive
	}{
		{name: "valid immediate directive", directive: controlruntime.ImmediateReevaluation()},
		{name: "invalid zero directive", directive: controlruntime.Directive{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cause := errors.New("attempt boundary failed")
			calls := 0
			attempt := controlruntime.Attempt[string](func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
				calls++
				return "must not be published", tt.directive, cause
			})
			controller := must(controlruntime.Start(ctx, 1, attempt))
			target := must(controlruntime.NewTargetIdentity("test", tt.name))
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
			if !ok || failure.Kind() != controlruntime.TargetAttemptFailed || !errors.Is(failure, cause) {
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
