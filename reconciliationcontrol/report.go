package reconciliationcontrol

// Report publishes exactly one Completion or one Failure for one target.
type Report[R any] struct {
	target    TargetIdentity
	result    R
	directive Directive
	failure   Failure
	completed bool
	failed    bool
}

func newCompletionReport[R any](target TargetIdentity, result R, directive Directive) Report[R] {
	return Report[R]{
		target:    target,
		result:    result,
		directive: directive,
		completed: true,
	}
}

func newFailureReport[R any](target TargetIdentity, failure Failure) Report[R] {
	return Report[R]{target: target, failure: failure, failed: true}
}

// Target returns the exact Target Identity associated with the outcome.
func (r Report[R]) Target() TargetIdentity { return r.target }

// Completion returns the successful target value and Control Directive. The
// presence flag is independent of whether R happens to have its zero value.
func (r Report[R]) Completion() (R, Directive, bool) {
	return r.result, r.directive, r.completed
}

// Failure returns the Attempt Failure when this is not a Completion.
func (r Report[R]) Failure() (Failure, bool) { return r.failure, r.failed }
