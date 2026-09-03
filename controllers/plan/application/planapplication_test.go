package planapplication_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kotokumu/arcloom/arcloom/authorization"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/application"
)

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func plans() (plan.Plan, plan.Plan) {
	current := must(plan.New("Current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	proposed := must(plan.New("Proposed", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	return current, proposed
}

func TestTargetRevisionAndRequestPreserveExactMeaning(t *testing.T) {
	target := must(planapplication.NewTargetReference("github", "owner/repo#1"))
	current, proposed := plans()
	revision := must(planapplication.NewRevision(target, current, proposed))
	policy := must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	var received planapplication.Request
	result, err := planapplication.RequestApplication(context.Background(), revision, policy, func(_ context.Context, request planapplication.Request) planapplication.ReceiptEvidence {
		received = request
		return planapplication.AcknowledgedReceipt("receipt-1")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("RequestApplication() error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, revision.Equal(received.Revision())); diff != "" {
		t.Errorf("Actor revision mismatch (-want +got):\n%s", diff)
	}
	receipt, hasReceipt := result.Receipt()
	if diff := cmp.Diff(true, hasReceipt); diff != "" {
		t.Fatalf("receipt presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(planapplication.ReceiptAcknowledged, receipt.Kind()); diff != "" {
		t.Errorf("receipt kind mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestApplicationAuthorizationOutcomes(t *testing.T) {
	current, proposed := plans()
	revision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	tests := []struct {
		name   string
		policy authorization.Policy[planapplication.Revision]
		want   authorization.Decision
	}{
		{name: "denied", policy: must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
			return authorization.Deny, nil
		})), want: authorization.Denied},
		{name: "undecidable", policy: must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
			return authorization.Unknown, nil
		})), want: authorization.Undecidable},
		{name: "invalid policy", policy: authorization.Policy[planapplication.Revision]{}, want: authorization.Undecidable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			result, err := planapplication.RequestApplication(context.Background(), revision, tt.policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
				calls++
				return planapplication.AcknowledgedReceipt("unexpected")
			})
			if diff := cmp.Diff(nil, err); diff != "" {
				t.Fatalf("RequestApplication() error mismatch (-want +got):\n%s", diff)
			}
			decision, hasDecision := result.AuthorizationDecision()
			if diff := cmp.Diff(true, hasDecision); diff != "" {
				t.Fatalf("decision presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, decision); diff != "" {
				t.Errorf("decision mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, calls); diff != "" {
				t.Errorf("Actor invocation count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestApplicationFailurePrecedence(t *testing.T) {
	current, proposed := plans()
	validRevision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	policy := must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	tests := []struct {
		name  string
		rev   planapplication.Revision
		actor planapplication.Actor
		code  planapplication.FailureCode
	}{
		{name: "invalid revision", actor: func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.NotReceivedReceipt()
		}, code: planapplication.InvalidRevision},
		{name: "invalid actor", rev: validRevision, code: planapplication.InvalidActor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planapplication.RequestApplication(context.Background(), tt.rev, policy, tt.actor)
			var failure planapplication.Failure
			if diff := cmp.Diff(true, errors.As(err, &failure)); diff != "" {
				t.Fatalf("Failure type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.code, failure.Code()); diff != "" {
				t.Errorf("Failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestApplicationReceiptAndCancellationSemantics(t *testing.T) {
	current, proposed := plans()
	revision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	policy := must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	tests := []struct {
		name  string
		actor planapplication.Actor
		want  planapplication.ReceiptKind
	}{
		{name: "acknowledged", actor: func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.AcknowledgedReceipt("ack")
		}, want: planapplication.ReceiptAcknowledged},
		{name: "refused", actor: func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.RefusedReceipt("refused")
		}, want: planapplication.ReceiptRefused},
		{name: "known not received", actor: func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.NotReceivedReceipt()
		}, want: planapplication.KnownNotReceived},
		{name: "zero uncertain", actor: func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.ReceiptEvidence{}
		}, want: planapplication.ReceiptUncertain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := planapplication.RequestApplication(context.Background(), revision, policy, tt.actor)
			if diff := cmp.Diff(nil, err); diff != "" {
				t.Fatalf("RequestApplication() error mismatch (-want +got):\n%s", diff)
			}
			receipt, hasReceipt := result.Receipt()
			if diff := cmp.Diff(true, hasReceipt); diff != "" {
				t.Fatalf("receipt presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, receipt.Kind()); diff != "" {
				t.Errorf("receipt kind mismatch (-want +got):\n%s", diff)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	cancelled, err := planapplication.RequestApplication(ctx, revision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		calls++
		return planapplication.AcknowledgedReceipt("not-called")
	})
	if diff := cmp.Diff(context.Canceled, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("entry cancellation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, calls); diff != "" {
		t.Errorf("entry cancellation Actor calls mismatch (-want +got):\n%s", diff)
	}
	if _, hasReceipt := cancelled.Receipt(); hasReceipt {
		t.Error("entry cancellation unexpectedly established receipt")
	}
}

func TestRequestApplicationRepeatedConcurrentIsolationAndNoRetry(t *testing.T) {
	current, firstPlan := plans()
	secondPlan := must(plan.New("Second", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	target := must(planapplication.NewTargetReference("github", "id"))
	firstRevision := must(planapplication.NewRevision(target, current, firstPlan))
	secondRevision := must(planapplication.NewRevision(target, current, secondPlan))
	policy := must(authorization.NewPolicy(func(_ context.Context, subject planapplication.Revision) (authorization.RuleConclusion, error) {
		if subject.Proposed().Name() == "Proposed" {
			return authorization.Permit, nil
		}
		return authorization.Deny, nil
	}))
	calls := 0
	first, err := planapplication.RequestApplication(context.Background(), firstRevision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		calls++
		return planapplication.ReceiptEvidence{}
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("first application error mismatch (-want +got):\n%s", diff)
	}
	second, err := planapplication.RequestApplication(context.Background(), secondRevision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		calls++
		return planapplication.AcknowledgedReceipt("unexpected")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("second application error mismatch (-want +got):\n%s", diff)
	}
	if receipt, ok := first.Receipt(); !ok || receipt.Kind() != planapplication.ReceiptUncertain {
		t.Error("first invocation did not preserve uncertainty")
	}
	if decision, ok := second.AuthorizationDecision(); !ok || decision != authorization.Denied {
		t.Error("second invocation did not preserve current denial")
	}
	if diff := cmp.Diff(1, calls); diff != "" {
		t.Errorf("Actor retry/invocation count mismatch (-want +got):\n%s", diff)
	}

	results := make(chan planapplication.Result, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		result, _ := planapplication.RequestApplication(context.Background(), firstRevision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.AcknowledgedReceipt("first")
		})
		results <- result
	}()
	go func() {
		defer wait.Done()
		result, _ := planapplication.RequestApplication(context.Background(), secondRevision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.AcknowledgedReceipt("not-called")
		})
		results <- result
	}()
	wait.Wait()
	close(results)
	seenReceipt, seenDecision := false, false
	for result := range results {
		if receipt, ok := result.Receipt(); ok && receipt.Kind() == planapplication.ReceiptAcknowledged {
			seenReceipt = true
		}
		if decision, ok := result.AuthorizationDecision(); ok && decision == authorization.Denied {
			seenDecision = true
		}
	}
	if diff := cmp.Diff(true, seenReceipt && seenDecision); diff != "" {
		t.Errorf("concurrent isolation mismatch (-want +got):\n%s", diff)
	}
}
