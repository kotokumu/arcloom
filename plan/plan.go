package plan

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ViolationCode identifies a stable class of rejected Plan input.
type ViolationCode string

const (
	InvalidText                  ViolationCode = "invalid_text"
	MultilineName                ViolationCode = "multiline_name"
	InvalidTargetDate            ViolationCode = "invalid_target_date"
	MissingAcceptanceCondition   ViolationCode = "missing_acceptance_condition"
	DuplicateAcceptanceCondition ViolationCode = "duplicate_acceptance_condition"
	DuplicateTask                ViolationCode = "duplicate_task"
)

// ElementKind identifies the Plan element associated with a validation error.
type ElementKind uint8

const (
	PlanNameElement ElementKind = iota + 1
	GoalElement
	AcceptanceConditionElement
	TaskElement
	TargetDateElement
)

// ValidationError describes one rejected Plan value or aggregate input.
type ValidationError struct {
	code     ViolationCode
	element  ElementKind
	index    int
	hasIndex bool
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "plan validation error"
	}
	if e.hasIndex {
		return fmt.Sprintf("plan validation failed: %s (%d, index %d)", e.code, e.element, e.index)
	}
	return fmt.Sprintf("plan validation failed: %s (%d)", e.code, e.element)
}

func (e *ValidationError) Code() ViolationCode {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *ValidationError) Element() ElementKind {
	if e == nil {
		return 0
	}
	return e.element
}

func (e *ValidationError) Index() (int, bool) {
	if e == nil {
		return 0, false
	}
	return e.index, e.hasIndex
}

// Goal is the desired outcome text of a Plan.
type Goal struct{ text string }

func NewGoal(text string) (Goal, error) {
	if err := validateText(text, GoalElement, false, -1); err != nil {
		return Goal{}, err
	}
	return Goal{text: text}, nil
}

func (g Goal) Text() string { return g.text }

// AcceptanceCondition is one criterion used to accept a Plan's Goal.
type AcceptanceCondition struct{ statement string }

func NewAcceptanceCondition(statement string) (AcceptanceCondition, error) {
	if err := validateText(statement, AcceptanceConditionElement, false, -1); err != nil {
		return AcceptanceCondition{}, err
	}
	return AcceptanceCondition{statement: statement}, nil
}

func (c AcceptanceCondition) Statement() string { return c.statement }

// Task is one named unit of work in a Plan.
type Task struct{ name string }

func NewTask(name string) (Task, error) {
	if err := validateText(name, TaskElement, true, -1); err != nil {
		return Task{}, err
	}
	return Task{name: name}, nil
}

func (t Task) Name() string { return t.name }

// TargetDate is a provider-independent Gregorian calendar date.
type TargetDate struct{ text string }

func ParseTargetDate(value string) (TargetDate, error) {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return TargetDate{}, &ValidationError{code: InvalidTargetDate, element: TargetDateElement}
	}
	for i, r := range value {
		if i == 4 || i == 7 {
			continue
		}
		if r < '0' || r > '9' {
			return TargetDate{}, &ValidationError{code: InvalidTargetDate, element: TargetDateElement}
		}
	}
	year := atoi(value[0:4])
	month := atoi(value[5:7])
	day := atoi(value[8:10])
	if year == 0 || month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month) {
		return TargetDate{}, &ValidationError{code: InvalidTargetDate, element: TargetDateElement}
	}
	return TargetDate{text: value}, nil
}

func (d TargetDate) String() string { return d.text }

// Plan is immutable provider-independent planning intent.
type Plan struct {
	name       string
	goal       Goal
	conditions []AcceptanceCondition
	tasks      []Task
	targetDate *TargetDate
	valid      bool
}

func New(name string, goal Goal, conditions []AcceptanceCondition, tasks []Task, targetDate *TargetDate) (Plan, error) {
	if err := validateText(name, PlanNameElement, true, -1); err != nil {
		return Plan{}, err
	}
	if err := validateText(goal.text, GoalElement, false, -1); err != nil {
		return Plan{}, err
	}
	if len(conditions) == 0 {
		return Plan{}, &ValidationError{code: MissingAcceptanceCondition, element: AcceptanceConditionElement}
	}
	seenConditions := make(map[string]struct{}, len(conditions))
	for i, condition := range conditions {
		if err := validateText(condition.statement, AcceptanceConditionElement, false, i); err != nil {
			return Plan{}, err
		}
		if _, exists := seenConditions[condition.statement]; exists {
			return Plan{}, &ValidationError{code: DuplicateAcceptanceCondition, element: AcceptanceConditionElement, index: i, hasIndex: true}
		}
		seenConditions[condition.statement] = struct{}{}
	}
	seenTasks := make(map[string]struct{}, len(tasks))
	for i, task := range tasks {
		if err := validateText(task.name, TaskElement, true, i); err != nil {
			return Plan{}, err
		}
		if _, exists := seenTasks[task.name]; exists {
			return Plan{}, &ValidationError{code: DuplicateTask, element: TaskElement, index: i, hasIndex: true}
		}
		seenTasks[task.name] = struct{}{}
	}
	var dateCopy *TargetDate
	if targetDate != nil {
		if _, err := ParseTargetDate(targetDate.text); err != nil {
			return Plan{}, err
		}
		date := *targetDate
		dateCopy = &date
	}
	return Plan{
		name: name, goal: goal,
		conditions: append([]AcceptanceCondition(nil), conditions...),
		tasks:      append([]Task(nil), tasks...), targetDate: dateCopy, valid: true,
	}, nil
}

func (p Plan) Name() string { return p.name }
func (p Plan) Goal() Goal   { return p.goal }
func (p Plan) AcceptanceConditions() []AcceptanceCondition {
	return append([]AcceptanceCondition(nil), p.conditions...)
}
func (p Plan) Tasks() []Task { return append([]Task(nil), p.tasks...) }
func (p Plan) TargetDate() (TargetDate, bool) {
	if p.targetDate == nil {
		return TargetDate{}, false
	}
	return *p.targetDate, true
}

// IsValid reports whether the Plan was successfully constructed.
func (p Plan) IsValid() bool { return p.valid }

func validateText(value string, element ElementKind, singleLine bool, index int) *ValidationError {
	if !utf8.ValidString(value) || strings.TrimFunc(value, unicode.IsSpace) == "" {
		return &ValidationError{code: InvalidText, element: element, index: index, hasIndex: index >= 0}
	}
	if singleLine && strings.ContainsAny(value, "\r\n\u0085\u2028\u2029") {
		return &ValidationError{code: MultilineName, element: element, index: index, hasIndex: index >= 0}
	}
	return nil
}

func atoi(s string) int {
	result := 0
	for _, r := range s {
		result = result*10 + int(r-'0')
	}
	return result
}

func daysInMonth(year, month int) int {
	days := [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month == 2 && (year%400 == 0 || year%4 == 0 && year%100 != 0) {
		return 29
	}
	return days[month-1]
}
