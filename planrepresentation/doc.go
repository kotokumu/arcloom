// Package planrepresentation reconciles an expected provider-independent Plan
// with one immutable provider-independent observation of an external Plan
// representation.
//
// The package owns observation validity, semantic correspondence, evidence,
// and the consumer-owned Observer boundary. It does not own external target
// identity, Provider state, mutation, authorization, or persistence.
package planrepresentation
