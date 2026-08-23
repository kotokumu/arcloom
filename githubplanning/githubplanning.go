// Package githubplanning creates immutable, non-executed GitHub request plans.
package githubplanning

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kotokumu/arcloom/plan"
)

// ViolationCode identifies a rejected GitHub-planning input.
type ViolationCode string

const (
	InvalidRepository         ViolationCode = "invalid_repository"
	InvalidRepresentation     ViolationCode = "invalid_representation"
	InvalidPlan               ViolationCode = "invalid_plan"
	UnsupportedRepresentation ViolationCode = "unsupported_representation"
)

// Field identifies the input associated with a validation error.
type Field uint8

const (
	RepositoryOwnerField Field = iota + 1
	RepositoryNameField
	RepresentationField
	PlanField
)

// ValidationError describes one rejected GitHub-planning input.
type ValidationError struct {
	code  ViolationCode
	field Field
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "github planning validation error"
	}
	return fmt.Sprintf("github planning validation failed: %s (%d)", e.code, e.field)
}
func (e *ValidationError) Code() ViolationCode {
	if e == nil {
		return ""
	}
	return e.code
}
func (e *ValidationError) Field() Field {
	if e == nil {
		return 0
	}
	return e.field
}

// Repository identifies a GitHub owner and repository path segment.
type Repository struct{ owner, name string }

func NewRepository(owner, name string) (Repository, error) {
	if !validRepositorySegment(owner) {
		return Repository{}, &ValidationError{code: InvalidRepository, field: RepositoryOwnerField}
	}
	if !validRepositorySegment(name) {
		return Repository{}, &ValidationError{code: InvalidRepository, field: RepositoryNameField}
	}
	return Repository{owner: owner, name: name}, nil
}

func (r Repository) Owner() string { return r.owner }
func (r Repository) Name() string  { return r.name }

// Representation selects the explicit GitHub-native representation of a Plan.
type Representation uint8

const (
	MilestoneRepresentation Representation = iota + 1
	IssueRepresentation
)

// RESTAPIVersion is the fixed GitHub.com REST API version represented by requests.
const RESTAPIVersion = "2022-11-28"

// ResultKind identifies the type of provider-assigned result used by a request.
type ResultKind uint8

const (
	MilestoneNumber ResultKind = iota + 1
	IssueNumber
	IssueID
)

// RequestPosition is a zero-based position in a RequestPlan's topological order.
type RequestPosition uint32

// ResultReference points to an earlier request's typed result.
type ResultReference struct {
	source RequestPosition
	kind   ResultKind
}

func (r ResultReference) Source() RequestPosition { return r.source }
func (r ResultReference) Kind() ResultKind        { return r.kind }

// CreateMilestoneRequest is the input for one GitHub milestone creation.
type CreateMilestoneRequest struct {
	title, description string
	dueOn              string
	hasDueOn           bool
}

func (r CreateMilestoneRequest) Title() string         { return r.title }
func (r CreateMilestoneRequest) Description() string   { return r.description }
func (r CreateMilestoneRequest) DueOn() (string, bool) { return r.dueOn, r.hasDueOn }
func (CreateMilestoneRequest) requestMarker()          {}

// CreateIssueRequest is the input for one GitHub issue creation.
type CreateIssueRequest struct {
	title, body  string
	hasBody      bool
	milestone    ResultReference
	hasMilestone bool
}

func (r CreateIssueRequest) Title() string                      { return r.title }
func (r CreateIssueRequest) Body() (string, bool)               { return r.body, r.hasBody }
func (r CreateIssueRequest) Milestone() (ResultReference, bool) { return r.milestone, r.hasMilestone }
func (CreateIssueRequest) requestMarker()                       {}

// AddSubIssueRequest is the input for one GitHub sub-issue relationship.
type AddSubIssueRequest struct {
	parentIssueNumber ResultReference
	subIssueID        ResultReference
}

func (r AddSubIssueRequest) ParentIssueNumber() ResultReference { return r.parentIssueNumber }
func (r AddSubIssueRequest) SubIssueID() ResultReference        { return r.subIssueID }
func (AddSubIssueRequest) requestMarker()                       {}

// Request is the sealed sum of supported GitHub request inputs.
type Request interface{ requestMarker() }

// RequestPlan is an immutable, dependency-aware creation request plan.
type RequestPlan struct {
	repository     Repository
	representation Representation
	requests       []Request
}

func NewCreationRequestPlan(repository Repository, representation Representation, value plan.Plan) (RequestPlan, error) {
	if !validRepository(repository) {
		return RequestPlan{}, &ValidationError{code: InvalidRepository, field: RepositoryOwnerField}
	}
	if representation != MilestoneRepresentation && representation != IssueRepresentation {
		return RequestPlan{}, &ValidationError{code: InvalidRepresentation, field: RepresentationField}
	}
	if !value.IsValid() {
		return RequestPlan{}, &ValidationError{code: InvalidPlan, field: PlanField}
	}
	var requests []Request
	if representation == MilestoneRepresentation {
		requests = milestoneRequests(value)
	} else {
		var supported bool
		requests, supported = issueRequests(value)
		if !supported {
			return RequestPlan{}, &ValidationError{code: UnsupportedRepresentation, field: RepresentationField}
		}
	}
	return RequestPlan{repository: repository, representation: representation, requests: requests}, nil
}

func (p RequestPlan) Repository() Repository         { return p.repository }
func (p RequestPlan) Representation() Representation { return p.representation }
func (p RequestPlan) APIVersion() string {
	if p.representation == 0 {
		return ""
	}
	return RESTAPIVersion
}
func (p RequestPlan) Requests() []Request { return append([]Request(nil), p.requests...) }

func validRepository(value Repository) bool {
	return validRepositorySegment(value.owner) && validRepositorySegment(value.name)
}

func validRepositorySegment(value string) bool {
	if !utf8.ValidString(value) || strings.TrimFunc(value, unicode.IsSpace) == "" || strings.Contains(value, "/") {
		return false
	}
	return !strings.ContainsAny(value, "\r\n\u0085\u2028\u2029")
}

type requestPlanConstruction struct{ requests []Request }

func newRequestPlanConstruction(capacity int) *requestPlanConstruction {
	return &requestPlanConstruction{requests: make([]Request, 0, capacity)}
}

func (c *requestPlanConstruction) addMilestone(request CreateMilestoneRequest) milestoneNumberResult {
	position := RequestPosition(len(c.requests))
	c.requests = append(c.requests, request)
	return milestoneNumberResult{position: position}
}

func (c *requestPlanConstruction) addIssue(request CreateIssueRequest) createdIssueResult {
	position := RequestPosition(len(c.requests))
	c.requests = append(c.requests, request)
	return createdIssueResult{number: issueNumberResult{position: position}, id: issueIDResult{position: position}}
}

func (c *requestPlanConstruction) addSubIssue(parent issueNumberResult, child issueIDResult) {
	c.requests = append(c.requests, AddSubIssueRequest{parentIssueNumber: parent.reference(), subIssueID: child.reference()})
}

func (c *requestPlanConstruction) snapshot() []Request {
	return append([]Request(nil), c.requests...)
}

type milestoneNumberResult struct{ position RequestPosition }

func (r milestoneNumberResult) reference() ResultReference {
	return ResultReference{source: r.position, kind: MilestoneNumber}
}

type issueNumberResult struct{ position RequestPosition }

func (r issueNumberResult) reference() ResultReference {
	return ResultReference{source: r.position, kind: IssueNumber}
}

type issueIDResult struct{ position RequestPosition }

func (r issueIDResult) reference() ResultReference {
	return ResultReference{source: r.position, kind: IssueID}
}

type createdIssueResult struct {
	number issueNumberResult
	id     issueIDResult
}

func milestoneRequests(value plan.Plan) []Request {
	tasks := value.Tasks()
	construction := newRequestPlanConstruction(1 + len(tasks))
	milestone := CreateMilestoneRequest{title: value.Name(), description: narrative(value)}
	if date, ok := value.TargetDate(); ok {
		milestone.dueOn = date.String() + "T00:00:00Z"
		milestone.hasDueOn = true
	}
	milestoneResult := construction.addMilestone(milestone)
	for _, task := range tasks {
		construction.addIssue(CreateIssueRequest{
			title:        task.Name(),
			milestone:    milestoneResult.reference(),
			hasMilestone: true,
		})
	}
	return construction.snapshot()
}

func issueRequests(value plan.Plan) ([]Request, bool) {
	tasks := value.Tasks()
	if len(tasks) > 100 {
		return nil, false
	}
	construction := newRequestPlanConstruction(1 + len(tasks)*2)
	parent := construction.addIssue(CreateIssueRequest{title: value.Name(), body: issueNarrative(value), hasBody: true})
	for _, task := range tasks {
		child := construction.addIssue(CreateIssueRequest{title: task.Name()})
		construction.addSubIssue(parent.number, child.id)
	}
	return construction.snapshot(), true
}

func narrative(value plan.Plan) string {
	var builder strings.Builder
	builder.WriteString("## Goal\n\n")
	builder.WriteString(value.Goal().Text())
	builder.WriteString("\n\n## Acceptance Conditions")
	for i, condition := range value.AcceptanceConditions() {
		fmt.Fprintf(&builder, "\n\n### %d\n\n", i+1)
		builder.WriteString(condition.Statement())
	}
	builder.WriteByte('\n')
	return builder.String()
}

func issueNarrative(value plan.Plan) string {
	text := narrative(value)
	if date, ok := value.TargetDate(); ok {
		text += "\n## Target Date\n\n" + date.String() + "\n"
	}
	return text
}
