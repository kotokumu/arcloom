package authorization_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kotokumu/arcloom/arcloom/authorization"
)

type immutableSubject struct {
	state []string
}

func (s immutableSubject) State() []string {
	return append([]string(nil), s.state...)
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func TestPolicyAggregation(t *testing.T) {
	tests := []struct {
		name  string
		rules []authorization.Rule[string]
		want  authorization.Decision
	}{
		{
			name: "all permit",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Permit, nil },
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Permit, nil },
			},
			want: authorization.Authorized,
		},
		{
			name: "deny overrides permit",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Permit, nil },
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Deny, nil },
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Unknown, nil },
			},
			want: authorization.Denied,
		},
		{
			name: "deny before unknown",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Deny, nil },
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Unknown, nil },
			},
			want: authorization.Denied,
		},
		{
			name: "unknown is undecidable",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Permit, nil },
				func(context.Context, string) (authorization.RuleConclusion, error) { return authorization.Unknown, nil },
			},
			want: authorization.Undecidable,
		},
		{
			name: "invalid conclusion is undecidable",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.RuleConclusion(99), nil
				},
			},
			want: authorization.Undecidable,
		},
		{
			name: "rule failure is undecidable",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Permit, errors.New("evidence unavailable")
				},
			},
			want: authorization.Undecidable,
		},
		{
			name: "failed rule before successful deny",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Permit, errors.New("evidence unavailable")
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
			},
			want: authorization.Denied,
		},
		{
			name: "invalid conclusion before successful deny",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.RuleConclusion(99), nil
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
			},
			want: authorization.Denied,
		},
		{
			name: "unknown before successful deny",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Unknown, nil
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
			},
			want: authorization.Denied,
		},
		{
			name: "deny with error is undecidable",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, errors.New("evidence unavailable")
				},
			},
			want: authorization.Undecidable,
		},
		{
			name: "successful deny overrides failed rule",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Permit, errors.New("evidence unavailable")
				},
			},
			want: authorization.Denied,
		},
		{
			name: "successful deny before invalid conclusion",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.RuleConclusion(99), nil
				},
			},
			want: authorization.Denied,
		},
		{
			name: "failed deny before successful deny",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, errors.New("evidence unavailable")
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
			},
			want: authorization.Denied,
		},
		{
			name: "successful deny before failed deny",
			rules: []authorization.Rule[string]{
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, nil
				},
				func(context.Context, string) (authorization.RuleConclusion, error) {
					return authorization.Deny, errors.New("evidence unavailable")
				},
			},
			want: authorization.Denied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := must(authorization.NewPolicy(tt.rules...))
			evaluation, err := policy.Evaluate(context.Background(), "subject")
			if diff := cmp.Diff(error(nil), err); diff != "" {
				t.Fatalf("Evaluate() error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, evaluation.Decision()); diff != "" {
				t.Errorf("decision mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("subject", evaluation.Subject()); diff != "" {
				t.Errorf("subject mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPolicyCopiesRuleSlice(t *testing.T) {
	rules := []authorization.Rule[int]{
		func(context.Context, int) (authorization.RuleConclusion, error) {
			return authorization.Permit, nil
		},
	}
	policy := must(authorization.NewPolicy(rules...))
	rules[0] = func(context.Context, int) (authorization.RuleConclusion, error) {
		return authorization.Deny, nil
	}
	evaluation := must(policy.Evaluate(context.Background(), 1))
	if diff := cmp.Diff(authorization.Authorized, evaluation.Decision()); diff != "" {
		t.Errorf("copied rule-set decision mismatch (-want +got):\n%s", diff)
	}
}

func TestInvalidPoliciesFailClosed(t *testing.T) {
	var nilRule authorization.Rule[int]
	_, emptyErr := authorization.NewPolicy[int]()
	_, nilErr := authorization.NewPolicy(nilRule)
	if diff := cmp.Diff(true, emptyErr != nil); diff != "" {
		t.Errorf("empty policy construction mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, nilErr != nil); diff != "" {
		t.Errorf("nil rule construction mismatch (-want +got):\n%s", diff)
	}
	var zero authorization.Policy[int]
	evaluation, err := zero.Evaluate(context.Background(), 4)
	if diff := cmp.Diff(error(nil), err); diff != "" {
		t.Fatalf("zero policy evaluation error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, evaluation.Decision()); diff != "" {
		t.Errorf("zero policy decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(4, evaluation.Subject()); diff != "" {
		t.Errorf("zero policy subject mismatch (-want +got):\n%s", diff)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, err := zero.Evaluate(ctx, 4)
	if diff := cmp.Diff(context.Canceled, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("zero policy cancellation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, cancelled.Subject()); diff != "" {
		t.Errorf("zero policy cancelled subject mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, cancelled.Decision()); diff != "" {
		t.Errorf("zero policy cancelled decision mismatch (-want +got):\n%s", diff)
	}
}

func TestExactSubjectAndSemanticImmutability(t *testing.T) {
	source := []string{"permit"}
	input := immutableSubject{state: append([]string(nil), source...)}
	var observed immutableSubject
	policy := must(authorization.NewPolicy(func(_ context.Context, got immutableSubject) (authorization.RuleConclusion, error) {
		observed = got
		return authorization.Permit, nil
	}))
	evaluation := must(policy.Evaluate(context.Background(), input))
	if diff := cmp.Diff(input.State(), observed.State()); diff != "" {
		t.Errorf("rule subject mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(input.State(), evaluation.Subject().State()); diff != "" {
		t.Errorf("evaluation subject mismatch (-want +got):\n%s", diff)
	}
	source[0] = "mutated outside the immutable subject"
	returnedState := evaluation.Subject().State()
	returnedState[0] = "mutated through accessor result"
	if diff := cmp.Diff([]string{"permit"}, evaluation.Subject().State()); diff != "" {
		t.Errorf("consumer subject immutability mismatch (-want +got):\n%s", diff)
	}
}

func TestCancellationReturnsNoEvaluation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	policy := must(authorization.NewPolicy(func(context.Context, int) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	evaluation, err := policy.Evaluate(ctx, 5)
	if diff := cmp.Diff(context.Canceled, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("cancellation error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, evaluation.Subject()); diff != "" {
		t.Errorf("cancelled evaluation subject mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, evaluation.Decision()); diff != "" {
		t.Errorf("cancelled evaluation decision mismatch (-want +got):\n%s", diff)
	}
}

func TestInFlightCancellationDoesNotAffectConcurrentEvaluation(t *testing.T) {
	started := make(chan struct{})
	policy := must(authorization.NewPolicy(func(ctx context.Context, value int) (authorization.RuleConclusion, error) {
		if value == 1 {
			close(started)
			<-ctx.Done()
			return authorization.Permit, nil
		}
		return authorization.Permit, nil
	}))
	cancelContext, cancel := context.WithCancel(context.Background())
	results := make(chan struct {
		evaluation authorization.Evaluation[int]
		err        error
	}, 2)
	go func() {
		evaluation, err := policy.Evaluate(cancelContext, 1)
		results <- struct {
			evaluation authorization.Evaluation[int]
			err        error
		}{evaluation: evaluation, err: err}
	}()
	<-started
	go func() {
		evaluation, err := policy.Evaluate(context.Background(), 2)
		results <- struct {
			evaluation authorization.Evaluation[int]
			err        error
		}{evaluation: evaluation, err: err}
	}()
	cancel()
	first := <-results
	second := <-results
	resultsBySubject := map[int]struct {
		evaluation authorization.Evaluation[int]
		err        error
	}{
		first.evaluation.Subject():  first,
		second.evaluation.Subject(): second,
	}
	cancelled := resultsBySubject[0]
	success := resultsBySubject[2]
	if diff := cmp.Diff(context.Canceled, cancelled.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("cancelled error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, cancelled.evaluation.Subject()); diff != "" {
		t.Errorf("cancelled zero Evaluation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, cancelled.evaluation.Decision()); diff != "" {
		t.Errorf("cancelled zero Evaluation decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(error(nil), success.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("unaffected error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, success.evaluation.Subject()); diff != "" {
		t.Errorf("unaffected subject mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Authorized, success.evaluation.Decision()); diff != "" {
		t.Errorf("unaffected decision mismatch (-want +got):\n%s", diff)
	}
}

func TestCancellationAfterPermitBeforeLaterRuleReturnsNoEvaluation(t *testing.T) {
	firstRuleDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	policy := must(authorization.NewPolicy(
		func(context.Context, int) (authorization.RuleConclusion, error) {
			close(firstRuleDone)
			return authorization.Permit, nil
		},
		func(ctx context.Context, _ int) (authorization.RuleConclusion, error) {
			<-ctx.Done()
			return authorization.Permit, nil
		},
	))
	result := make(chan struct {
		evaluation authorization.Evaluation[int]
		err        error
	}, 1)
	go func() {
		evaluation, err := policy.Evaluate(ctx, 7)
		result <- struct {
			evaluation authorization.Evaluation[int]
			err        error
		}{evaluation: evaluation, err: err}
	}()
	<-firstRuleDone
	cancel()
	got := <-result
	if diff := cmp.Diff(context.Canceled, got.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("post-permit cancellation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, got.evaluation.Subject()); diff != "" {
		t.Errorf("post-permit cancelled subject mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Undecidable, got.evaluation.Decision()); diff != "" {
		t.Errorf("post-permit cancelled decision mismatch (-want +got):\n%s", diff)
	}
}

func TestRepeatedAndConcurrentEvaluationsRemainIsolated(t *testing.T) {
	permit := must(authorization.NewPolicy(func(context.Context, int) (authorization.RuleConclusion, error) {
		return authorization.Permit, nil
	}))
	deny := must(authorization.NewPolicy(func(context.Context, int) (authorization.RuleConclusion, error) {
		return authorization.Deny, nil
	}))
	first := must(permit.Evaluate(context.Background(), 10))
	second := must(deny.Evaluate(context.Background(), 10))
	if diff := cmp.Diff(authorization.Authorized, first.Decision()); diff != "" {
		t.Errorf("first decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Denied, second.Decision()); diff != "" {
		t.Errorf("second decision mismatch (-want +got):\n%s", diff)
	}

	results := make(chan authorization.Evaluation[int], 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		results <- must(permit.Evaluate(context.Background(), 21))
	}()
	go func() {
		defer wait.Done()
		results <- must(permit.Evaluate(context.Background(), 22))
	}()
	wait.Wait()
	close(results)
	subjects := []int{}
	for evaluation := range results {
		subjects = append(subjects, evaluation.Subject())
	}
	if diff := cmp.Diff([]int{21, 22}, subjects, cmpopts.SortSlices(func(left, right int) bool { return left < right })); diff != "" {
		t.Errorf("concurrent subject isolation mismatch (-want +got):\n%s", diff)
	}
}

func TestSequentialEvidenceRecalculationAndConcurrentEvidenceIsolation(t *testing.T) {
	var evidence atomic.Bool
	policy := must(authorization.NewPolicy(func(context.Context, int) (authorization.RuleConclusion, error) {
		if evidence.Load() {
			return authorization.Permit, nil
		}
		return authorization.Unknown, nil
	}))
	first := must(policy.Evaluate(context.Background(), 1))
	evidence.Store(true)
	second := must(policy.Evaluate(context.Background(), 1))
	if diff := cmp.Diff(authorization.Undecidable, first.Decision()); diff != "" {
		t.Errorf("initial evidence decision mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(authorization.Authorized, second.Decision()); diff != "" {
		t.Errorf("recalculated evidence decision mismatch (-want +got):\n%s", diff)
	}

	var subjectEvidence sync.Map
	subjectEvidence.Store(2, true)
	subjectEvidence.Store(3, false)
	barrierEntered := make(chan struct{}, 2)
	barrierRelease := make(chan struct{})
	boundPolicy := must(authorization.NewPolicy(func(_ context.Context, subject int) (authorization.RuleConclusion, error) {
		barrierEntered <- struct{}{}
		<-barrierRelease
		value, _ := subjectEvidence.Load(subject)
		if permitted, _ := value.(bool); permitted {
			return authorization.Permit, nil
		}
		return authorization.Unknown, nil
	}))
	results := make(chan authorization.Evaluation[int], 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		results <- must(boundPolicy.Evaluate(context.Background(), 2))
	}()
	go func() {
		defer wait.Done()
		results <- must(boundPolicy.Evaluate(context.Background(), 3))
	}()
	<-barrierEntered
	<-barrierEntered
	close(barrierRelease)
	wait.Wait()
	close(results)
	decisions := map[int]authorization.Decision{}
	for evaluation := range results {
		decisions[evaluation.Subject()] = evaluation.Decision()
	}
	if diff := cmp.Diff(map[int]authorization.Decision{
		2: authorization.Authorized,
		3: authorization.Undecidable,
	}, decisions); diff != "" {
		t.Errorf("concurrent evidence isolation mismatch (-want +got):\n%s", diff)
	}
}
