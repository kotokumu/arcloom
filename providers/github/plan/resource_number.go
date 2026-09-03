package githubplan

// ResourceNumber is the positive GitHub number used to bind a Milestone or
// Issue representation. It is distinct from a REST resource id.
type ResourceNumber struct{ value int64 }

// NewResourceNumber declares the Host-facing resource-number contract.
// Validation behavior is implemented in a later task.
func NewResourceNumber(value int64) (ResourceNumber, error) {
	if value <= 0 {
		return ResourceNumber{}, &ValidationError{code: InvalidResourceNumber, field: ResourceNumberField}
	}
	return ResourceNumber{value: value}, nil
}

// Int64 returns the underlying resource number.
func (n ResourceNumber) Int64() int64 { return n.value }
