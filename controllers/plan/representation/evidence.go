package planrepresentation

import (
	"sort"

	"github.com/kotokumu/arcloom/controllers/plan"
)

// DifferenceCategory identifies one known semantic difference.
type DifferenceCategory uint8

const (
	ExpectedAbsentCategory DifferenceCategory = iota + 1
	UnexpectedPresentCategory
	ValueDifferentCategory
	InvalidObservedCategory
)

// Meaning is a closed provider-independent meaning value.
type Meaning interface{ isMeaning() }

// PlanMeaning carries an immutable expected Plan meaning.
type PlanMeaning interface {
	Meaning
	isPlanMeaning()
	Plan() plan.Plan
}

// TextMeaning carries exact preserved text meaning.
type TextMeaning interface {
	Meaning
	isTextMeaning()
	Text() string
}

// TargetDateMeaning carries a provider-independent target-date meaning.
type TargetDateMeaning interface {
	Meaning
	isTargetDateMeaning()
	TargetDate() plan.TargetDate
}

type planMeaning struct{ value plan.Plan }

func (planMeaning) isMeaning()        {}
func (planMeaning) isPlanMeaning()    {}
func (m planMeaning) Plan() plan.Plan { return m.value }

type textMeaning struct{ value string }

func (textMeaning) isMeaning()     {}
func (textMeaning) isTextMeaning() {}
func (m textMeaning) Text() string { return m.value }

type targetDateMeaning struct{ value plan.TargetDate }

func (targetDateMeaning) isMeaning()                    {}
func (targetDateMeaning) isTargetDateMeaning()          {}
func (m targetDateMeaning) TargetDate() plan.TargetDate { return m.value }

// Difference is a closed provider-independent known difference.
type Difference interface {
	Category() DifferenceCategory
	Location() Location
	isDifference()
}

// ExpectedAbsentDifference identifies expected meaning absent from a known
// observation.
type ExpectedAbsentDifference interface {
	Difference
	isExpectedAbsentDifference()
	Expected() Meaning
}

// UnexpectedPresentDifference identifies known observed meaning not expected.
type UnexpectedPresentDifference interface {
	Difference
	isUnexpectedPresentDifference()
	Observed() Meaning
}

// ValueDifferentDifference identifies unequal expected and observed meaning.
type ValueDifferentDifference interface {
	Difference
	isValueDifferentDifference()
	Expected() Meaning
	Observed() Meaning
}

// InvalidObservedDifference identifies a stable Plan-invariant violation.
type InvalidObservedDifference interface {
	Difference
	isInvalidObservedDifference()
	Violation() plan.ViolationCode
}

type expectedAbsentDifference struct {
	location Location
	expected Meaning
}

func (expectedAbsentDifference) isDifference()                  {}
func (expectedAbsentDifference) isExpectedAbsentDifference()    {}
func (d expectedAbsentDifference) Category() DifferenceCategory { return ExpectedAbsentCategory }
func (d expectedAbsentDifference) Location() Location           { return d.location }
func (d expectedAbsentDifference) Expected() Meaning            { return d.expected }

type unexpectedPresentDifference struct {
	location Location
	observed Meaning
}

func (unexpectedPresentDifference) isDifference()                  {}
func (unexpectedPresentDifference) isUnexpectedPresentDifference() {}
func (d unexpectedPresentDifference) Category() DifferenceCategory { return UnexpectedPresentCategory }
func (d unexpectedPresentDifference) Location() Location           { return d.location }
func (d unexpectedPresentDifference) Observed() Meaning            { return d.observed }

type valueDifferentDifference struct {
	location Location
	expected Meaning
	observed Meaning
}

func (valueDifferentDifference) isDifference()                  {}
func (valueDifferentDifference) isValueDifferentDifference()    {}
func (d valueDifferentDifference) Category() DifferenceCategory { return ValueDifferentCategory }
func (d valueDifferentDifference) Location() Location           { return d.location }
func (d valueDifferentDifference) Expected() Meaning            { return d.expected }
func (d valueDifferentDifference) Observed() Meaning            { return d.observed }

type invalidObservedDifference struct {
	location  Location
	violation plan.ViolationCode
}

func (invalidObservedDifference) isDifference()                   {}
func (invalidObservedDifference) isInvalidObservedDifference()    {}
func (d invalidObservedDifference) Category() DifferenceCategory  { return InvalidObservedCategory }
func (d invalidObservedDifference) Location() Location            { return d.location }
func (d invalidObservedDifference) Violation() plan.ViolationCode { return d.violation }

// UnavailableInformation identifies one Plan location whose facts could not
// be established.
type UnavailableInformation interface {
	Location() Location
	isUnavailableInformation()
}

type unavailableInformation struct{ location Location }

func (unavailableInformation) isUnavailableInformation() {}
func (u unavailableInformation) Location() Location      { return u.location }

// Determination is the evidence-derived state of a Plan Representation Result.
type Determination uint8

const (
	Satisfied Determination = iota + 1
	NotSatisfied
	Undecidable
)

// Result is an immutable Plan-specific reconciliation result. Its evidence is
// canonically ordered, unique, aggregated by Plan violation and location, and
// defensively snapshotted. The zero Result is invalid: Determination returns
// the zero determination and both evidence accessors return nil. Results
// returned by Controller are valid; a valid empty result is Satisfied.
type Result struct {
	differences []Difference
	unavailable []UnavailableInformation
	valid       bool
}

func newResult(differences []Difference, unavailable []UnavailableInformation) Result {
	differences, unavailable = canonicalEvidence(differences, unavailable)
	return Result{
		differences: append([]Difference(nil), differences...),
		unavailable: append([]UnavailableInformation(nil), unavailable...),
		valid:       true,
	}
}

// canonicalEvidence is the Result boundary for Plan-specific evidence. The
// correspondence functions only derive candidates; this boundary owns
// covering, aggregation, uniqueness, and canonical ordering.
func canonicalEvidence(differences []Difference, unavailable []UnavailableInformation) ([]Difference, []UnavailableInformation) {
	differences = append([]Difference(nil), differences...)
	unavailable = append([]UnavailableInformation(nil), unavailable...)

	for _, item := range unavailable {
		if item != nil && item.Location().Kind() == PlanRootLocation {
			return nil, []UnavailableInformation{unavailableInformation{location: PlanRoot()}}
		}
	}
	for _, item := range differences {
		if item != nil && item.Location().Kind() == PlanRootLocation {
			return []Difference{item}, nil
		}
	}

	differences = uniqueDifferences(differences)
	unavailable = uniqueUnavailable(unavailable)
	sortDifferences(differences)
	sortUnavailable(unavailable)
	return differences, unavailable
}

type differenceKey struct {
	category     DifferenceCategory
	location     Location
	violation    plan.ViolationCode
	expectedKind uint8
	expectedText string
	observedKind uint8
	observedText string
}

func uniqueDifferences(values []Difference) []Difference {
	seen := make(map[differenceKey]struct{}, len(values))
	result := make([]Difference, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		key := differenceKey{category: value.Category(), location: value.Location()}
		switch difference := value.(type) {
		case InvalidObservedDifference:
			key.violation = difference.Violation()
		case ExpectedAbsentDifference:
			key.expectedKind, key.expectedText = meaningKey(difference.Expected())
		case UnexpectedPresentDifference:
			key.observedKind, key.observedText = meaningKey(difference.Observed())
		case ValueDifferentDifference:
			key.expectedKind, key.expectedText = meaningKey(difference.Expected())
			key.observedKind, key.observedText = meaningKey(difference.Observed())
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func meaningKey(value Meaning) (uint8, string) {
	switch meaning := value.(type) {
	case TextMeaning:
		return 1, meaning.Text()
	case TargetDateMeaning:
		return 2, meaning.TargetDate().String()
	case PlanMeaning:
		return 3, meaning.Plan().Name()
	default:
		return 0, ""
	}
}

func uniqueUnavailable(values []UnavailableInformation) []UnavailableInformation {
	seen := make(map[Location]struct{}, len(values))
	result := make([]UnavailableInformation, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		location := value.Location()
		if _, exists := seen[location]; exists {
			continue
		}
		seen[location] = struct{}{}
		result = append(result, value)
	}
	return result
}

// Determination returns the state derived from the complete evidence snapshot.
func (r Result) Determination() Determination {
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
func (r Result) Differences() []Difference {
	if !r.valid {
		return nil
	}
	return append([]Difference{}, r.differences...)
}

// UnavailableInformation returns an independent snapshot of unavailable
// evidence.
func (r Result) UnavailableInformation() []UnavailableInformation {
	if !r.valid {
		return nil
	}
	return append([]UnavailableInformation{}, r.unavailable...)
}

func reconcileObservation(expected plan.Plan, observation Observation) Result {
	switch observation.root {
	case rootAbsent:
		return newResult([]Difference{expectedAbsentDifference{location: PlanRoot(), expected: planMeaning{value: expected}}}, nil)
	case rootUnavailable:
		return newResult(nil, []UnavailableInformation{unavailableInformation{location: PlanRoot()}})
	}

	differences := make([]Difference, 0)
	unavailable := make([]UnavailableInformation, 0)
	if difference, unavailableInformation := compareScalarObservation(PlanName(), expected.Name(), scalarObservation{
		state:     observation.name.state,
		value:     observation.name.value,
		violation: observation.name.violation,
	}); difference != nil {
		differences = append(differences, difference)
	} else if unavailableInformation != nil {
		unavailable = append(unavailable, unavailableInformation)
	}
	if difference, unavailableInformation := compareScalarObservation(Goal(), expected.Goal().Text(), scalarObservation{
		state:     observation.goal.state,
		value:     observation.goal.value,
		violation: observation.goal.violation,
	}); difference != nil {
		differences = append(differences, difference)
	} else if unavailableInformation != nil {
		unavailable = append(unavailable, unavailableInformation)
	}
	conditionDifferences, conditionUnavailable := compareAcceptanceConditions(expected.AcceptanceConditions(), observation.conditions)
	differences = append(differences, conditionDifferences...)
	unavailable = append(unavailable, conditionUnavailable...)
	taskDifferences, taskUnavailable := compareTasks(expected.Tasks(), observation.tasks)
	differences = append(differences, taskDifferences...)
	unavailable = append(unavailable, taskUnavailable...)
	if difference, unavailableInformation := compareTargetDate(expected, observation.targetDate); difference != nil {
		differences = append(differences, difference)
	} else if unavailableInformation != nil {
		unavailable = append(unavailable, unavailableInformation)
	}

	return newResult(differences, unavailable)
}

type scalarObservation struct {
	state     scalarObservationState
	value     string
	violation plan.ViolationCode
}

func compareScalarObservation(
	location Location,
	expected string,
	observed scalarObservation,
) (Difference, UnavailableInformation) {
	switch observed.state {
	case scalarValid:
		if observed.value != expected {
			return valueDifferentDifference{
				location: location,
				expected: textMeaning{value: expected},
				observed: textMeaning{value: observed.value},
			}, nil
		}
	case scalarInvalid:
		return invalidObservedDifference{location: location, violation: observed.violation}, nil
	case scalarUnavailable:
		return nil, unavailableInformation{location: location}
	}
	return nil, nil
}

func compareAcceptanceConditions(
	expected []plan.AcceptanceCondition,
	observed AcceptanceConditionsObservation,
) ([]Difference, []UnavailableInformation) {
	return compareCollection(
		AcceptanceConditions(),
		collectionObservation(observed),
		acceptanceMembers(expected),
		func(member string) (Location, Meaning) {
			return acceptanceConditionLocation(member), textMeaning{value: member}
		},
	)
}

func compareTasks(
	expected []plan.Task,
	observed TasksObservation,
) ([]Difference, []UnavailableInformation) {
	return compareCollection(
		Tasks(),
		collectionObservation(observed),
		taskMembers(expected),
		func(member string) (Location, Meaning) {
			return taskLocation(member), textMeaning{value: member}
		},
	)
}

type collectionObservation struct {
	state      collectionObservationState
	members    []string
	violations []plan.ViolationCode
}

func compareCollection(
	collection Location,
	observed collectionObservation,
	expectedMembers []string,
	memberMeaning func(string) (Location, Meaning),
) ([]Difference, []UnavailableInformation) {
	differences := make([]Difference, 0, len(observed.violations)+len(expectedMembers))
	unavailable := make([]UnavailableInformation, 0, 1)
	for _, violation := range observed.violations {
		differences = append(differences, invalidObservedDifference{location: collection, violation: violation})
	}
	expectedSet := make(map[string]struct{}, len(expectedMembers))
	for _, member := range expectedMembers {
		expectedSet[member] = struct{}{}
	}
	observedSet := make(map[string]struct{}, len(observed.members))
	for _, member := range observed.members {
		observedSet[member] = struct{}{}
		if _, exists := expectedSet[member]; !exists {
			location, meaning := memberMeaning(member)
			differences = append(differences, unexpectedPresentDifference{location: location, observed: meaning})
		}
	}
	if observed.state == collectionComplete {
		for _, member := range expectedMembers {
			if _, exists := observedSet[member]; !exists {
				location, meaning := memberMeaning(member)
				differences = append(differences, expectedAbsentDifference{location: location, expected: meaning})
			}
		}
	} else {
		unavailable = append(unavailable, unavailableInformation{location: collection})
	}
	return differences, unavailable
}

func compareTargetDate(
	expected plan.Plan,
	observed TargetDateObservation,
) (Difference, UnavailableInformation) {
	expectedDate, expectedHasDate := expected.TargetDate()
	switch observed.state {
	case targetDatePresent:
		if !expectedHasDate {
			return unexpectedPresentDifference{location: TargetDate(), observed: targetDateMeaning{value: observed.value}}, nil
		}
		if observed.value.String() != expectedDate.String() {
			return valueDifferentDifference{location: TargetDate(), expected: targetDateMeaning{value: expectedDate}, observed: targetDateMeaning{value: observed.value}}, nil
		}
	case targetDateAbsent:
		if expectedHasDate {
			return expectedAbsentDifference{location: TargetDate(), expected: targetDateMeaning{value: expectedDate}}, nil
		}
	case targetDateInvalid:
		return invalidObservedDifference{location: TargetDate(), violation: observed.violation}, nil
	case targetDateUnavailable:
		return nil, unavailableInformation{location: TargetDate()}
	}
	return nil, nil
}

func acceptanceMembers(values []plan.AcceptanceCondition) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Statement())
	}
	return result
}

func taskMembers(values []plan.Task) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Name())
	}
	return result
}

func sortDifferences(values []Difference) {
	sort.SliceStable(values, func(i, j int) bool {
		left := values[i]
		right := values[j]
		leftLocation := left.Location()
		rightLocation := right.Location()
		leftRank := locationRank(leftLocation.Kind())
		rightRank := locationRank(rightLocation.Kind())
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftMember, _ := leftLocation.Member()
		rightMember, _ := rightLocation.Member()
		if leftMember != rightMember {
			return leftMember < rightMember
		}
		if left.Category() != right.Category() {
			return left.Category() < right.Category()
		}
		leftInvalid, leftIsInvalid := left.(InvalidObservedDifference)
		rightInvalid, rightIsInvalid := right.(InvalidObservedDifference)
		if leftIsInvalid && rightIsInvalid {
			return string(leftInvalid.Violation()) < string(rightInvalid.Violation())
		}
		leftExpected, leftHasExpected := differenceExpectedMeaning(left)
		rightExpected, rightHasExpected := differenceExpectedMeaning(right)
		if leftHasExpected && rightHasExpected && leftExpected != rightExpected {
			return leftExpected < rightExpected
		}
		leftObserved, leftHasObserved := differenceObservedMeaning(left)
		rightObserved, rightHasObserved := differenceObservedMeaning(right)
		return leftHasObserved && rightHasObserved && leftObserved < rightObserved
	})
}

func differenceExpectedMeaning(value Difference) (string, bool) {
	switch difference := value.(type) {
	case ExpectedAbsentDifference:
		_, text := meaningKey(difference.Expected())
		return text, true
	case ValueDifferentDifference:
		_, text := meaningKey(difference.Expected())
		return text, true
	default:
		return "", false
	}
}

func differenceObservedMeaning(value Difference) (string, bool) {
	switch difference := value.(type) {
	case UnexpectedPresentDifference:
		_, text := meaningKey(difference.Observed())
		return text, true
	case ValueDifferentDifference:
		_, text := meaningKey(difference.Observed())
		return text, true
	default:
		return "", false
	}
}

func sortUnavailable(values []UnavailableInformation) {
	sort.SliceStable(values, func(i, j int) bool {
		return locationRank(values[i].Location().Kind()) < locationRank(values[j].Location().Kind())
	})
}

func locationRank(kind LocationKind) int {
	switch kind {
	case PlanRootLocation:
		return 0
	case PlanNameLocation:
		return 1
	case GoalLocation:
		return 2
	case AcceptanceConditionCollectionLocation:
		return 3
	case AcceptanceConditionMemberLocation:
		return 4
	case TaskCollectionLocation:
		return 5
	case TaskMemberLocation:
		return 6
	case TargetDateLocation:
		return 7
	default:
		return 99
	}
}
