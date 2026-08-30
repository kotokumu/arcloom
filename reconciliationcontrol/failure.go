package reconciliationcontrol

import "fmt"

// FailureKind distinguishes target execution from control protocol failure.
type FailureKind uint8

const (
	TargetAttemptFailed FailureKind = iota + 1
	ControlDirectiveRejected
)

// Failure is one target-bound Attempt or Directive protocol failure.
type Failure struct {
	kind  FailureKind
	cause error
}

func newFailure(kind FailureKind, cause error) Failure {
	return Failure{kind: kind, cause: cause}
}

// Kind returns the semantic control failure kind.
func (f Failure) Kind() FailureKind { return f.kind }

// Error describes the failure.
func (f Failure) Error() string {
	switch f.kind {
	case TargetAttemptFailed:
		return fmt.Sprintf("target attempt failed: %v", f.cause)
	case ControlDirectiveRejected:
		return fmt.Sprintf("control directive rejected: %v", f.cause)
	default:
		return "invalid attempt failure"
	}
}

// Unwrap returns the preserved target or Directive validation cause.
func (f Failure) Unwrap() error { return f.cause }
