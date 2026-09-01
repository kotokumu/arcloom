package planrepresentation

import "github.com/kotokumu/arcloom/controllers/plan"

// ObservationValidationError identifies an impossible provider-independent
// observation or location state. Callers can inspect it with errors.As; it
// never wraps a Provider error.
type ObservationValidationError struct {
	reason string
}

func (e *ObservationValidationError) Error() string {
	if e == nil {
		return "observation validation error"
	}
	return "observation validation failed: " + e.reason
}

// LocationKind identifies one provider-independent Plan location.
type LocationKind uint8

const (
	PlanRootLocation LocationKind = iota + 1
	PlanNameLocation
	GoalLocation
	AcceptanceConditionCollectionLocation
	AcceptanceConditionMemberLocation
	TaskCollectionLocation
	TaskMemberLocation
	TargetDateLocation
)

// Location identifies one provider-independent Plan position.
type Location struct {
	kind   LocationKind
	member string
}

// Kind returns the location kind. The zero Location has kind zero.
func (l Location) Kind() LocationKind { return l.kind }

// Member returns the exact member text for a member location.
func (l Location) Member() (string, bool) {
	switch l.kind {
	case AcceptanceConditionMemberLocation, TaskMemberLocation:
		return l.member, true
	default:
		return "", false
	}
}

// PlanRoot returns the covering Plan-root location.
func PlanRoot() Location { return Location{kind: PlanRootLocation} }

// PlanName returns the Plan-name location.
func PlanName() Location { return Location{kind: PlanNameLocation} }

// Goal returns the Goal location.
func Goal() Location { return Location{kind: GoalLocation} }

// AcceptanceConditions returns the acceptance-condition collection location.
func AcceptanceConditions() Location {
	return Location{kind: AcceptanceConditionCollectionLocation}
}

// AcceptanceCondition returns a validated acceptance-condition member
// location preserving the exact statement. Invalid text returns the
// Plan-owned *plan.ValidationError and a zero Location.
func AcceptanceCondition(statement string) (Location, error) {
	value, err := plan.NewAcceptanceCondition(statement)
	if err != nil {
		return Location{}, err
	}
	return acceptanceConditionLocation(value.Statement()), nil
}

func acceptanceConditionLocation(statement string) Location {
	return Location{kind: AcceptanceConditionMemberLocation, member: statement}
}

// Tasks returns the Task collection location.
func Tasks() Location { return Location{kind: TaskCollectionLocation} }

// Task returns a validated Task member location preserving the exact name.
// Invalid text returns the Plan-owned *plan.ValidationError and a zero
// Location.
func Task(name string) (Location, error) {
	value, err := plan.NewTask(name)
	if err != nil {
		return Location{}, err
	}
	return taskLocation(value.Name()), nil
}

func taskLocation(name string) Location {
	return Location{kind: TaskMemberLocation, member: name}
}

// TargetDate returns the target-date location.
func TargetDate() Location { return Location{kind: TargetDateLocation} }

// LeastUnavailableLocation returns the least Plan location covering all
// affected locations. A member affects its collection; different top-level
// branches affect the Plan root. Empty input or an invalid Location returns a
// zero Location and an ObservationValidationError.
func LeastUnavailableLocation(affected ...Location) (Location, error) {
	if len(affected) == 0 {
		return Location{}, &ObservationValidationError{reason: "no affected location"}
	}

	covering := Location{}
	for _, location := range affected {
		current, ok := coveringLocation(location)
		if !ok {
			return Location{}, &ObservationValidationError{reason: "invalid location"}
		}
		if covering.kind == 0 {
			covering = current
			continue
		}
		if covering.kind != current.kind {
			return PlanRoot(), nil
		}
	}
	return covering, nil
}

func coveringLocation(location Location) (Location, bool) {
	switch location.kind {
	case PlanRootLocation, PlanNameLocation, GoalLocation, AcceptanceConditionCollectionLocation, TaskCollectionLocation, TargetDateLocation:
		return Location{kind: location.kind}, true
	case AcceptanceConditionMemberLocation:
		return AcceptanceConditions(), true
	case TaskMemberLocation:
		return Tasks(), true
	default:
		return Location{}, false
	}
}
