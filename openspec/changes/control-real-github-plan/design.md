# Control a Real GitHub Plan

## 0. Document Scope

| Information | Governing document |
|---|---|
| Product value, scope, capabilities, and principles | \`PRODUCT.md\` |
| Component responsibilities, external Context ownership, and permitted dependencies | \`ARCHITECTURE.md\` |
| Observable behavior and acceptance conditions | Specs under this OpenSpec change |
| Concepts, Package boundaries, public contracts, verification design, and design choices for this Change | This DesignDoc |
| Implementation order and completion status | \`tasks.md\` |

## 1. Purpose / Non-Goals

### Purpose

- Expose the independent contracts needed for a Host to observe one real GitHub Milestone Plan, obtain a Codex Plan Control assessment, authorize an exact Plan revision, ask an external Actor to apply it, and observe the authoritative result later.
- Preserve responsibility ownership so no Arcloom Component becomes a fixed workflow, a GitHub mutation engine, or an authoritative state store.
- Make one real Plan completion demonstrable with disposable verification evidence.

### Non-Goals

- Direct creation or mutation of GitHub Milestones or Issues by an Arcloom Provider.
- A long-running controller, workflow, scheduler, retry coordinator, or durable loop history.
- Task execution, Delivery Acceptance, or treating Task completion as Goal completion.
- A product-wide Observation schema or a common Provider interface.
- Mutable Issue-as-Plan, GitHub Projects, Linear, or another planning Provider.
- Stabilizing the experimental \`Change\` concept.

## 2. Behavior Design

### 2.1 Functional Requirements

The capability specs under this Change are normative. The following table assigns their behavior to design owners without restating individual scenarios.

| ID | Function | Observable rule | Decision owner |
|---|---|---|---|
| FR-1 | Plan snapshot | One fresh target-bound GitHub observation yields a current Plan only from coherent, complete, valid Plan facts. | Plan Snapshot and Plan |
| FR-2 | Progress evidence | The same observation yields provider-independent representation and member progress with explicit membership completeness. | Plan Representation Progress |
| FR-3 | Authorization | One non-empty Policy evaluates the exact consumer-established subject using deny-overrides, all-permit, otherwise-undecidable semantics. | Authorization Policy and Decision |
| FR-4 | Plan revision | One target, current Plan, and unequal proposed Plan retain one exact immutable revision meaning. | Plan Revision |
| FR-5 | Application request | Only a revision Authorized in the same invocation can enter a possibly-sent Application Attempt, and that Attempt is never retried. | Plan Application Attempt |
| FR-6 | Request result | Authorization denial or uncertainty and each degree of request-receipt certainty remain distinct and never establish external state. | Plan Application Result |
| FR-7 | Codex assessment | One disposable read-only Codex interaction translates to the existing Plan Control Assessor response or fails closed. | Codex Plan Control adapter and Plan Control |
| FR-8 | Re-observation | A later fresh snapshot alone establishes the next current Plan; correspondence does not by itself prove causality. | Plan Snapshot and disposable proof context |

### 2.2 Non-Functional Requirements

| Quality | Target | Verification |
|---|---|---|
| Race safety | Supported concurrent calls share no per-invocation state. | \`go test -race ./...\` plus concurrent contract tests |
| Cancellation | Every external boundary returns the caller context error when cancellation occurs before a result is established. | Boundary cancellation tests |
| Bounded Codex shutdown | A cancelled assessment returns no later than the configured finite grace period plus test scheduling tolerance. | Helper-process protocol test with a non-cooperative child |
| Dependency isolation | Provider-independent Packages import no GitHub or Codex Package and no Provider DTO or error crosses their public contracts. | Import review and black-box contract tests |
| Statelessness | No Product Package persists Plan, progress, Assessment, Authorization, request, acknowledgement, session, or loop history. | Design and code review; live proof starts from fresh external facts |

### 2.3 Change Acceptance Criteria

These criteria verify this Change without introducing a Product workflow or durable proof state.

| ID | Acceptance criterion |
|---|---|
| CA-1 | The proof operator selects one real GitHub Milestone target and records an initial current Snapshot whose Plan and representation progress were established from that target. |
| CA-2 | Codex assesses the exact initial Plan with caller-owned observation material and returns a valid Revise assessment containing one exact proposed Plan. |
| CA-3 | The exact stable target reference, current Plan, and proposed Plan form one Revision; current externally authoritative authorization evidence bound to that Revision establishes Authorized; and one external Actor interaction yields ReceiptAcknowledged. Any other receipt outcome leaves this proof incomplete. |
| CA-4 | A later current Snapshot independently establishes the external Plan after the Actor interaction and is compared with the proposed Plan without claiming causality from correspondence. |
| CA-5 | The proof record names the external facts used as Goal and acceptance-condition evidence, and Codex returns Complete for the later current Plan and those facts. |
| CA-6 | The record contains references to the selected target, both Snapshots, both assessments, Revision, the subject-bound externally authoritative authorization evidence, and ReceiptAcknowledged evidence; it is disposable, operator-owned, and establishes neither authoritative external state nor a mandatory operation order. |

## 3. Structural Design

### 3.1 Conceptual Model

| Concept | Meaning | Identity | Rule / invariant | Authority or lifecycle owner |
|---|---|---|---|---|
| Plan | Immutable provider-independent planning intent. | Its complete semantic value. | Name, Goal, conditions, Tasks, and optional date satisfy Plan invariants. | Plan owns meaning; no Arcloom store owns an authoritative instance. |
| External Plan Target Reference | Stable provider-independent reference to the exact externally owned representation to which a revision refers. | Non-empty Context and target identity strings. | The immutable reference identifies but does not contain or establish target state. | Plan Application Request Module owns value validity; Planning Context owns target lifecycle. |
| Plan Representation Scheme | Correspondence between one native representation and Plan/progress meaning. | Selected representation kind. | Root role, field sources, Task membership, and native-state mapping stay coherent. | Planning Provider Module. |
| Versioned Plan Representation Payload Format | Recognized envelope and members embedded in a native representation. | Format version. | A version is decoded according to its own complete contract; unsupported versions are unavailable. | Planning Provider Module. |
| Plan Snapshot | One disposable coherent result from one fresh observation. | One target-bound invocation. | A current Plan is present only when required Plan facts are complete and valid. | Plan Snapshot owns result invariants; external facts remain Planning Context-owned. |
| Plan Representation Progress | Provider-independent progress facts for one snapshot. | Overall representation state plus ordered observed members; member names may repeat. | State is Open, Closed, or Unknown; membership is Complete or Incomplete; no state establishes Plan completion. | Disposable snapshot result. |
| Assessment Observation Material | Caller-selected facts supplied for one assessment. | One assessment invocation. | It has no universal Arcloom vocabulary and remains immutable during assessment. | Caller and external fact owners. |
| Authorization Policy | One non-empty set of typed Rules. | Its immutable Rule set. | Empty or invalid Policy cannot authorize. | Consumer supplies it; Authorization owns aggregation semantics. |
| Authorization Rule | One cohesive interpretation of current authorization facts for an exact subject. | Rule implementation within a Policy. | It yields Permit, Deny, Unknown, or failure for that subject. | Rule owns interpretation; external Context owns facts. |
| Authorization Decision | Recalculable aggregate verdict produced by a Policy. | Authorized, Denied, or Undecidable. | Any Deny is Denied; all Permit is Authorized; otherwise Undecidable. | Authorization owns verdict semantics. |
| Authorization Evaluation | Immutable association between one exact consumer-established subject and its aggregate Decision. | One Policy evaluation invocation and subject. | The subject and reachable state remain semantically immutable for the Evaluation's complete lifetime, and the Decision cannot be reused without that association. | Disposable Authorization result. |
| Plan Revision | Exact proposed alteration of one current Plan for one stable target reference. | Target reference, current Plan, and proposed Plan together. | Both Plans are valid and unequal; target/current provenance is a Host precondition. | Disposable application input. |
| Plan Application Request | Passive external instruction concerning one exact revision. | Its revision. | It contains no authorization or transmission state and cannot establish target state. | Plan Application Request Module. |
| Plan Application Attempt | Invocation-local safety state for authorizing and possibly transmitting one request. | One invocation and revision. | AuthorizationDenied and AuthorizationUndecidable are terminal without transmission; once transmission may have begun, the Attempt never returns to a sendable state or retries. | Disposable Plan Application Request Module state. |
| Request Receipt Evidence | External evidence about whether the exact request was received. | Actor response for one request. | ReceiptAcknowledged, ReceiptRefused, KnownNotReceived, and ReceiptUncertain describe only request receipt, never mutation or completion. | External Actor supplies it; Plan Application Request Module validates its meaning. |
| Plan Application Result | Provider-independent classification of one Application Attempt. | One application-request invocation. | AuthorizationDenied, AuthorizationUndecidable, and the four receipt-evidence states remain distinct from target state. | Plan Application Request Module. |
| Plan Control Assessment | Existing immutable external-AI judgment for one current Plan. | Exact assessed Plan and one outcome. | Revise alone contains one valid unequal proposed Plan. | Plan Control owns result invariants; external AI owns judgment. |
| Disposable Codex Interaction | One provider-specific assessment exchange. | One process, connection, thread, and turn. | No prior session is required; one final structured response is admissible; cancellation is call-local. | Codex Provider adapter. |
| Verification Evidence Record | Passive references proving that this Change's acceptance criteria were exercised for one real target. | One operator-selected proof run. | It records exact inputs and externally owned evidence without asserting authority, causality, or a reusable workflow. | Proof operator; discarded after verification. |

The proof composition is not a Product Component. Its Verification Evidence Record is a disposable verification artifact, not authoritative Product state or a repeated lifecycle owner.

### 3.2 Responsibility Assignment

| Responsibility / decision | Owner | Information and authority used | Invariant protected | Not owner / reason |
|---|---|---|---|---|
| Validate Plan structure and semantic equality | Plan | Complete Plan values | Only valid immutable Plans participate in revisions and assessments. | Plan Control and application do not duplicate Plan semantics. |
| Map GitHub Milestone facts to Plan meaning | GitHub Plan Representation Scheme | GitHub root, payload, Issue membership, native state | One representation has one coherent mapping. | Plan Snapshot owns result coherence, not GitHub mapping. |
| Recognize and decode \`arcloom-plan\` versions | Versioned payload format in \`githubplan\` | Payload bytes and declared version | Versions coexist without partial interpretation. | HTTP access and Plan do not own serialization. |
| Establish current Plan eligibility | Plan Snapshot | One Scheme projection plus Plan invariants | Incomplete or invalid required facts never expose a current Plan. | Host does not assemble validity procedurally. |
| Classify representation progress completeness and states | Plan Representation Progress | Coherent mapped progress facts | Unknown state differs from incomplete membership; repeated member names remain observable; no state establishes Complete. | Plan does not own actual progress. |
| Interpret one authorization fact | Authorization Rule | Exact subject and current external facts | Failure or unavailable fact never contributes Permit. | Policy aggregates but does not acquire facts. |
| Aggregate Rule conclusions and bind the result | Authorization Policy and Evaluation | Rule conclusions for one exact subject | Deny-overrides/all-permit semantics have one owner and the Decision remains associated with its subject. | Each consumer does not reimplement policy. |
| Preserve target/current/proposed identity | Plan Revision | Stable target reference and two Plans | Authorization and Actor request concern the same revision. | Assessment has no target; Authorization does not inspect it. |
| Protect one external application attempt | Plan Application Attempt | Revision, current Policy, and Request Receipt Evidence | No request without current Authorization; once receipt is possible, the attempt is never retried. | Actor owns action-time mutation; Authorization owns only permission. |
| Perform action-time conflict judgment and GitHub mutation | External Actor | Current native state and permissions | Native safety and resulting state stay outside Arcloom. | GitHub Provider is read-only in this Change. |
| Translate Codex protocol | \`codexplancontrol\` | Host configuration and experimental app-server protocol | Provider details cannot weaken Plan Control response semantics. | Plan Control remains provider-independent. |
| Select observation vocabulary | Assessor consumer | Facts needed for its control judgment | No universal Observation model is invented. | Codex adapter only encodes supplied material. |
| Schedule or repeat operations | Host composition | Deployment-specific policy | No Arcloom Package owns a fixed Feedback Loop lifecycle. | No workflow, runtime, or orchestrator Package is introduced. |
| Preserve proof completeness without becoming authoritative | Verification Evidence Record | Native references and disposable contract results selected by the proof operator | Every CA-1 through CA-6 relation is inspectable without treating the record as external state or causality. | Product Components do not own the record. |

### 3.3 Package Design

| Package | Responsibilities implemented | Contracts published | Hidden implementation | Permitted dependencies |
|---|---|---|---|---|
| \`plan\` | Plan value, invariants, and semantic equality. | Existing Plan API plus equality. | Validation and comparison details. | Standard library only. |
| \`plansnapshot\` | Provider-independent Snapshot and Plan Representation Progress invariants; consumer-owned observation Port. | Snapshot, Progress Evidence, Task Progress, Observation Failure, Observer, Observe. | Defensive copies, membership matching, contract validation. | \`plan\`, standard library. |
| \`githubplan\` | GitHub target binding, fresh REST reads, Milestone Scheme, payload format, and Snapshot Observer implementation. | Milestone target and Snapshot Observer construction; existing APIs remain. | HTTP/JSON DTOs, pagination, native identity, API version, state mapping. | \`plan\`, \`plansnapshot\`, existing \`planrepresentation\`, standard library. |
| \`authorization\` | Generic Policy, Rule, subject-bound Evaluation, and aggregate Decision semantics. | Rule, Policy, Evaluation, Decision. | Rule iteration and failure localization. | Standard library only. |
| \`planapplication\` | Stable target reference and Plan Revision invariants, Authorization gating, Actor request Port, Application Attempt safety, and request-result classification. | Target Reference, Revision, Request, Actor, Receipt Evidence, Result, RequestApplication. | Exact one-attempt classification and defensive value handling. | \`plan\`, \`authorization\`, standard library. |
| \`plancontrol\` | Existing provider-independent Assessment semantics. | Existing Assessor and Assessment contracts. | Response validation. | \`plan\`, standard library. |
| \`codexplancontrol\` | Codex app-server implementation of the Assessor Port. | Validated Configuration, Observation Encoder, Assessor construction. | Process lifecycle, JSON-RPC, prompt/schema, correlation, DTOs, shutdown. | \`plan\`, \`plancontrol\`, standard library. |

No production Package is added for proof composition. Opt-in verification code composes public contracts as a test/Host concern and owns no domain decision.

| Source Package | Target Package | Public contract used | Dependency reason | Details that do not cross |
|---|---|---|---|---|
| \`plansnapshot\` | \`plan\` | Plan values and invariants | Snapshot may expose one valid current Plan. | Provider targets and progress do not enter Plan. |
| \`githubplan\` | \`plansnapshot\` | Observer and result values | GitHub implements the Plan Snapshot observation boundary. | HTTP DTOs, IDs, URLs, payload bytes, and errors. |
| \`githubplan\` | \`plan\` | Plan construction | Scheme reconstructs provider-independent Plan meaning. | GitHub state does not enter Plan. |
| \`planapplication\` | \`authorization\` | Typed Policy and subject-bound Evaluation | The exact Revision is authorized during the request invocation. | Authorization does not depend on Plan or target semantics. |
| \`planapplication\` | \`plan\` | Plan values and equality | Revision protects current/proposed invariants. | Actor protocol and receipt state do not enter Plan. |
| \`plancontrol\` | \`plan\` | Plan values | Existing Assessment concerns one exact current Plan. | AI protocol does not enter Plan Control. |
| \`codexplancontrol\` | \`plancontrol\` | Consumer-owned Assessor Port | Codex provides one concrete external-AI judgment. | JSON-RPC, process, model metadata, and raw output. |
| \`codexplancontrol\` | \`plan\` | Plan serialization input | The adapter supplies the exact current Plan to Codex. | Codex representation does not redefine Plan. |

### 3.4 Interface Design

#### Plan semantic equality

\`\`\`go
package plan

// Equal compares complete provider-independent Plan meaning.
// It performs no normalization and has no side effects.
func (p Plan) Equal(other Plan) bool
\`\`\`

#### Provider-independent Plan snapshot

\`\`\`go
package plansnapshot

type ProgressState uint8

const (
    Open ProgressState = iota + 1
    Closed
    Unknown
)

type TaskProgress struct { /* immutable name and state */ }
func NewTaskProgress(name string, state ProgressState) (TaskProgress, error)
func (p TaskProgress) Name() string
func (p TaskProgress) State() ProgressState

type ProgressEvidence struct { /* immutable overall and Task progress */ }
func CompleteProgress(overall ProgressState, tasks []TaskProgress) (ProgressEvidence, error)
func IncompleteProgress(overall ProgressState, tasks []TaskProgress) (ProgressEvidence, error)
func (p ProgressEvidence) OverallState() ProgressState
func (p ProgressEvidence) Tasks() []TaskProgress
func (p ProgressEvidence) MembershipComplete() bool

type Snapshot struct { /* optional current Plan and optional progress evidence */ }

// New requires a valid current Plan, complete ProgressEvidence, and exact
// correspondence between the Plan's Task names and progress members.
func New(current plan.Plan, progress ProgressEvidence) (Snapshot, error)

// WithoutCurrent preserves valid progress evidence when other Plan facts do
// not permit construction of a current Plan.
func WithoutCurrent(progress ProgressEvidence) (Snapshot, error)

func (s Snapshot) CurrentPlan() (plan.Plan, bool)
func (s Snapshot) Progress() (ProgressEvidence, bool)

type ObservationFailureCode uint8
const ObservationUnavailable ObservationFailureCode = 1

// ObservationFailure means a current observation could not establish a
// coherent Snapshot. It does not claim target absence. Its zero value is
// invalid and is rejected by Observe.
type ObservationFailure struct { /* stable code */ }
func UnavailableObservationFailure() ObservationFailure
func (f ObservationFailure) Error() string
func (f ObservationFailure) Code() ObservationFailureCode

// Observer is the consumer-owned Port for one immutable external target.
// Implementations map Provider failures to ObservationFailure and
// return only that stable failure or the supplied context error.
type Observer func(context.Context) (Snapshot, error)

// Observe validates context, Observer presence, and the returned Snapshot.
func Observe(context.Context, Observer) (Snapshot, error)
\`\`\`

#### GitHub Milestone Snapshot adapter

\`\`\`go
package githubplan

type MilestoneTarget struct { /* immutable Repository and ResourceNumber */ }
func NewMilestoneTarget(Repository, ResourceNumber) (MilestoneTarget, error)

// NewMilestoneSnapshotObserver validates local configuration without I/O.
// Every Observer call performs fresh GitHub reads and returns Provider-
// independent Snapshot values. Existing observation and preview APIs remain.
func NewMilestoneSnapshotObserver(
    client *http.Client,
    target MilestoneTarget,
) (plansnapshot.Observer, error)
\`\`\`

#### Generic Authorization

\`\`\`go
package authorization

type RuleConclusion uint8
const (
    Permit RuleConclusion = iota + 1
    Deny
    Unknown
)

type Decision uint8
const (
    Authorized Decision = iota + 1
    Denied
    Undecidable
)

// S and state reachable from it must remain semantically immutable for the
// complete lifetime of any returned Evaluation. A Rule does not mutate that
// state and is safe for concurrent Policy evaluations. A non-context error
// or invalid conclusion contributes Unknown.
type Rule[S any] func(context.Context, S) (RuleConclusion, error)

type Policy[S any] struct { /* immutable non-empty Rule set */ }
func NewPolicy[S any](rules ...Rule[S]) (Policy[S], error)

type Evaluation[S any] struct { /* exact subject and aggregate Decision */ }
func (e Evaluation[S]) Subject() S
func (e Evaluation[S]) Decision() Decision

// Evaluate supplies the exact subject to Rules and binds the Decision to that
// subject. A zero/invalid Policy produces Undecidable. Caller cancellation
// returns ctx.Err and no Evaluation.
func (p Policy[S]) Evaluate(context.Context, S) (Evaluation[S], error)
\`\`\`

#### Plan Application Request

\`\`\`go
package planapplication

// TargetReference is a stable immutable value, not target state.
type TargetReference struct { /* non-empty Context and identity */ }
func NewTargetReference(context, identity string) (TargetReference, error)
func (t TargetReference) Context() string
func (t TargetReference) Identity() string

// Revision preserves one stable target reference and valid unequal Plans.
// The Host owns the precondition that target and current came from one fresh
// Provider snapshot.
type Revision struct { /* target, current, proposed */ }
func NewRevision(target TargetReference, current, proposed plan.Plan) (Revision, error)
func (r Revision) Target() TargetReference
func (r Revision) Current() plan.Plan
func (r Revision) Proposed() plan.Plan
func (r Revision) Equal(other Revision) bool

type Request struct { /* exact immutable Revision */ }
func (r Request) Revision() Revision

type ReceiptKind uint8
const (
    ReceiptAcknowledged ReceiptKind = iota + 1
    ReceiptRefused
    KnownNotReceived
    ReceiptUncertain
)

// ReceiptEvidence may include an Actor-native reference as non-authoritative
// verification evidence. Its zero value means ReceiptUncertain.
type ReceiptEvidence struct { /* kind and optional native reference */ }
func AcknowledgedReceipt(reference string) ReceiptEvidence
func RefusedReceipt(reference string) ReceiptEvidence
func NotReceivedReceipt() ReceiptEvidence
func UncertainReceipt() ReceiptEvidence
func (e ReceiptEvidence) Kind() ReceiptKind
func (e ReceiptEvidence) Reference() (string, bool)

// Actor owns action-time interpretation, conflict handling, and mutation. It
// must not begin transmission when ctx is already cancelled. Cancellation
// observed before transmission returns KnownNotReceived; cancellation after
// transmission may have begun returns ReceiptUncertain unless stronger exact
// evidence exists. It returns request-receipt evidence, never target state.
type Actor func(context.Context, Request) ReceiptEvidence

// Result contains either a non-Authorized Decision or one ReceiptEvidence.
// Its zero value is invalid and no combination can represent target state.
type Result struct { /* authorization decision or receipt evidence */ }
func (r Result) AuthorizationDecision() (authorization.Decision, bool)
func (r Result) Receipt() (ReceiptEvidence, bool)

type FailureCode string
const (
    InvalidRevision FailureCode = "invalid_revision"
    InvalidActor FailureCode = "invalid_actor"
)

type Failure struct { /* stable code; zero value has no code */ }
func (f Failure) Error() string
func (f Failure) Code() FailureCode

// RequestApplication owns one invocation-local Application Attempt, evaluates
// Policy for revision, verifies the subject-bound Evaluation, and may invoke
// actor once. Entry cancellation wins before local validation; then Revision,
// Actor, and Policy are considered in that order. An invalid Policy yields an
// AuthorizationUndecidable Result. Invalid Revision or Actor returns Failure.
// Cancellation before Actor invocation returns ctx.Err. After invocation,
// established receipt evidence wins; otherwise cancellation or zero evidence
// yields ReceiptUncertain. No result establishes application or completion.
func RequestApplication(
    ctx context.Context,
    revision Revision,
    policy authorization.Policy[Revision],
    actor Actor,
) (Result, error)
\`\`\`

#### Codex Plan Control Assessor

\`\`\`go
package codexplancontrol

type Model struct { /* non-empty Codex model identifier */ }
func NewModel(value string) (Model, error)

type ReasoningEffort struct { /* validated supported effort */ }
func NewReasoningEffort(value string) (ReasoningEffort, error)

type WorkingDirectory struct { /* existing absolute directory */ }
func NewWorkingDirectory(value string) (WorkingDirectory, error)

type ShutdownGrace struct { /* positive duration up to 30 seconds */ }
func NewShutdownGrace(value time.Duration) (ShutdownGrace, error)

type Configuration struct { /* immutable cohesive values */ }
func NewConfiguration(
    model Model,
    reasoningEffort ReasoningEffort,
    workingDirectory WorkingDirectory,
    shutdownGrace ShutdownGrace,
) Configuration

// ObservationEncoder preserves caller vocabulary and must not mutate input.
type ObservationEncoder[O any] func(context.Context, O) (string, error)

// NewAssessor validates Configuration and encoder without starting Codex.
// The public contract always launches the trusted Host-installed codex
// app-server with read-only sandbox and no mutation approval; callers cannot
// select another executable or relax those constraints. Each invocation
// implements the existing Plan Control Assessor contract independently.
func NewAssessor[O any](
    configuration Configuration,
    encode ObservationEncoder[O],
) (plancontrol.Assessor[O], error)
\`\`\`

### 3.5 Boundary Decisions

| Boundary | Consumer and protected constraint | Hidden detail | Why an existing boundary is insufficient |
|---|---|---|---|
| Plan Snapshot Observer | Host/control composition needs one coherent current Plan and progress value from an external target. | Provider reads, identity, pagination, and representation mapping. | Existing Plan Representation Observer produces comparison evidence and cannot expose current Plan or progress. |
| Authorization Rule | Policy needs substitutable current external fact interpretations for an exact consumer-established subject. | Approval/permission Provider access and fact vocabulary. | A boolean or pre-established decision loses Rule failure, Unknown, and recalculation semantics. |
| Plan Application Actor | Application request needs an external owner for action-time mutation and receipt evidence. | Human/agent/system protocol and GitHub mutation. | Direct GitHub mutation lacks the required concurrency/idempotency guarantees. |
| Plan Control Assessor | Existing Plan Controller needs one external AI judgment. | Codex process and protocol. | Plan Control must remain independent of the experimental Provider. |
| Observation Encoder | Assessor consumer owns observation meaning while the Codex adapter needs serializable material. | Caller vocabulary. | A universal Observation model would couple unrelated evidence sources. |

#### Snapshot outcome classification

| Established current facts | Result | Current Plan | Representation progress |
|---|---|---|---|
| Local target is invalid | Construction failure before GitHub access | None | None |
| Current root cannot be established | ObservationFailure | None | None |
| Caller cancellation is observed before completion | Supplied context error | None | None |
| Root is coherent; payload is missing, unsupported, or unusable; membership is complete | Successful Snapshot | None | Complete and preserved |
| Root and membership are coherent; known Plan values violate Plan invariants | Successful Snapshot | None | Complete and preserved |
| Root is coherent; member collection is incomplete, later unavailable, or contradictory for one native member | Successful Snapshot | None | Incomplete and preserved |
| Root, valid Plan facts, and complete membership are coherent | Successful Snapshot | Exact current Plan | Complete and exactly corresponding |
| Complete membership contains distinct members with duplicate names | Successful Snapshot | None | Complete with repeated names preserved |

#### Plan Application precedence

| Condition established first | Result | Actor invocation |
|---|---|---|
| Entry cancellation | Supplied context error | None |
| Invalid Revision | InvalidRevision Failure | None |
| Missing Actor | InvalidActor Failure | None |
| Invalid Policy | AuthorizationUndecidable Result | None |
| Cancellation during Authorization | Supplied context error | None |
| Denied or Undecidable Evaluation bound to the exact Revision | Corresponding Authorization Result | None |
| Cancellation after Authorization but before Actor invocation | Supplied context error | None |
| Actor observes cancellation before beginning transmission | Result containing KnownNotReceived | Actor invoked once; no transmission |
| ReceiptAcknowledged, ReceiptRefused, or KnownNotReceived established by Actor | Result containing that exact ReceiptEvidence, even if cancellation is then observed | Once |
| ReceiptUncertain, zero evidence, or cancellation after possible receipt without other established evidence | Result containing ReceiptUncertain | Once; never retried |

### 3.6 Test Specification

#### Requirement Coverage

| Requirement / acceptance criterion | Observable behavior | Automated test or verification | Owner | Evidence |
|---|---|---|---|---|
| GHPS-1 through GHPS-6 | Valid facts produce exact Plan/progress; invalid, incomplete, and unavailable observations do not produce a current Plan. | Table-driven HTTP contract tests and race tests. | \`githubplan\`, \`plansnapshot\` | Passing Go tests. |
| AUTH-1 through AUTH-5 | Exact subject forwarding and aggregate decisions fail closed. | Table-driven unit tests, cancellation tests, import review. | \`authorization\` | Passing Go tests and dependency check. |
| PAR-1 through PAR-8, including PAR-2A | Only Authorized enters a possibly-sent Attempt; every receipt certainty maps exactly; no state claim. | Table-driven unit tests with the Actor Port only. | \`planapplication\` | Passing Go tests. |
| CPCA-1 through CPCA-7 | Valid protocol yields the existing Plan Control response; drift, cancellation, and concurrency fail safely. | Helper-process JSON-RPC contract tests; no live model. | \`codexplancontrol\` | Passing Go tests and bounded-time assertions. |
| Canonical contracts | Authorization is Change-independent; GitHub observation is read-only; no lifecycle owner or authoritative store exists. | Architecture, import, and source review. | Design reviewer | Review record. |
| CA-1 through CA-6 | Real observation, Revise, current Authorization, one Actor request, external reflection, exact re-observation, named completion evidence, and Complete are recorded. | Opt-in external verification; excluded from CI. | Host/proof operator | Disposable Verification Evidence Record with native references. |

#### Test Specifications

| Behavior | Given | When | Then | Level | Notes |
|---|---|---|---|---|---|
| Snapshot reconstructs current Plan | Valid Milestone payload and complete paginated Issues | Observe | Exact Plan and complete progress are returned. | HTTP contract | Assert no IDs/URLs/raw data in public values. |
| Snapshot rejects incomplete membership | Valid root and incomplete/contradictory pages | Observe | No current Plan; incomplete progress preserves coherent members. | HTTP contract | Unknown state alone remains valid. |
| Snapshot preserves duplicate names | Complete membership with two distinct native members sharing a name | Observe | No current Plan; progress preserves both members and complete membership. | HTTP contract | Duplicate name is a Plan violation, not incomplete observation. |
| Snapshot reports observation failure | Required current facts cannot be established | Observe | Stable provider-independent ObservationFailure; no Snapshot. | HTTP contract | No Provider error or authoritative-absence claim crosses. |
| Authorization deny-overrides | Mixed Permit, Deny, Unknown, and failures | Evaluate | Subject-bound Evaluation is Denied. | Unit | Rules use the exact subject value. |
| Authorization all-permit | Non-empty Rules all Permit | Evaluate | Subject-bound Evaluation is Authorized. | Unit | Zero Policy yields Undecidable. |
| Revision invariant | Valid/equal/invalid Plan combinations | Construct Revision | Only valid unequal pair succeeds. | Unit | Uses Plan-owned equality. |
| Application receipt mapping | Each Authorization decision and receipt-evidence kind | Request application | Exact authorization or receipt outcome and at most one Actor call. | Unit | Do not assert private call order beyond safety boundary. |
| Application cancellation | Cancellation occurs at each Plan Application precedence boundary | Request application | The exact context-error or receipt-evidence row is preserved. | Unit | No retry. |
| Codex structured output | Helper process emits supported handshake and one final response | Assess | Existing AssessorResponse is returned. | Process contract | No Provider data crosses. |
| Codex request meaning | Every Plan element and caller-owned observation value varies independently | Assess through a capturing helper | The helper receives the exact semantic Plan and observation material with existing Plan Control outcome meanings. | External-boundary contract | Do not assert exact prompt prose. |
| Codex protocol drift | Helper emits each completed-invalid or interaction-failure case | Assess | Failure follows the Codex failure-classification table. | Process contract | Tests consume only the required protocol subset. |
| Codex bounded cancellation | Helper ignores cooperative termination | Cancel one assessment | That assessment returns within grace and another concurrent assessment remains unaffected. | Process contract | Do not assert process count or identity. |

#### Invariant Tests

| Invariant | Example | Expected result |
|---|---|---|
| Snapshot Plan and progress have exact Task membership. | Plan Task \`A\`; progress only for \`B\`. | Snapshot construction fails. |
| Incomplete progress never accompanies an exposed current Plan. | Valid Plan with Incomplete progress. | Snapshot construction fails. |
| Authorized requires every Rule Permit. | Permit plus failure or Unknown. | Undecidable. |
| Revision preserves exact three-part identity. | Inspect target/current/proposed after construction. | Exact immutable values are returned. |
| Receipt evidence never establishes target state. | ReceiptAcknowledged with a native reference. | Result contains only that ReceiptEvidence; no Plan/state accessor exists. |
| Assessments are behaviorally isolated. | Two concurrent calls with distinct material and one cancellation. | Each result uses only its own material; the uncancelled call remains unaffected. |

#### Error and Edge Case Tests

| Case | Given | When | Then |
|---|---|---|---|
| Closed Milestone and mixed Tasks | Valid closed root and open/closed/unknown Task states | Observe | Current Plan remains valid; exact progress states are exposed. |
| Unsupported payload version | Known root with unknown version and complete membership | Observe | No current Plan; complete progress is preserved without Provider error leakage. |
| Invalid Policy/Rule | Empty Policy, nil Rule, invalid conclusion, Rule error | Evaluate | Subject-bound Evaluation is never Authorized; cancellation remains caller-owned. |
| Ambiguous Actor response | Zero evidence or possible receipt without certainty | Request application | Result contains ReceiptUncertain and no retry. |
| Actor acknowledgement differs from later state | ReceiptAcknowledged, later GitHub facts differ | Re-observe | Later snapshot is authoritative; no causality/application claim. |
| Encoder failure | Live context and encoder error | Assess | AI boundary failure. |
| Duplicate Codex final | Two final agent messages | Assess | AI contract failure. |

#### Codex failure classification

| External boundary result | Assessor result consumed by Plan Control | Final Plan Control meaning |
|---|---|---|
| Valid InsufficientInformation output | One valid AssessorResponse | Successful InsufficientInformation Assessment |
| Completed output is empty, malformed, ambiguous, conflicting, duplicate, or otherwise untranslatable | ErrUntranslatableAIResponse | AIContractFailure |
| Interaction fails before one translatable response while caller context remains active | Non-contract Assessor error | AIBoundaryFailure |
| Caller cancellation or deadline is observed before a valid Assessment is established | Supplied context error | Supplied context error |

Testability feedback:

- The exact target/current/proposed relation must be one Revision value; separate parameters would make mismatch tests procedural.
- Actor receipt uncertainty must be a typed reply; a plain \`error\` cannot distinguish known non-receipt from possible receipt.
- Snapshot and progress require named constructors so tests can express complete and incomplete evidence without boolean flags.
- No Repository, clock, workflow, or shared-session mock is required.

## 4. Design Decisions

### 4.1 External Actor owns GitHub mutation

- Adopted: Arcloom authorizes and sends one exact revision request to an external Actor. Fresh GitHub observation establishes the result.
- Rejected: Direct Milestone/Issue mutation from \`githubplan\`.
- Reason: GitHub documents no compare-and-swap, conditional unsafe request, idempotency key, or create-if-absent contract for these resources. The Actor can perform action-time conflict judgment; an Arcloom retry cannot.

### 4.2 Snapshot is separate from Plan Representation reconciliation

- Adopted: \`plansnapshot\` owns current-Plan and progress coherence; \`githubplan\` implements its Observer.
- Rejected: Expose internals of \`planrepresentation.Observation\` or make its Controller materialize the current Plan.
- Reason: Comparison with a caller-supplied expected Plan and establishment of a current Plan have different consumers, results, and failure meaning.

### 4.3 Representation and payload variation remain inside \`githubplan\`

- Adopted: Milestone Scheme and versioned payload format are cohesive private responsibilities in the concrete Provider Package.
- Rejected: Public Scheme registry, Factory, or Plugin interfaces.
- Reason: Milestone and Issue mapping plus payload v1 are evidenced variations; no current consumer requires a public substitution boundary.

### 4.4 Authorization is generic and subject-agnostic

- Adopted: Generic typed Rules and Policy aggregation; Plan Revision plays the Authorization subject role.
- Rejected: Change Authorization or a Plan-specific authorizer.
- Reason: Authorization decides whether one exact proposed external request may be issued. Its aggregation rules do not depend on Change, Plan, or target data.

### 4.5 Plan Revision, Request, and Attempt are separate concepts

- Adopted: Revision is the immutable target/current/proposed relationship. Request is a passive instruction carrying that Revision. Application Attempt owns invocation-local authorization and receipt certainty. Assessment remains an AI judgment.
- Rejected: Put target and request state into \`plancontrol.Assessment\`, or require \`Change\`.
- Reason: Proposals may come from outside Plan Control, Assessment has no target, and \`Change\` remains experimental.

### 4.6 Application Attempt and receipt certainty are result invariants

- Adopted: One invocation-local Application Attempt distinguishes authorization outcomes from explicit receipt, refusal, known non-receipt, and uncertainty. A passive Request contains no lifecycle state.
- Rejected: Boolean success or retry on an Actor error.
- Reason: A failed response after transmission may duplicate work if retried and never proves resulting GitHub state.

### 4.7 Codex process is disposable per assessment

- Adopted: One app-server process/connection/thread/turn per Assessor invocation with bounded cancellation.
- Rejected: Shared long-lived Codex thread or session pool.
- Reason: The app-server protocol is experimental; disposable ownership isolates protocol drift, output correlation, cancellation, and authoritative-state risk.

### 4.8 Observation vocabulary remains caller-owned

- Adopted: The caller supplies a typed encoder for its own material.
- Rejected: A universal \`Observation\` hierarchy in Plan Control or the Codex adapter.
- Reason: Progress, CI, quality, requirements, and telemetry evolve under different authorities and do not share one proven product taxonomy.

### 4.9 Real proof is composition, not a Product workflow

- Adopted: Opt-in verification invokes independent public contracts and records disposable native evidence.
- Rejected: A workflow, pipeline, controller runtime, or persisted loop execution.
- Reason: Delivery method and repeated lifecycle belong to the Host. The proof needs one concrete ordering but does not make it mandatory product behavior.

## Risks / Trade-offs

| Risk / trade-off | Mitigation |
|---|---|
| External Actor acknowledges receipt but applies a partial or different revision. | Treat acknowledgement only as receipt; compare a later fresh Plan exactly and require another control judgment for deviations. |
| Authorization facts change after decision. | Recalculate immediately before the one Actor request; the Actor still owns action-time conflict and permission checks. |
| Codex app-server protocol changes. | Isolate the consumed JSON-RPC subset, use structured output, and fail closed in process contract tests. |
| GitHub API version \`2022-11-28\` is retired. | Keep version and DTO handling inside \`githubplan\`; update its contract tests without changing provider-independent Packages. |
| A stable target reference can be mapped to the wrong native target by a Host or Actor. | Require non-empty Context and identity values, preserve them exactly through Revision and Request, and leave action-time native-target validation with the Actor. |
| Live proof depends on real credentials, exact approval, and Actor availability. | Keep it opt-in, require explicit Host inputs, and never infer approval from repository access. |
| Disposable process per assessment costs startup time. | Accept the cost for isolation; optimize only after measured need without changing the Assessor contract. |

## 5. Impact, Migration, and Rollback

### Impact

- Additive Packages: \`plansnapshot\`, \`authorization\`, \`planapplication\`, and \`codexplancontrol\`.
- Additive read-side API in \`githubplan\`; existing Plan Representation observation and creation-preview contracts remain.
- Additive semantic equality in \`plan\`; \`plancontrol\` delegates its existing comparison rule to that owner.
- \`PRODUCT.md\` and \`ARCHITECTURE.md\` replace Change-specific Authorization and the Plan Change Target with the generic Authorization and Plan Application Request boundaries.

### Migration

- No data migration exists because Arcloom owns no persistent state.
- Existing callers remain source compatible.
- New Hosts opt into each contract independently and retain target/provenance in their composition.

### Rollback

- Remove the additive Packages and GitHub Snapshot API, restore the previous canonical-document terminology, and retain the existing read-only observer, dry-run preview, and Plan Control APIs.
- No external data rollback is performed. Any GitHub change is owned and recorded by the external Actor and Planning Context.

## Open Questions

None. Provider-specific external Actor protocol and the real Plan selected for proof are Host inputs and do not alter these contracts.
