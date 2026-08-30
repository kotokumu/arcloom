// Package reconciliationcontrol runs target-independent, caller-scoped
// control lifecycles without owning target-specific reconciliation meaning.
package reconciliationcontrol

import (
	"errors"
	"strings"
)

var ErrInvalidTargetIdentity = errors.New("invalid target identity")

// TargetIdentity is a stable caller-established control key, not target state.
type TargetIdentity struct {
	kind string
	key  string
}

// NewTargetIdentity constructs a TargetIdentity.
func NewTargetIdentity(kind, key string) (TargetIdentity, error) {
	if !isValidIdentityPart(kind) || !isValidIdentityPart(key) {
		return TargetIdentity{}, ErrInvalidTargetIdentity
	}
	return TargetIdentity{kind: kind, key: key}, nil
}

// Kind returns the preserved target kind.
func (t TargetIdentity) Kind() string { return t.kind }

// Key returns the preserved target key.
func (t TargetIdentity) Key() string { return t.key }

func (t TargetIdentity) isValid() bool {
	return isValidIdentityPart(t.kind) && isValidIdentityPart(t.key)
}

func isValidIdentityPart(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}
