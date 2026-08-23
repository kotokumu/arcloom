package planrepresentation

import (
	"errors"
	"sort"

	"github.com/kotokumu/arcloom/plan"
)

type rootObservationState uint8

const (
	rootPresent rootObservationState = iota + 1
	rootAbsent
	rootUnavailable
)

type scalarObservationState uint8

const (
	scalarValid scalarObservationState = iota + 1
	scalarInvalid
	scalarUnavailable
)

type targetDateObservationState uint8

const (
	targetDatePresent targetDateObservationState = iota + 1
	targetDateAbsent
	targetDateInvalid
	targetDateUnavailable
)

type collectionObservationState uint8

const (
	collectionComplete collectionObservationState = iota + 1
	collectionIncomplete
)

// PlanNameObservation is a provider-independent observation of Plan name.
// Its state is created only through the named constructors in this package.
type PlanNameObservation struct {
	state     scalarObservationState
	value     string
	violation plan.ViolationCode
}

// GoalObservation is a provider-independent observation of Goal.
type GoalObservation struct {
	state     scalarObservationState
	value     string
	violation plan.ViolationCode
}

// TargetDateObservation is a provider-independent observation of target date.
type TargetDateObservation struct {
	state     targetDateObservationState
	value     plan.TargetDate
	violation plan.ViolationCode
}

// AcceptanceConditionsObservation is a provider-independent observation of
// acceptance-condition membership and validity.
type AcceptanceConditionsObservation struct {
	state      collectionObservationState
	members    []string
	violations []plan.ViolationCode
}

// TasksObservation is a provider-independent observation of Task membership
// and validity.
type TasksObservation struct {
	state      collectionObservationState
	members    []string
	violations []plan.ViolationCode
}

// Observation is one immutable logical observation of a bound Plan target.
// Its zero value is invalid and is rejected by Controller boundary checks.
type Observation struct {
	root       rootObservationState
	name       PlanNameObservation
	goal       GoalObservation
	conditions AcceptanceConditionsObservation
	tasks      TasksObservation
	targetDate TargetDateObservation
}

// AbsentObservation represents authoritative absence of the external Plan.
func AbsentObservation() Observation {
	return Observation{root: rootAbsent}
}

// UnavailableObservation represents unavailable knowledge of external Plan
// existence.
func UnavailableObservation() Observation {
	return Observation{root: rootUnavailable}
}

// NewPresentObservation constructs a present Observation after validating all
// child state values. Absent and unavailable roots cannot carry descendants.
// A zero or contradictory child returns a zero Observation and an
// ObservationValidationError, which callers can inspect with errors.As.
func NewPresentObservation(
	name PlanNameObservation,
	goal GoalObservation,
	conditions AcceptanceConditionsObservation,
	tasks TasksObservation,
	targetDate TargetDateObservation,
) (Observation, error) {
	observation := Observation{
		root:       rootPresent,
		name:       name,
		goal:       goal,
		conditions: conditions,
		tasks:      tasks,
		targetDate: targetDate,
	}
	if !observation.valid() {
		return Observation{}, &ObservationValidationError{reason: "invalid present observation"}
	}
	return observation, nil
}

// ClassifyPlanName applies Plan-owned Plan-name validation to a raw observed
// value. Invalid values become stable violations and are not retained.
func ClassifyPlanName(value string) PlanNameObservation {
	if err := plan.ValidateName(value); err != nil {
		validation := planValidationError(err)
		return PlanNameObservation{state: scalarInvalid, violation: validation.Code()}
	}
	return PlanNameObservation{state: scalarValid, value: value}
}

// UnavailablePlanName represents unavailable Plan-name knowledge.
func UnavailablePlanName() PlanNameObservation {
	return PlanNameObservation{state: scalarUnavailable}
}

// ClassifyGoal applies Plan-owned Goal validation to a raw observed value.
func ClassifyGoal(value string) GoalObservation {
	goal, err := plan.NewGoal(value)
	if err != nil {
		validation := planValidationError(err)
		return GoalObservation{state: scalarInvalid, violation: validation.Code()}
	}
	return GoalObservation{state: scalarValid, value: goal.Text()}
}

// UnavailableGoal represents unavailable Goal knowledge.
func UnavailableGoal() GoalObservation {
	return GoalObservation{state: scalarUnavailable}
}

// ClassifyTargetDate applies Plan-owned target-date validation to a raw
// observed value.
func ClassifyTargetDate(value string) TargetDateObservation {
	targetDate, err := plan.ParseTargetDate(value)
	if err != nil {
		validation := planValidationError(err)
		return TargetDateObservation{state: targetDateInvalid, violation: validation.Code()}
	}
	return TargetDateObservation{state: targetDatePresent, value: targetDate}
}

// AbsentTargetDate represents known absence of a target date.
func AbsentTargetDate() TargetDateObservation {
	return TargetDateObservation{state: targetDateAbsent}
}

// UnavailableTargetDate represents unavailable target-date knowledge.
func UnavailableTargetDate() TargetDateObservation {
	return TargetDateObservation{state: targetDateUnavailable}
}

// CompleteAcceptanceConditions classifies raw acceptance-condition statements
// as complete membership. It applies Plan-owned typed element and exact
// duplicate validation, preserves each first valid member's exact bytes, and
// aggregates repeated invalid members by violation code. It performs no case,
// whitespace, line-ending, or Unicode normalization; nil and empty input are
// equivalent. The input slice is consumed during construction and never
// retained.
func CompleteAcceptanceConditions(values []string) AcceptanceConditionsObservation {
	return classifyAcceptanceConditions(values, collectionComplete)
}

// IncompleteAcceptanceConditions classifies raw acceptance-condition
// statements as incomplete membership. It applies Plan-owned typed element
// and exact duplicate validation, preserves each first valid member's exact
// bytes, and aggregates repeated invalid members by violation code. It
// performs no normalization; nil and empty input are equivalent. The input
// slice is consumed during construction and never retained.
func IncompleteAcceptanceConditions(values []string) AcceptanceConditionsObservation {
	return classifyAcceptanceConditions(values, collectionIncomplete)
}

// CompleteTasks classifies raw Task names as complete membership. It applies
// Plan-owned typed element and exact duplicate validation, preserves each
// first valid name's exact bytes, and aggregates repeated invalid members by
// violation code. It performs no case, whitespace, line-ending, or Unicode
// normalization; nil and empty input are equivalent. The input slice is
// consumed during construction and never retained.
func CompleteTasks(values []string) TasksObservation {
	return classifyTasks(values, collectionComplete)
}

// IncompleteTasks classifies raw Task names as incomplete membership. It
// applies Plan-owned typed element and exact duplicate validation, preserves
// each first valid name's exact bytes, and aggregates repeated invalid members
// by violation code. It performs no normalization; nil and empty input are
// equivalent. The input slice is consumed during construction and never
// retained.
func IncompleteTasks(values []string) TasksObservation {
	return classifyTasks(values, collectionIncomplete)
}

// planValidationError extracts the Plan-owned validation contract. Every
// classifier above calls a Plan constructor or validator that promises this
// error type; a missing type is an internal impossible path, not observed
// Provider evidence.
func planValidationError(err error) *plan.ValidationError {
	var validation *plan.ValidationError
	if !errors.As(err, &validation) {
		panic("unreachable: Plan validation did not return ValidationError")
	}
	return validation
}

func (o Observation) valid() bool {
	switch o.root {
	case rootAbsent, rootUnavailable:
		return o.name == (PlanNameObservation{}) &&
			o.goal == (GoalObservation{}) &&
			o.conditions.zero() &&
			o.tasks.zero() &&
			o.targetDate == (TargetDateObservation{})
	case rootPresent:
		return o.name.valid() &&
			o.goal.valid() &&
			o.conditions.valid() &&
			o.tasks.valid() &&
			o.targetDate.valid()
	default:
		return false
	}
}

func (o PlanNameObservation) valid() bool {
	switch o.state {
	case scalarValid:
		return o.violation == "" && plan.ValidateName(o.value) == nil
	case scalarInvalid:
		return o.value == "" && validScalarViolation(o.violation)
	case scalarUnavailable:
		return o.value == "" && o.violation == ""
	default:
		return false
	}
}

func (o GoalObservation) valid() bool {
	switch o.state {
	case scalarValid:
		_, err := plan.NewGoal(o.value)
		return o.violation == "" && err == nil
	case scalarInvalid:
		return o.value == "" && o.violation == plan.InvalidText
	case scalarUnavailable:
		return o.value == "" && o.violation == ""
	default:
		return false
	}
}

func (o TargetDateObservation) valid() bool {
	switch o.state {
	case targetDatePresent:
		_, err := plan.ParseTargetDate(o.value.String())
		return o.violation == "" && err == nil
	case targetDateAbsent, targetDateUnavailable:
		return o.value == (plan.TargetDate{}) && o.violation == ""
	case targetDateInvalid:
		return o.value == (plan.TargetDate{}) && o.violation == plan.InvalidTargetDate
	default:
		return false
	}
}

func (o AcceptanceConditionsObservation) valid() bool {
	if o.state != collectionComplete && o.state != collectionIncomplete {
		return false
	}
	seen := make(map[string]struct{}, len(o.members))
	for _, member := range o.members {
		if _, err := plan.NewAcceptanceCondition(member); err != nil {
			return false
		}
		if _, exists := seen[member]; exists {
			return false
		}
		seen[member] = struct{}{}
	}
	return validCollectionViolations(o.violations, validAcceptanceConditionViolation)
}

func (o AcceptanceConditionsObservation) zero() bool {
	return o.state == 0 && o.members == nil && o.violations == nil
}

func (o TasksObservation) valid() bool {
	if o.state != collectionComplete && o.state != collectionIncomplete {
		return false
	}
	seen := make(map[string]struct{}, len(o.members))
	for _, member := range o.members {
		if _, err := plan.NewTask(member); err != nil {
			return false
		}
		if _, exists := seen[member]; exists {
			return false
		}
		seen[member] = struct{}{}
	}
	return validCollectionViolations(o.violations, validTaskViolation)
}

func (o TasksObservation) zero() bool {
	return o.state == 0 && o.members == nil && o.violations == nil
}

func validScalarViolation(value plan.ViolationCode) bool {
	switch value {
	case plan.InvalidText, plan.MultilineName:
		return true
	default:
		return false
	}
}

func validCollectionViolations(values []plan.ViolationCode, allowed func(plan.ViolationCode) bool) bool {
	seen := make(map[plan.ViolationCode]struct{}, len(values))
	for _, value := range values {
		if !allowed(value) {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validAcceptanceConditionViolation(value plan.ViolationCode) bool {
	switch value {
	case plan.InvalidText, plan.DuplicateAcceptanceCondition:
		return true
	default:
		return false
	}
}

func validTaskViolation(value plan.ViolationCode) bool {
	switch value {
	case plan.InvalidText, plan.MultilineName, plan.DuplicateTask:
		return true
	default:
		return false
	}
}

func classifyAcceptanceConditions(values []string, state collectionObservationState) AcceptanceConditionsObservation {
	members := make([]string, 0, len(values))
	validValues := make([]plan.AcceptanceCondition, 0, len(values))
	violations := make(map[plan.ViolationCode]struct{})
	for _, raw := range values {
		value, err := plan.NewAcceptanceCondition(raw)
		if err != nil {
			validation := planValidationError(err)
			violations[validation.Code()] = struct{}{}
			continue
		}
		candidate := append(append([]plan.AcceptanceCondition(nil), validValues...), value)
		if err := plan.ValidateAcceptanceConditions(candidate); err != nil {
			validation := planValidationError(err)
			violations[validation.Code()] = struct{}{}
			continue
		}
		validValues = append(validValues, value)
		members = append(members, value.Statement())
	}
	return AcceptanceConditionsObservation{
		state:      state,
		members:    members,
		violations: violationCodes(violations),
	}
}

func classifyTasks(values []string, state collectionObservationState) TasksObservation {
	members := make([]string, 0, len(values))
	validValues := make([]plan.Task, 0, len(values))
	violations := make(map[plan.ViolationCode]struct{})
	for _, raw := range values {
		value, err := plan.NewTask(raw)
		if err != nil {
			validation := planValidationError(err)
			violations[validation.Code()] = struct{}{}
			continue
		}
		candidate := append(append([]plan.Task(nil), validValues...), value)
		if err := plan.ValidateTasks(candidate); err != nil {
			validation := planValidationError(err)
			violations[validation.Code()] = struct{}{}
			continue
		}
		validValues = append(validValues, value)
		members = append(members, value.Name())
	}
	return TasksObservation{
		state:      state,
		members:    members,
		violations: violationCodes(violations),
	}
}

func violationCodes(values map[plan.ViolationCode]struct{}) []plan.ViolationCode {
	codes := make([]plan.ViolationCode, 0, len(values))
	for code := range values {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return string(codes[i]) < string(codes[j]) })
	return codes
}
