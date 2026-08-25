package planapplication_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kotokumu/arcloom/authorization"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planapplication"
)

func TestRequestApplicationBindsExactAuthorizationSubject(t *testing.T) {
	current := must(planapplication.NewTargetReference("github", "owner/repo#1"))
	currentPlan, proposedPlan := plans()
	revision := must(planapplication.NewRevision(current, currentPlan, proposedPlan))
	var observed planapplication.Revision
	policy := must(authorization.NewPolicy(func(_ context.Context, subject planapplication.Revision) (authorization.RuleConclusion, error) {
		observed = subject
		return authorization.Permit, nil
	}))
	result, err := planapplication.RequestApplication(context.Background(), revision, policy, func(_ context.Context, request planapplication.Request) planapplication.ReceiptEvidence {
		return planapplication.AcknowledgedReceipt("native:42")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Target().Context(), observed.Target().Context()); diff != "" {
		t.Errorf("authorization target context mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Target().Identity(), observed.Target().Identity()); diff != "" {
		t.Errorf("authorization target identity mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Current().Name(), observed.Current().Name()); diff != "" {
		t.Errorf("authorization current name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Current().Goal().Text(), observed.Current().Goal().Text()); diff != "" {
		t.Errorf("authorization current goal mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Proposed().Name(), observed.Proposed().Name()); diff != "" {
		t.Errorf("authorization proposed name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(revision.Proposed().Goal().Text(), observed.Proposed().Goal().Text()); diff != "" {
		t.Errorf("authorization proposed goal mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, revision.Current().Equal(observed.Current())); diff != "" {
		t.Errorf("authorization current semantic mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, revision.Proposed().Equal(observed.Proposed())); diff != "" {
		t.Errorf("authorization proposed semantic mismatch (-want +got):\n%s", diff)
	}
	decision, hasDecision := result.AuthorizationDecision()
	receipt, hasReceipt := result.Receipt()
	if diff := cmp.Diff(false, hasDecision); diff != "" {
		t.Errorf("authorized result decision presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, hasReceipt); diff != "" {
		t.Errorf("authorized result receipt presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, decision); diff != "" {
		t.Errorf("hidden decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(planapplication.ReceiptAcknowledged, receipt.Kind()); diff != "" {
		t.Errorf("receipt kind mismatch (-want +got):\n%s", diff)
	}
	if reference, ok := receipt.Reference(); !ok {
		t.Error("acknowledged receipt reference was not exposed")
	} else if diff := cmp.Diff("native:42", reference); diff != "" {
		t.Errorf("receipt reference mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestApplicationFullFailurePrecedence(t *testing.T) {
	current, proposed := plans()
	validRevision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name      string
		ctx       context.Context
		revision  planapplication.Revision
		policy    authorization.Policy[planapplication.Revision]
		actor     planapplication.Actor
		wantError error
		wantCode  planapplication.FailureCode
	}{
		{name: "entry cancellation overrides every invalid input", ctx: cancelled, revision: planapplication.Revision{}, policy: authorization.Policy[planapplication.Revision]{}, actor: nil, wantError: context.Canceled},
		{name: "invalid revision overrides nil actor and invalid policy", ctx: context.Background(), revision: planapplication.Revision{}, policy: authorization.Policy[planapplication.Revision]{}, actor: nil, wantCode: planapplication.InvalidRevision},
		{name: "nil actor overrides invalid policy", ctx: context.Background(), revision: validRevision, policy: authorization.Policy[planapplication.Revision]{}, actor: nil, wantCode: planapplication.InvalidActor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			actor := tt.actor
			if actor == nil && tt.wantCode == "" {
				actor = func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
					calls++
					return planapplication.AcknowledgedReceipt("unexpected")
				}
			}
			result, err := planapplication.RequestApplication(tt.ctx, tt.revision, tt.policy, actor)
			if tt.wantError != nil {
				if diff := cmp.Diff(tt.wantError, err, cmpopts.EquateErrors()); diff != "" {
					t.Fatalf("error mismatch (-want +got):\n%s", diff)
				}
			} else {
				var failure planapplication.Failure
				if diff := cmp.Diff(true, errors.As(err, &failure)); diff != "" {
					t.Fatalf("failure type mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tt.wantCode, failure.Code()); diff != "" {
					t.Errorf("failure code mismatch (-want +got):\n%s", diff)
				}
			}
			if _, present := result.AuthorizationDecision(); present {
				t.Error("failure result exposed a decision")
			}
			if _, present := result.Receipt(); present {
				t.Error("failure result exposed a receipt")
			}
			if diff := cmp.Diff(0, calls); diff != "" {
				t.Errorf("Actor calls mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestApplicationInFlightAuthorizationCancellation(t *testing.T) {
	current, proposed := plans()
	revision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	entered := make(chan struct{})
	policy := must(authorization.NewPolicy(func(ctx context.Context, _ planapplication.Revision) (authorization.RuleConclusion, error) {
		close(entered)
		<-ctx.Done()
		return authorization.Unknown, nil
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct {
		result planapplication.Result
		err    error
	}, 1)
	go func() {
		result, err := planapplication.RequestApplication(ctx, revision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
			return planapplication.AcknowledgedReceipt("not-called")
		})
		done <- struct {
			result planapplication.Result
			err    error
		}{result, err}
	}()
	<-entered
	cancel()
	outcome := <-done
	if diff := cmp.Diff(context.Canceled, outcome.err, cmpopts.EquateErrors()); diff != "" {
		t.Fatalf("cancellation error mismatch (-want +got):\n%s", diff)
	}
	if decision, present := outcome.result.AuthorizationDecision(); present {
		t.Errorf("cancelled evaluation exposed decision %v", decision)
	}
	if _, present := outcome.result.Receipt(); present {
		t.Error("cancelled evaluation exposed receipt")
	}
}

func TestRequestApplicationActorCancellationEvidenceLSP(t *testing.T) {
	current, proposed := plans()
	revision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	policy := must(authorization.NewPolicy(func(context.Context, planapplication.Revision) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	tests := []struct {
		name      string
		evidence  func() planapplication.ReceiptEvidence
		wantKind  planapplication.ReceiptKind
		wantRef   string
		wantRefOK bool
	}{
		{name: "pre-transmission known not received", evidence: func() planapplication.ReceiptEvidence { return planapplication.NotReceivedReceipt() }, wantKind: planapplication.KnownNotReceived},
		{name: "possible transmission uncertain", evidence: func() planapplication.ReceiptEvidence { return planapplication.UncertainReceipt() }, wantKind: planapplication.ReceiptUncertain},
		{name: "acknowledged wins", evidence: func() planapplication.ReceiptEvidence { return planapplication.AcknowledgedReceipt("ack-ref") }, wantKind: planapplication.ReceiptAcknowledged, wantRef: "ack-ref", wantRefOK: true},
		{name: "refused wins", evidence: func() planapplication.ReceiptEvidence { return planapplication.RefusedReceipt("ref-ref") }, wantKind: planapplication.ReceiptRefused, wantRef: "ref-ref", wantRefOK: true},
		{name: "known not received wins", evidence: func() planapplication.ReceiptEvidence { return planapplication.NotReceivedReceipt() }, wantKind: planapplication.KnownNotReceived},
		{name: "explicit uncertain wins", evidence: func() planapplication.ReceiptEvidence { return planapplication.UncertainReceipt() }, wantKind: planapplication.ReceiptUncertain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			entered := make(chan struct{})
			done := make(chan struct {
				result planapplication.Result
				err    error
			}, 1)
			calls := 0
			go func() {
				result, err := planapplication.RequestApplication(ctx, revision, policy, func(actorContext context.Context, _ planapplication.Request) planapplication.ReceiptEvidence {
					calls++
					evidence := tt.evidence()
					close(entered)
					<-actorContext.Done()
					return evidence
				})
				done <- struct {
					result planapplication.Result
					err    error
				}{result, err}
			}()
			<-entered
			cancel()
			outcome := <-done
			result, err := outcome.result, outcome.err
			if diff := cmp.Diff(nil, err); diff != "" {
				t.Fatalf("Actor cancellation error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(1, calls); diff != "" {
				t.Fatalf("Actor call count mismatch (-want +got):\n%s", diff)
			}
			decision, hasDecision := result.AuthorizationDecision()
			receipt, hasReceipt := result.Receipt()
			if diff := cmp.Diff(false, hasDecision); diff != "" {
				t.Errorf("receipt result decision presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(authorization.Undecidable, decision); diff != "" {
				t.Errorf("receipt result hidden decision mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasReceipt); diff != "" {
				t.Fatalf("receipt result presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantKind, receipt.Kind()); diff != "" {
				t.Errorf("receipt kind mismatch (-want +got):\n%s", diff)
			}
			reference, referenceOK := receipt.Reference()
			if diff := cmp.Diff(tt.wantRefOK, referenceOK); diff != "" {
				t.Errorf("receipt reference presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRef, reference); diff != "" {
				t.Errorf("receipt reference mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestApplicationInvalidRevisionDoesNotCallNonNilActor(t *testing.T) {
	policy := authorization.Policy[planapplication.Revision]{}
	calls := 0
	_, err := planapplication.RequestApplication(context.Background(), planapplication.Revision{}, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		calls++
		return planapplication.AcknowledgedReceipt("unexpected")
	})
	var failure planapplication.Failure
	if diff := cmp.Diff(true, errors.As(err, &failure)); diff != "" {
		t.Fatalf("failure type mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(planapplication.InvalidRevision, failure.Code()); diff != "" {
		t.Errorf("failure code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, calls); diff != "" {
		t.Errorf("Actor calls mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestApplicationSameRevisionFreshEvidence(t *testing.T) {
	current, proposed := plans()
	revision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "id")), current, proposed))
	var evidence atomic.Int32
	policy := must(authorization.NewPolicy(func(_ context.Context, subject planapplication.Revision) (authorization.RuleConclusion, error) {
		if !subject.Equal(revision) {
			return authorization.Unknown, nil
		}
		switch evidence.Load() {
		case 0:
			return authorization.Permit, nil
		case 1:
			return authorization.Unknown, nil
		default:
			return authorization.Deny, nil
		}
	}))
	actorCalls := 0
	first, err := planapplication.RequestApplication(context.Background(), revision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		actorCalls++
		return planapplication.AcknowledgedReceipt("first")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("first evaluation error mismatch (-want +got):\n%s", diff)
	}
	evidence.Store(1)
	second, err := planapplication.RequestApplication(context.Background(), revision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		actorCalls++
		return planapplication.AcknowledgedReceipt("unexpected-unknown")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("second evaluation error mismatch (-want +got):\n%s", diff)
	}
	evidence.Store(2)
	third, err := planapplication.RequestApplication(context.Background(), revision, policy, func(context.Context, planapplication.Request) planapplication.ReceiptEvidence {
		actorCalls++
		return planapplication.AcknowledgedReceipt("unexpected-denied")
	})
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("third evaluation error mismatch (-want +got):\n%s", diff)
	}
	if receipt, ok := first.Receipt(); !ok {
		t.Fatal("first evaluation did not return receipt")
	} else if reference, ok := receipt.Reference(); !ok {
		t.Errorf("first receipt omitted reference")
	} else if diff := cmp.Diff("first", reference); diff != "" {
		t.Errorf("first receipt mismatch (-want +got):\n%s", diff)
	}
	if decision, ok := second.AuthorizationDecision(); !ok {
		t.Fatal("second evaluation did not return decision")
	} else if diff := cmp.Diff(authorization.Undecidable, decision); diff != "" {
		t.Errorf("second decision mismatch (-want +got):\n%s", diff)
	}
	if decision, ok := third.AuthorizationDecision(); !ok {
		t.Fatal("third evaluation did not return decision")
	} else if diff := cmp.Diff(authorization.Denied, decision); diff != "" {
		t.Errorf("third decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, actorCalls); diff != "" {
		t.Errorf("Actor invocation count mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestApplicationConcurrentLabeledIsolation(t *testing.T) {
	current, proposed := plans()
	cancelledRevision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "cancelled")), current, proposed))
	otherPlan := must(plan.New("Other", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	otherRevision := must(planapplication.NewRevision(must(planapplication.NewTargetReference("github", "other")), current, otherPlan))
	entered := make(chan string, 2)
	release := make(chan struct{})
	policy := must(authorization.NewPolicy(func(ctx context.Context, subject planapplication.Revision) (authorization.RuleConclusion, error) {
		name := subject.Target().Identity()
		entered <- name
		if name == "cancelled" {
			<-ctx.Done()
			return authorization.Unknown, nil
		}
		<-release
		return authorization.Permit, nil
	}))
	cancelledContext, cancel := context.WithCancel(context.Background())
	type outcome struct {
		label   string
		request planapplication.Request
		result  planapplication.Result
		err     error
	}
	outcomes := make(chan outcome, 4)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		result, err := planapplication.RequestApplication(cancelledContext, cancelledRevision, policy, func(_ context.Context, request planapplication.Request) planapplication.ReceiptEvidence {
			outcomes <- outcome{label: "cancelled-actor", request: request}
			return planapplication.AcknowledgedReceipt("wrong")
		})
		outcomes <- outcome{label: "cancelled", result: result, err: err}
	}()
	go func() {
		defer wait.Done()
		result, err := planapplication.RequestApplication(context.Background(), otherRevision, policy, func(_ context.Context, request planapplication.Request) planapplication.ReceiptEvidence {
			outcomes <- outcome{label: "other-actor", request: request}
			return planapplication.AcknowledgedReceipt("other")
		})
		outcomes <- outcome{label: "other", result: result, err: err}
	}()
	<-entered
	<-entered
	cancel()
	close(release)
	wait.Wait()
	close(outcomes)
	var cancelledOutcome, otherOutcome outcome
	for item := range outcomes {
		switch item.label {
		case "cancelled":
			cancelledOutcome = item
		case "other":
			otherOutcome = item
		case "cancelled-actor", "other-actor":
			if item.label == "cancelled-actor" {
				t.Errorf("cancelled call invoked Actor")
			} else if diff := cmp.Diff(otherRevision.Equal(item.request.Revision()), true); diff != "" {
				t.Errorf("other Actor request mismatch (-want +got):\n%s", diff)
			}
		}
	}
	if diff := cmp.Diff(context.Canceled, cancelledOutcome.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("cancelled call error mismatch (-want +got):\n%s", diff)
	}
	if _, present := cancelledOutcome.result.AuthorizationDecision(); present {
		t.Error("cancelled call exposed a decision")
	}
	if _, present := cancelledOutcome.result.Receipt(); present {
		t.Error("cancelled call exposed a receipt")
	}
	if diff := cmp.Diff(nil, otherOutcome.err); diff != "" {
		t.Errorf("unaffected call error mismatch (-want +got):\n%s", diff)
	}
	if receipt, present := otherOutcome.result.Receipt(); !present {
		t.Error("unaffected call omitted receipt")
	} else {
		if diff := cmp.Diff(planapplication.ReceiptAcknowledged, receipt.Kind()); diff != "" {
			t.Errorf("unaffected receipt kind mismatch (-want +got):\n%s", diff)
		}
		if reference, ok := receipt.Reference(); !ok {
			t.Error("unaffected receipt omitted reference")
		} else if diff := cmp.Diff("other", reference); diff != "" {
			t.Errorf("unaffected receipt reference mismatch (-want +got):\n%s", diff)
		}
	}
	if _, present := otherOutcome.result.AuthorizationDecision(); present {
		t.Error("unaffected receipt result exposed a decision")
	}
}
