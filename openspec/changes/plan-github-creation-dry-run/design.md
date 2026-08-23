# GitHub Plan Creation Dry-Run

## 0. Document Scope

| Information | Governing document |
|---|---|
| Product value, capability boundaries, and principles | `PRODUCT.md` |
| Component ownership, external Contexts, and dependency direction | `ARCHITECTURE.md` |
| Accepted behavior of Plan and the GitHub dry-run | `specs/plan/spec.md` and `specs/github-plan-creation-dry-run/spec.md` |
| Concepts, responsibilities, package boundaries, public contracts, tests, and decisions for this Change | This DesignDoc |
| Construction progress | `tasks.md` |

---

## 1. Purpose / Out of Scope

### 1.1 Purpose

The Change establishes a Provider-independent Plan and proves that one Plan can produce an inspectable GitHub creation request plan in either Milestone or Issue form without external state or manually prepared fixtures.

### 1.2 Out of Scope

- Sending a GitHub request
- Reading or reconciling existing GitHub resources
- Updating, deleting, retrying, or resuming an external mutation
- Change Authorization
- A GitHub API client, SDK, token, or permission check
- Persistent state, a CLI, or a server
- Providers other than GitHub
- GitHub Enterprise Server and reverse reconstruction of a Plan from rendered Markdown
- Plan hierarchy, grouping, dependencies, Task details, and Plan sufficiency

### 1.3 Risk Assessment and Approval

| Risk | Trigger | Required gate | Status |
|---|---|---|---|
| High | New public Go contracts and mapping to an external GitHub API contract | Complete Structure-Behavior Design workflow and human review before construction | Initial design and the post-review validity/builder correction were approved by the user on 2026-08-23 |

---

## 2. Behavior Design

### 2.1 Functional Requirements

#### Provider-independent Plan

The Plan contract protects the meaning and validity used by every external representation.

| ID | Rule | Source |
|---|---|---|
| FR-1 | A valid Plan contains a name, Goal, at least one acceptance condition, optional target date, and zero or more Tasks. | PLN-1 |
| FR-2 | Required text is non-blank valid UTF-8, names are single-line, and valid text is preserved exactly. | PLN-2 |
| FR-3 | Acceptance-condition statements and Task names are exact-text unique and retain declared order. | PLN-3 |
| FR-4 | A target date is a valid Provider-independent Gregorian calendar date. | PLN-4 |
| FR-5 | Constructor failures return a typed, located validation result and no partial value; Plan itself reports whether it is valid so Provider packages do not reconstruct its invariants. | PLN-5 |

#### GitHub Plan representation

The GitHub Planning Provider derives one structurally valid request plan for the explicitly selected representation.

| ID | Rule | Source |
|---|---|---|
| FR-6 | A locally valid Repository target and one explicit Milestone or Issue representation are required. | GPCD-1 |
| FR-7 | Goal, acceptance conditions, and the Issue representation's target date use the canonical Markdown narrative. | GPCD-2 |
| FR-8 | Milestone representation produces one Milestone creation template and one associated Issue creation template per Task. | GPCD-3 |
| FR-9 | Issue representation produces one parent Issue creation template and one Task-Issue and sub-issue-relation template pair per Task, up to GitHub's limit of 100 sub-issues. | GPCD-4 |
| FR-10 | The request plan is deterministic, topologically ordered, immutable to consumers, and contains only valid typed backward references. | GPCD-5 |
| FR-11 | Planning targets the fixed GitHub.com REST contract, produces no external side effect, and makes no claim about external executability or success. | GPCD-6 |
| FR-12 | Dry-run failures return a typed input result, no partial plan, and distinguish unsupported representation from invalid input. | GPCD-7 |

### 2.2 Non-Functional Requirements

| Requirement | Threshold | Measurement |
|---|---|---|
| Race safety | All package tests pass under the Go race detector | `go test -v -race ./...` |
| Production dependency hygiene | The provider-independent Planning Model has no GitHub dependency; production code in `githubplanning` has no network, process, environment, filesystem, persistence, or third-party dependency | Non-test direct-import allowlist inspection, `go list -deps ./...`, and `go mod tidy -diff`; `go-cmp` remains test-only under the repository's Go test-authoring workflow |
| Static quality | No configured lint finding | `golangci-lint run` |

### 2.3 Requirements Analysis

#### Problem, Current Behavior, and Desired Behavior

| Aspect | Definition |
|---|---|
| Problem | A read-only first slice requires manually prepared GitHub facts and does not verify creation mapping or response-dependent identifiers. |
| Current behavior | The Go module contains only a build-check package. It has no Plan or GitHub planning behavior. |
| Desired behavior | Composition code supplies a valid Plan, Repository target, and representation to a concrete GitHub Planning Provider and receives an immutable GitHub API request plan without network access. |

#### Inputs and Outputs

| Direction | Value | Contract |
|---|---|---|
| Input | Plan | Valid Provider-independent value |
| Input | GitHub Repository target | Locally valid owner and repository path segments |
| Input | Representation | Exactly Milestone or Issue |
| Output | Request Plan | Topologically ordered GitHub request templates with typed symbolic references |
| Error | Validation error | Identifies invalid Plan, Repository, or representation; no partial plan is returned |

#### Current-System Evidence

Evidence packet ID: `GCRD-v1`, frozen on 2026-08-21.

| Fact | Source | Relevance |
|---|---|---|
| The Go module contains only `internal/buildcheck` and has no Plan or GitHub Provider behavior. | Repository tree and `go.mod` | Establishes current behavior and absence of compatibility obligations |
| Arcloom owns no authoritative durable business or control state. | `AGENTS.md`; `ARCHITECTURE.md` 2.1 and 3.1 | Prohibits persistence of Request Plans or invented Provider facts |
| Plan is Provider-independent; Milestones and Issues can be Planning Context representations. | `PRODUCT.md` 4.1/4.2; `ARCHITECTURE.md` 3.3.1 | Fixes concept ownership and dependency direction |
| The accepted first slice creates no external facts and returns a GitHub request preview for Milestone or Issue representation. | User decision in this Change | Fixes desired behavior and non-goals |
| Milestone creation returns a `number`; Issue creation accepts that `number` in `milestone`. | [GitHub REST Milestones](https://docs.github.com/en/rest/issues/milestones); [GitHub REST Issues](https://docs.github.com/en/rest/issues/issues), API version `2022-11-28`, accessed 2026-08-21 | Requires a typed symbolic Milestone-number result |
| Add-sub-issue uses parent Issue `number` in the path and child Issue `id` in the body. | [GitHub REST Sub-issues](https://docs.github.com/en/rest/issues/sub-issues), API version `2022-11-28`, accessed 2026-08-21 | Requires distinct typed Issue-number and Issue-ID results |
| One parent Issue supports at most 100 sub-issues. | [GitHub sub-issue documentation](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues), accessed 2026-08-21 | Bounds the Issue representation without bounding Plan itself |

#### GitHub Request Contract

| Request value | REST method and path | Literal input | Symbolic input | Result used later |
|---|---|---|---|---|
| Create Milestone | `POST /repos/{owner}/{repo}/milestones` | `title`, `description`, optional `due_on` | None | Milestone `number` |
| Create Issue | `POST /repos/{owner}/{repo}/issues` | `title`, optional `body` | Optional Milestone `number` | Issue `number` and `id` |
| Add sub-issue | `POST /repos/{owner}/{repo}/issues/{parent_number}/sub_issues` | None | Parent Issue `number`; child Issue `id` | None in this Change |

#### Error and Edge Cases

| Case | Required result |
|---|---|
| Zero Tasks | One root-resource creation request remains |
| Multiple Tasks | Declared Task order determines request order and symbolic-reference targets |
| Missing target date | No Milestone `due_on` or Issue target-date narrative section |
| Unicode text | Preserved without normalization |
| Invalid or zero Plan | Typed validation error; no request plan |
| Invalid Repository or representation | Typed GitHub-planning validation error; no request plan |
| Repository is absent or inaccessible | Not locally detectable; the dry-run makes no success claim |
| Issue representation has 101 Tasks | Unsupported-representation error; no partial Request Plan |
| Goal or condition contains delimiter-like Markdown or trailing LF | Bytes are inserted unchanged; fixed delimiters are still added |

### 2.4 Test Specification

#### Requirement Coverage

| Requirement | Observable behavior | Automated evidence |
|---|---|---|
| PLN-1 | Minimum Plan is valid and contains no Milestone concept | Public table-driven Plan tests |
| PLN-2 | Unicode preservation and every invalid text boundary | Public table-driven text tests |
| PLN-3 | Duplicate rejection, order preservation, and defensive copies | Plan aggregate tests |
| PLN-4 | Gregorian boundaries and canonical date text | Target-date tests |
| PLN-5 | Stable error category, element/index, `errors.As`, zero result, and Plan-owned validity | Constructor error-contract and Plan-validity tests |
| GPCD-1 | Both representations and invalid target/representation results | GitHub-planning contract tests |
| GPCD-2 | Exact Markdown for one/multiple conditions and target-date variants | Narrative output tests through public request values |
| GPCD-3 | Request count, values, order, and Milestone-number references | Milestone representation tests |
| GPCD-4 | Request count, values, order, parent-number and child-ID references | Issue representation tests |
| GPCD-5 | Repeatability, typed backward references, zero-value rejection, defensive copies | Public request-reference property tests and API-surface verification |
| GPCD-6 | No client/network contract or side effect and no success state in output | Exported-API and dependency review plus unit tests |
| GPCD-7 | Stable field/category, zero Plan handling, unsupported result, and zero output | Dry-run error-contract tests |

#### Behavior Tests

| Behavior | Given | When | Then | Level |
|---|---|---|---|---|
| Minimum Plan | Name, Goal, one condition | Construct Plan | Valid Plan with no Task/date | Unit |
| Plan validity ownership | Constructed and zero Plan values | Query validity | Constructed Plan reports valid; zero Plan reports invalid without a Provider reconstructing Plan rules | Unit |
| Exact uniqueness | Equal and near-equal Unicode values | Construct Plan | Exact duplicate rejected; near-equal preserved | Unit |
| Milestone request plan | Plan with two Tasks and date | Plan creation | Milestone request then two Issue requests referencing its number | Unit |
| Issue request plan | Plan with two Tasks and date | Plan creation | Parent Issue then two child/relation pairs with typed references | Unit |
| Empty Task collection | Valid Plan without Tasks | Plan creation in either representation | Exactly one root request | Unit |
| Invalid public values | Zero/invalid Plan, Repository, representation | Plan creation | Typed error and invalid zero Request Plan | Unit |
| Defensive ownership | Caller mutates input or returned slices | Read values again | Source values are unchanged | Unit |
| Issue representation limit | Plans with 100 and 101 Tasks | Plan creation | Complete plan at 100; unsupported and zero plan at 101 | Unit |
| API conformance | Each returned public request value | Inspect the preview | Version, operation shape, optional fields, and result kinds match the frozen contract table | Contract |
| Non-side-effect boundary | Package imports and exported API | Run the reproducible verification | No network/client/executor/persistence dependency or success result exists | Architecture |
| Result-reference integrity | Milestone Tasks at 0, 1, and representative multiple counts; Issue Tasks from 0 through 100 | Traverse only the public `Requests()` result | Every reference source precedes its consumer in the same slice; Milestone input uses the root Milestone `number`; parent uses the root Issue `number`; child uses its corresponding Task Issue `id` | Property |

#### Edge-Case Matrix

| Dimension | Cases |
|---|---|
| Representation | Milestone; Issue |
| Task count | 0; 1; 2; Issue-only 100 and 101 |
| Target date | Absent; present |
| Acceptance conditions | 1; multiple |
| Text | ASCII; non-normalized Unicode variants; leading/trailing whitespace; CRLF; delimiter-like Markdown; trailing LF |
| Repository | owner/name valid; either blank; `/`; line-break; otherwise unusual but locally accepted text |
| Target date boundary | `0001-01-01`; `9999-12-31`; 1900/2000/2028 leap rules; zero/non-zero padding; invalid month/day; whitespace; time suffix |
| Zero values | Every exported value accessor is panic-free; zero Plan/Repository/representation is rejected at planning boundary |

#### Verification Methods

| Criterion | Owner | Method | Required evidence / pass predicate |
|---|---|---|---|
| Automated behavior | Change implementer | `go test -v -race ./...` | Exit 0; every requirements table row has a named test |
| Module cleanliness | Change implementer | `go mod tidy -diff` | Exit 0 and empty diff |
| Static quality | Change implementer | `golangci-lint run` | Exit 0 |
| Dependency boundary | Architecture reviewer | Inspect non-test direct imports with `go list -json ./plan ./githubplanning` and full production dependencies with `go list -deps ./...` | `plan` uses standard library only; `githubplanning` directly imports only `plan` and approved pure standard-library packages; neither production Package directly imports `net`, `net/http`, `os`, `os/exec`, persistence, or a third-party module; transitive standard-library internals are not direct boundary dependencies |
| Public surface | Interface reviewer | `go doc -all ./plan` and `go doc -all ./githubplanning` | No client, executor, persistence, external-success result, public request/reference constructor, or Core/Module Port contains GitHub-specific types |

#### TDD Plan

| Behavior | Mode | Red test | Smallest implementation | Refactor target |
|---|---|---|---|---|
| Plan text and dates | Red-Green-Refactor | Invalid/valid boundary table | Local immutable values | One text-validation owner |
| Plan aggregate | Red-Green-Refactor | Cardinality, duplicates, order, snapshots | Plan constructor and accessors | One aggregate-invariant owner |
| Plan validity ownership | Red-Green-Refactor | `IsValid` is missing and a zero Plan crosses into Provider-side rule reconstruction | `Plan.IsValid()` backed by Plan-owned construction state | Remove Provider reconstruction of Plan invariants |
| GitHub target and representations | Red-Green-Refactor | Target/representation table | Repository value and representation enum | Remove provider primitives from call sites |
| Milestone request plan | Red-Green-Refactor | Exact public request assertions | Minimum request values and references | Centralize graph validity |
| Issue request plan | Red-Green-Refactor | Exact public request assertions | Reuse the same request/reference meanings | Remove duplicated narrative construction |
| Request graph construction | Baseline-Green refactor | Public reference-integrity property tests | Private Request Plan construction value issues positions and typed references while requests are added | Remove post-hoc graph reinterpretation and input-error misattribution |
| Test contract correction | Baseline-Green test expansion | Passing behavior baseline with incomplete error/oracle coverage | `gotests` table scaffolds compare literal public observations and every typed error location | Remove self-referential constructor oracles and private-layout coupling |
| Boundary verification | Automated verification | Passing package behavior baseline | No network/client implementation | Exported API and dependency review |

---

## 3. Structure Design

### 3.1 Conceptual Model

| Concept | Meaning | Identity | Rule / Invariant |
|---|---|---|---|
| Plan | Provider-independent planning intent | Its value; no Provider identity or Arcloom-owned lifecycle | Contains required Plan meaning, reports its own validity, and contains no Milestone or Provider identifier |
| Goal | Desired outcome of one Plan | Its preserved text within the Plan | Non-blank valid UTF-8 |
| Acceptance Condition | Condition for accepting achievement of the Goal | Its exact statement within the Plan | At least one; exact-text unique |
| Task | Work included in the current Plan | Its exact name within the Plan | Zero or more; exact-name unique; no detail or relationship in this Change |
| Target Date | Provider-independent calendar date associated with the Plan | Canonical `YYYY-MM-DD` value | Valid Gregorian date; no time-zone or Provider semantics |
| GitHub Repository Target | Repository to which requests would be sent | Owner and repository path segments | Local validity does not imply existence or access |
| GitHub Plan Representation | Explicit Milestone or Issue rules for representing a Plan | Selected representation value | Exactly one supported representation; never inferred |
| Creation Request Plan | Complete unexecuted GitHub creation intent | One transient value per planning call | Deterministic topological request order and closed typed references |
| Request Template | One known GitHub API client input whose response may be referenced | Its ordinal within a Request Plan | Has all literal inputs and only valid backward result references |
| Symbolic Result Reference | Typed reference to a Provider-assigned result not yet available | Source request ordinal plus result kind | Source precedes consumer; result kind matches the consuming field |

GitHub Milestones, Issues, their relationships, identifiers, meanings, and lifecycles remain external facts owned by GitHub. The Request Plan does not assert that any of them exist.

#### Relationships

| Concept A | Relationship | Concept B | Cardinality / consistency |
|---|---|---|---|
| Plan | has | Goal | Exactly 1 |
| Plan | has | Acceptance Condition | 1..*, exact-text unique |
| Plan | has | Task | 0..*, exact-name unique |
| Plan | has | Target Date | 0..1 |
| GitHub Plan Representation | represents | Plan | Exactly one explicit representation per call |
| Creation Request Plan | targets | GitHub Repository Target | Exactly 1 |
| Creation Request Plan | contains | Request Template | 1..*, topologically ordered |
| Request Template | consumes | Symbolic Result Reference | 0..*, backward and type-compatible |

#### Concept Minimality

| Candidate | Remove / merge test | Decision | Reason |
|---|---|---|---|
| Plan and GitHub resource | Merge without Provider identity or lifecycle leakage | Keep separate | Merge violates Provider independence and external authority |
| Goal and Acceptance Condition | Merge while retaining one outcome and multiple acceptance criteria | Keep separate | Meaning and cardinality differ |
| Target Date and GitHub `due_on` | Merge while preserving Issue representation and calendar-date meaning | Keep separate value | `due_on` is only one Provider encoding |
| Request Plan and Request Template | Merge while retaining graph-wide closure and per-operation input rules | Keep separate values | Their invariants have different scope |
| Symbolic reference and untyped dependency edge | Merge while retaining `number` versus `id` correctness | Keep typed reference | Order alone loses result meaning |
| Generator, Builder, Executor, Registry | Remove without losing identity, authority, lifecycle, or invariant | Reject as concepts | They only name procedures or speculative extension points |
| Authorization | Remove from a non-mutating dry-run | Exclude | No external application decision occurs |

### 3.2 Independent Evolution Scenarios and Responsibility Stress Test

The conceptual-model author and the independent scenario author received the same frozen evidence packet `GCRD-v1`. The scenario author received no proposed model, Package, Interface, or expected finding. The scenario set was produced independently before responsibility and boundary review.

| Scenario / confidence | Primary owner | Expected propagation | Verdict |
|---|---|---|---|
| Representation changes per call / Committed | GitHub Plan Representation | GitHub request-planning tests only | Pass |
| Task count changes from zero to many / Committed | Plan cardinality and Request Plan graph | Plan and GitHub-planning tests | Pass |
| Acceptance conditions increase / Committed | Plan uniqueness/order and GitHub narrative | Plan and GitHub-planning tests | Pass |
| Target date is present or absent / Committed | Target Date and representation mapping | Date and both representation tests | Pass |
| Unicode normalization variants appear / Evidence-backed plausible | Plan exact-text policy | Plan validation tests; no Provider policy duplication | Pass |
| Narrative format changes / Evidence-backed plausible | GitHub Plan Representation | GitHub-planning package and its tests | Pass; no format strategy abstraction is added |
| GitHub API identifier contract changes / Evidence-backed plausible | GitHub Planning Provider boundary | Request values/references and Provider tests only | Pass |
| Observation, update, or deletion is added / Evidence-backed plausible | New target-specific responsibilities | Explicit future design Change | Pass; not forced into creation values |
| Request execution, retry, or authorization is added / Evidence-backed plausible | Future Change Target and execution boundaries | New contracts and integration tests | Pass; Request Plan remains a transient input |
| Another Provider is added / Evidence-backed plausible | New Provider Context Module | New Provider mapping; Plan remains stable | Pass; no generic Provider abstraction is added now |

### 3.3 Responsibility Assignment

| Responsibility / decision | Owner | Information and authority | Invariant protected | Not owner / reason |
|---|---|---|---|---|
| Validate local Plan text and date | Corresponding Plan value | Raw Provider-independent input | Invalid local values do not enter Plan | GitHub mapping lacks authority over Plan meaning |
| Validate aggregate cardinality and uniqueness | Plan | All Plan elements | One coherent valid Plan | Caller would duplicate invariants |
| Report whether a Plan value is valid | Plan | Private construction state owned by Plan | Zero or invalid Plan values do not cross Package boundaries as valid Plans | Provider mapping would reconstruct and duplicate Plan rules |
| Select representation | Consumer input represented by GitHub Plan Representation | Explicit representation value | No inference or ambiguous mapping | Plan has no GitHub knowledge |
| Encode narrative and target date for GitHub | GitHub Plan Representation | Valid Plan and GitHub contract | Deterministic Provider representation | Plan must remain Provider-independent |
| Establish operation order and symbolic references | Creation Request Plan | All request templates and result kinds | Closed, topological, type-compatible graph | Individual request lacks graph-wide information |
| Define request and response-result semantics | GitHub Planning Provider boundary | GitHub API contract | `number` and `id` are not confused | Core has no Provider authority |
| Own native resource state and assigned identifiers | GitHub | GitHub API and repository state | External authority and lifecycle | Arcloom is stateless and non-authoritative |

#### SOLID and Procedural Risk

| Principle | Risk | Mitigation |
|---|---|---|
| SRP | One planner function accumulates Plan validation, representation constraints, Markdown, graph, and API decisions | Plan reports its own validity; each GitHub representation owns its constraints and narrative completion; private Request Plan construction owns positions and typed-reference closure |
| OCP | Premature generic Provider or formatter strategies | Implement only the two committed GitHub representations with one explicit enum |
| LSP | Dry-run is disguised as an executor implementation | No execution interface exists in this Change |
| ISP | Broad GitHub client interface appears before a consumer | No client Port or network client is introduced |
| DIP | GitHub request DTOs leak into Plan | Only `githubplanning` depends on `plan`; `plan` has no outward dependency |

### 3.4 Package Design

| Package | Responsibility | Carries | Does not carry |
|---|---|---|---|
| `plan` | Implement Provider-independent Plan meaning and invariants | Goal, acceptance condition, Task, target date, Plan validation, validity query, and immutable access | Milestone, GitHub types, representation, request planning, external authority |
| `githubplanning` | Implement the concrete GitHub Planning Provider's Host-facing creation dry-run | Repository target, explicit representation, canonical narrative, request templates, typed symbolic references, Request Plan invariants | Plan meaning, consumer Ports, authorization, observation, execution, tokens, persistence |

| Boundary | Hidden detail | Dependency direction |
|---|---|---|
| `plan` | Text/date validation and aggregate storage | Standard library only |
| `githubplanning` | GitHub request fields, narrative format, operation graph, `number`/`id` semantics | The concrete Provider package depends inward on public `plan`; only an Arcloom Host may consume its Provider-specific preview |

No third package, generic Provider interface, formatter strategy, executor, repository, mapper, or factory is justified by the accepted behavior.

Within `githubplanning`, two private mapping functions own the Milestone and Issue representation rules. The Issue mapping owns its 100-Task constraint and target-date narrative. A shared private narrative function owns only Goal and acceptance-condition Markdown. A private Request Plan construction value allocates request positions and produces typed references as requests are added, making dangling, forward, and result-kind-incompatible references unconstructible through the mapping code. It is an implementation representation, not a public Builder Concept or extension point.

### 3.5 Architecture Boundary Gate

| Boundary candidate | Consumer / evidence | Owner | Constraint | Dependency | Simpler alternative | Decision |
|---|---|---|---|---|---|---|
| `plan` | Controllers, Change targets, and this Provider mapping require common Plan meaning | Plan | Provider independence and Plan invariants | Outer packages depend inward | GitHub DTO as Plan | Accept |
| `githubplanning` | An Arcloom Host requires the explicitly requested GitHub preview; Provider conformance tests protect it | Planning Provider Module | External API volatility and identifier semantics | Concrete Provider depends inward on `plan`; no Core or Module Port depends outward | Put GitHub fields in `plan` | Accept under the Host-facing preview rule in `ARCHITECTURE.md` |
| GitHub client Port | No execution consumer in scope | N/A | No current constraint | N/A | Request values only | Reject as premature |
| Generic Provider mapping Port | Only GitHub request output exists | N/A | No current substitutability | N/A | Concrete Provider package | Reject as premature |

| Scenario / confidence | Primary boundary | Expected propagation | Unexplained impact | Verdict |
|---|---|---|---|---|
| GitHub API contract changes / Evidence-backed plausible | `githubplanning` | GitHub request values and tests | None | Pass |
| Another Provider is added / Evidence-backed plausible | New Provider boundary | New package and composition | None in `plan` | Pass |
| Network execution is added / Evidence-backed plausible | Future consumer-owned application Port | Provider executor and integration tests | Not forced into current packages | Pass |
| Storage or resume is added / Evidence-backed plausible | External facts and future execution design | Explicit new contracts; no Arcloom authoritative store | None | Pass |

### 3.6 Interface Design

The following blocks are proposed Go public contracts, not existing implementation.

| Public contract | Consumer | Owner | Result and error contract | Hidden detail / protected constraint |
|---|---|---|---|---|
| Plan value constructors, accessors, and `IsValid` query | Reconciliation Modules, Plan Change Target Modules, and concrete Host-facing Provider previews | Plan | Success returns a valid immutable value; failure returns an invalid zero value and typed `plan.ValidationError`; `IsValid` reports constructed versus zero/invalid Plan without exposing individual rules | Validation, construction state, storage, and defensive copying protect Plan invariants without Provider-side reconstruction |
| `NewRepository` | An Arcloom Host configuring the GitHub Provider preview | GitHub Planning Provider boundary | Success returns a locally valid target; failure returns typed `githubplanning.ValidationError` | GitHub target syntax is kept out of Core contracts |
| `NewCreationRequestPlan` | An Arcloom Host inspecting the explicitly selected GitHub preview | GitHub Planning Provider boundary | Success returns a closed immutable request graph; failure returns a typed input error and invalid zero plan; no side effect | GitHub API fields and `number`/`id` result semantics stay in the concrete Provider boundary |
| Request inspection values | An Arcloom Host rendering the preview in transient memory | GitHub Planning Provider boundary | A sealed request sum contains exactly one valid request shape; zero values are invalid and all accessors are panic-free | Consumers cannot construct or mutate graph edges |

```go
package plan

type ViolationCode string

const (
	InvalidText                 ViolationCode = "invalid_text"
	MultilineName               ViolationCode = "multiline_name"
	InvalidTargetDate           ViolationCode = "invalid_target_date"
	MissingAcceptanceCondition ViolationCode = "missing_acceptance_condition"
	DuplicateAcceptanceCondition ViolationCode = "duplicate_acceptance_condition"
	DuplicateTask               ViolationCode = "duplicate_task"
)

type ElementKind uint8
const (
	PlanNameElement ElementKind = iota + 1
	GoalElement
	AcceptanceConditionElement
	TaskElement
	TargetDateElement
)

type ValidationError struct { /* private */ }
func (e *ValidationError) Error() string
func (e *ValidationError) Code() ViolationCode
func (e *ValidationError) Element() ElementKind
func (e *ValidationError) Index() (int, bool)

type Goal struct { /* private */ }
func NewGoal(text string) (Goal, error)
func (g Goal) Text() string

type AcceptanceCondition struct { /* private */ }
func NewAcceptanceCondition(statement string) (AcceptanceCondition, error)
func (c AcceptanceCondition) Statement() string

type Task struct { /* private */ }
func NewTask(name string) (Task, error)
func (t Task) Name() string

type TargetDate struct { /* private */ }
func ParseTargetDate(value string) (TargetDate, error)
func (d TargetDate) String() string

type Plan struct { /* private */ }
func New(
	name string,
	goal Goal,
	conditions []AcceptanceCondition,
	tasks []Task,
	targetDate *TargetDate,
) (Plan, error)
func (p Plan) Name() string
func (p Plan) Goal() Goal
func (p Plan) AcceptanceConditions() []AcceptanceCondition
func (p Plan) Tasks() []Task
func (p Plan) TargetDate() (TargetDate, bool)
func (p Plan) IsValid() bool
```

```go
package githubplanning

type ViolationCode string
const (
	InvalidRepository     ViolationCode = "invalid_repository"
	InvalidRepresentation ViolationCode = "invalid_representation"
	InvalidPlan           ViolationCode = "invalid_plan"
	UnsupportedRepresentation ViolationCode = "unsupported_representation"
)

type Field uint8
const (
	RepositoryOwnerField Field = iota + 1
	RepositoryNameField
	RepresentationField
	PlanField
)

type ValidationError struct { /* private */ }
func (e *ValidationError) Error() string
func (e *ValidationError) Code() ViolationCode
func (e *ValidationError) Field() Field

type Repository struct { /* private */ }
func NewRepository(owner, name string) (Repository, error)
func (r Repository) Owner() string
func (r Repository) Name() string

type Representation uint8
const (
	MilestoneRepresentation Representation = iota + 1
	IssueRepresentation
)

const RESTAPIVersion = "2022-11-28"

type ResultKind uint8
const (
	MilestoneNumber ResultKind = iota + 1
	IssueNumber
	IssueID
)

type RequestPosition uint32

type ResultReference struct { /* private */ }
func (r ResultReference) Source() RequestPosition
func (r ResultReference) Kind() ResultKind

type CreateMilestoneRequest struct { /* private */ }
func (r CreateMilestoneRequest) Title() string
func (r CreateMilestoneRequest) Description() string
func (r CreateMilestoneRequest) DueOn() (string, bool)

type CreateIssueRequest struct { /* private */ }
func (r CreateIssueRequest) Title() string
func (r CreateIssueRequest) Body() (string, bool)
func (r CreateIssueRequest) Milestone() (ResultReference, bool)

type AddSubIssueRequest struct { /* private */ }
func (r AddSubIssueRequest) ParentIssueNumber() ResultReference
func (r AddSubIssueRequest) SubIssueID() ResultReference

// Request is a sealed sum. Its concrete values are CreateMilestoneRequest,
// CreateIssueRequest, and AddSubIssueRequest. Consumers inspect it with a type switch.
type Request interface { /* unexported marker */ }

type RequestPlan struct { /* private */ }
func NewCreationRequestPlan(
	repository Repository,
	representation Representation,
	value plan.Plan,
) (RequestPlan, error)
func (p RequestPlan) Repository() Repository
func (p RequestPlan) Representation() Representation
func (p RequestPlan) APIVersion() string
func (p RequestPlan) Requests() []Request
```

`Plan.IsValid` is the minimum cross-Package query required because Go permits callers to construct a zero value even though Plan fields are private. It does not expose or repeat individual Plan rules. `NewCreationRequestPlan` is pure and has no side effect. It asks Plan for validity, delegates representation-specific rules to the selected private mapping, and delegates position/reference creation to private Request Plan construction. Request positions are zero-based positions in the returned topological list. Public constructors do not permit callers to assemble a Request, ResultReference, or RequestPlan independently. This keeps dangling, forward, and result-kind-incompatible references outside the public state space.

Every exported zero value is invalid, but every accessor is panic-free. A zero Plan supplied to `NewCreationRequestPlan` produces `InvalidPlan` with `PlanField` and no invented cause. `errors.As` identifies both Package validation-error types independently. Constructors return the zero result value on error. `ElementKind`, optional collection index, and `Field` locate the rejected input without exposing Provider error text. No ordering among simultaneous independent violations is public behavior. Request values never duplicate Repository or representation; their Request Plan owns both.

The proposed usage below is illustrative and not yet executable implementation.

```go
goal, err := plan.NewGoal("Ship the first planning slice")
if err != nil { return err }
condition, err := plan.NewAcceptanceCondition("The GitHub request preview is inspectable")
if err != nil { return err }
value, err := plan.New("M0", goal, []plan.AcceptanceCondition{condition}, nil, nil)
if err != nil { return err }

repository, err := githubplanning.NewRepository("kotokumu", "arcloom")
if err != nil {
	var validation *githubplanning.ValidationError
	if errors.As(err, &validation) {
		return fmt.Errorf("invalid GitHub preview input %v: %w", validation.Field(), err)
	}
	return err
}
preview, err := githubplanning.NewCreationRequestPlan(
	repository,
	githubplanning.MilestoneRepresentation,
	value,
)
if err != nil { return err }
if preview.APIVersion() != githubplanning.RESTAPIVersion {
	return fmt.Errorf("unexpected GitHub API version")
}

requests := preview.Requests() // One snapshot gives every Source position its context.
for position, request := range requests {
	switch request := request.(type) {
	case githubplanning.CreateMilestoneRequest:
		_ = request.Title()
	case githubplanning.CreateIssueRequest:
		if reference, ok := request.Milestone(); ok {
			if err := inspectReference(reference, githubplanning.MilestoneNumber, githubplanning.RequestPosition(position), requests); err != nil {
				return err
			}
		}
	case githubplanning.AddSubIssueRequest:
		if err := inspectReference(request.ParentIssueNumber(), githubplanning.IssueNumber, githubplanning.RequestPosition(position), requests); err != nil {
			return err
		}
		if err := inspectReference(request.SubIssueID(), githubplanning.IssueID, githubplanning.RequestPosition(position), requests); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown GitHub preview request %T", request)
	}
}

func inspectReference(
	reference githubplanning.ResultReference,
	expected githubplanning.ResultKind,
	consumer githubplanning.RequestPosition,
	requests []githubplanning.Request,
) error {
	if reference.Source() >= consumer || int(reference.Source()) >= len(requests) {
		return fmt.Errorf("invalid result reference")
	}
	if reference.Kind() != expected {
		return fmt.Errorf("unexpected result kind")
	}
	source := requests[reference.Source()]
	switch reference.Kind() {
	case githubplanning.MilestoneNumber:
		if _, ok := source.(githubplanning.CreateMilestoneRequest); !ok { return fmt.Errorf("invalid milestone source") }
	case githubplanning.IssueNumber, githubplanning.IssueID:
		if _, ok := source.(githubplanning.CreateIssueRequest); !ok { return fmt.Errorf("invalid issue source") }
	default:
		return fmt.Errorf("unknown result kind")
	}
	return nil
}
```

### 3.7 Detailed Design Gate

| Implementation unit | Implements | Dependencies | Behavior / test | Migration impact |
|---|---|---|---|---|
| Plan local values | Goal, acceptance condition, Task, target date | Standard library | Local validation boundary tables | None |
| Plan aggregate value | Plan cardinality, uniqueness, validity, snapshots | Plan local values | Aggregate, validity, and defensive-copy tests | None |
| Repository and representation values | GitHub target and explicit mapping selection | Standard library | Target/enum validation tests | None |
| GitHub request values | Provider request input and result-reference semantics | Repository value | Public accessor and zero-value tests | None |
| Request Plan value and private construction | Topological graph closure, position allocation, typed-reference issuance, and deterministic mapping | `plan`, GitHub request values | Both representation matrices and public reference-integrity property | None |
| Canonical narrative functions | Shared Goal/acceptance-condition text plus representation-specific completion | `plan` accessors | Exact output tests through public requests | None |
| Private representation functions | One committed GitHub representation rule each, including Issue-only constraints | Plan accessors and Request Plan construction | Representation-specific request and limit tests | None |

| Implementation unit | Simplest representation | Need | Rejected alternative | Verdict |
|---|---|---|---|---|
| Goal, condition, Task, date | Immutable values | Local meaning and invariant | Primitive strings at every call site | Pass |
| Plan | Immutable aggregate value | Aggregate invariants and defensive ownership | Public data-only struct | Pass |
| Representation | Enum | Exactly two committed choices and invalid zero | Strategy interface | Pass |
| Request | Sealed sum value | Three request shapes in one immutable ordered collection | Broad client interface or `map[string]any` | Pass |
| ResultReference | Typed value | Provider-assigned result kind and backward-reference invariant | Integer/string placeholder | Pass |
| Request Plan | Immutable aggregate value | Graph-wide validity and deterministic ordering | Bare request slice | Pass |
| Private Request Plan construction | Private stateful value within one mapping call | Owns ordinal allocation and typed-reference closure while constructing the immutable Request Plan | Public Builder, post-hoc graph validator, or repeated ordinal arithmetic in mappings | Pass |

Construction and the post-review correction are approved by the user after the applicable design findings were presented.

### 3.8 Construction Log

| Slice | Initial state | Smallest passing change | Refactor / retained evidence |
|---|---|---|---|
| Plan local values and aggregate | Public boundary tables failed because the values and invariants did not exist | Added immutable values, typed validation results, aggregate construction, and defensive accessors | Consolidated text and aggregate validation under Plan-owned rules |
| GitHub Milestone representation | Public request assertions failed because no Provider request model existed | Added Repository, representation, request values, and Milestone mapping | Kept GitHub semantics in `githubplanning` and Plan semantics in `plan` |
| GitHub Issue representation | Parent/child request and boundary tests failed | Added Issue mapping with typed result references and the 100-Task boundary | Preserved deterministic order and public reference-integrity properties |
| Plan validity correction | The new public test failed because `Plan.IsValid` was absent | Added construction-owned validity and made the zero Plan invalid | Removed Provider reconstruction of Plan invariants |
| Request graph correction | Existing public graph properties passed while private mappings still calculated raw positions and reinterpreted them afterward | Added private construction that allocates positions and issues typed result references | Removed post-hoc graph validation and moved Issue-only rules to the Issue mapping |
| Test contract correction | The passing baseline relied on private-layout and constructor-derived expectations in several cases | Re-authored the affected checks against literal public observations and complete typed error results | Retained table-driven boundaries, race execution, and defensive-copy properties |

---

## 4. Design Decisions

### 4.1 Model Milestone as an external representation

- Adopted: Plan contains no Milestone; GitHub Milestone is one representation selected at the Provider boundary.
- Rejected: Plan owns Milestones. This makes a Provider-native grouping and lifecycle part of every development method.
- Rejected: Plan is a subtype of Milestone. Their meanings and ownership differ.

### 4.2 Return a dependency-aware request plan

- Adopted: Requests retain typed symbolic references to results of earlier requests.
- Rejected: Generate fake GitHub IDs. Fake values are not Provider facts and can conceal `number`/`id` mistakes.
- Rejected: Return fully concrete HTTP requests. Later URLs and bodies cannot be known until GitHub assigns identifiers.

### 4.3 Use one concrete GitHub package

- Adopted: A concrete package contains both committed representation rules and exposes an enum.
- Rejected: Provider registry, generic mapping Port, formatter strategy, or one class per representation. No second Provider or formatter consumer exists.

### 4.4 Do not define an executor or client Port

- Adopted: The dry-run stops at request values conforming to known client inputs.
- Rejected: A no-op client implementing an execution contract. It would violate substitutability by claiming application without applying a Change.
- Rejected: A network client abstraction. No execution consumer exists in this Change.

### 4.5 Encode calendar dates canonically

- Adopted: Target Date remains `YYYY-MM-DD`; Milestone `due_on` uses the deterministic encoding `YYYY-MM-DDT00:00:00Z`.
- Trade-off: The request is an encoding of a calendar target, not a time-zone-aware end-of-day guarantee.

### 4.6 Let Plan own its validity

- Adopted: `Plan.IsValid()` reports whether a Plan value was successfully constructed. Provider packages use the query and translate an invalid result at their own boundary.
- Rejected: Provider packages reconstruct Plan validity from public fields. This duplicates policy and makes Plan evolution propagate to every Provider.
- Rejected: A generic Validator interface. Plan is the sole owner, and no substitution boundary exists.

### 4.7 Construct request graphs by invariant

- Adopted: Private Request Plan construction allocates positions and returns operation-specific typed references while requests are added.
- Rejected: Representation mappings calculate ordinals independently and a later validator reinterprets the same graph. This duplicates reference meaning and can misattribute an internal defect to caller input.
- Rejected: A public Builder. Consumers inspect a completed preview and have no need or authority to assemble its graph.

---

## 5. Impact, Migration, and Rollback

| Area | Impact |
|---|---|
| Product document | Removes Milestone from the provider-independent Plan capability description |
| Architecture document | Removes Milestone ownership from Plan and Plan Controller while retaining it as a Planning Context representation |
| Existing Go behavior | None; only the build-check package exists |
| Existing OpenSpec change | `reconcile-plan-representation` remains unmodified and is not implemented by this Change |
| External systems | None; no request is sent |

Rollback consists of removing the new unarchived Change and its new Go packages. No durable or external state requires migration or recovery.
