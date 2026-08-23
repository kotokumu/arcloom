// Package reconciliation defines target-independent reconciliation results.
package reconciliation

// Determination is the evidence-derived state of a reconciliation result.
type Determination uint8

const (
	Satisfied Determination = iota + 1
	NotSatisfied
	Undecidable
)

// Result is an immutable slice-only snapshot of target-specific differences
// and unavailable information. NewResult defensively copies the input slices
// and each accessor returns another slice copy; it does not deep-copy D or U
// elements. Callers must supply immutable, valid D and U values and must not
// mutate their reachable state after construction. A zero Result is invalid:
// Determination returns zero and both accessors return nil. NewResult(nil,
// nil) is a distinct valid Satisfied Result with empty evidence.
type Result[D any, U any] struct {
	differences []D
	unavailable []U
	valid       bool
}

// NewResult snapshots the supplied evidence slices and derives the result
// determination from their contents. The slices are copied, but D and U
// elements are not deep-copied and must already be immutable valid values.
func NewResult[D any, U any](differences []D, unavailable []U) Result[D, U] {
	differenceCopy := make([]D, len(differences))
	copy(differenceCopy, differences)
	unavailableCopy := make([]U, len(unavailable))
	copy(unavailableCopy, unavailable)
	return Result[D, U]{
		differences: differenceCopy,
		unavailable: unavailableCopy,
		valid:       true,
	}
}

// Determination derives the three-state result from its complete evidence
// snapshot: no evidence is Satisfied, differences without unavailable
// information are NotSatisfied, and any unavailable information is
// Undecidable. The zero Result returns the zero Determination.
func (r Result[D, U]) Determination() Determination {
	if !r.valid {
		return 0
	}
	if len(r.unavailable) > 0 {
		return Undecidable
	}
	if len(r.differences) > 0 {
		return NotSatisfied
	}
	return Satisfied
}

// Differences returns an independent snapshot of known differences.
func (r Result[D, U]) Differences() []D {
	if !r.valid {
		return nil
	}
	result := make([]D, len(r.differences))
	copy(result, r.differences)
	return result
}

// UnavailableInformation returns an independent snapshot of unavailable
// information.
func (r Result[D, U]) UnavailableInformation() []U {
	if !r.valid {
		return nil
	}
	result := make([]U, len(r.unavailable))
	copy(result, r.unavailable)
	return result
}
