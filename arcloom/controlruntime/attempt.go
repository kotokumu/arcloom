package controlruntime

import "context"

// Attempt evaluates one Target Identity from fresh target-specific facts. It
// may invoke zero or one semantic Reconciliation. One value may be called
// concurrently for distinct identities, but a Controller never calls it
// concurrently for equal identities. Implementations must return within their
// documented bound after ctx ends. A non-nil error always wins and makes the
// returned R and Directive irrelevant; a nil error requires a Directive created
// by this package. The Controller treats R as an opaque successful value.
type Attempt[R any] func(context.Context, TargetIdentity) (R, Directive, error)
