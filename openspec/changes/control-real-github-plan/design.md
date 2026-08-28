# Control a Real GitHub Plan

## 0. Document Scope

| Information | Governing document |
|---|---|
| Product value, scope, capabilities, and principles | \`PRODUCT.md\` |
| Component responsibilities, external Context ownership, and permitted dependencies | \`ARCHITECTURE.md\` |
| Capability boundary, concepts, relationships, and requirement candidates | \`model.md\` |
| Observable behavior and acceptance conditions | Specs under this OpenSpec change |
| Package boundaries, public contracts, verification design, and design choices for this Change | This DesignDoc |
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
- Implementing the Codex app-server Go SDK in this Change; this Change defines
  and consumes only the SDK contract required by Plan Control.

## 2. Behavior Design

### 2.1 Functional Requirements

The capability specs under this Change are normative. The following table assigns their behavior to design owners without restating individual scenarios.

| Requirement group | Function | Observable rule | Decision owner |
|---|---|---|---|
| \`github-plan-snapshot/*\` | Plan snapshot and progress | One fresh target-bound GitHub observation yields a current Plan only from coherent, complete, valid facts and preserves provider-independent progress otherwise. | GitHub Plan Snapshot and Plan |
| \`authorization/*\` | Authorization | One non-empty Policy evaluates the exact consumer-established subject using deny-overrides, all-permit, otherwise-undecidable semantics. | Authorization Policy and Evaluation |
| \`plan-application-request/exact-plan-revision\` | Plan revision | One target, current Plan, and unequal proposed Plan retain one exact immutable revision meaning. | Plan Revision |
| \`plan-application-request/current-authorization-required\` and \`at-most-one-transmission\` | Application request | Only a Revision Authorized in the same invocation can enter a possibly-sent Application Attempt, and that Attempt is never retried. | Plan Application Attempt |
| \`plan-application-request/request-interaction-result\` through \`stateless-provider-independent-meaning\` | Request result and re-observation | Authorization and receipt outcomes remain distinct; only later fresh observation establishes external Plan state. | Plan Application Result and GitHub Plan Snapshot |
| \`codex-plan-control-assessment/*\` | Codex assessment | One disposable read-only Codex interaction translates to the existing Plan Control Assessment or fails closed. | Codex Plan Control adapter and Plan Control |

### 2.2 Non-Functional Requirements

| Quality | Target | Verification |
|---|---|---|
| Race safety | Supported concurrent calls share no per-invocation state. | \`go test -race ./...\` plus concurrent contract tests |
| Cancellation | Every external boundary returns the caller context error when cancellation occurs before a result is established. | Boundary cancellation tests |
| Bounded Codex shutdown | A cancelled assessment returns no later than the SDK-configured finite grace period plus test scheduling tolerance. | Codex app-server Go SDK process contract test; the Plan Control adapter verifies cancellation propagation at its Client boundary. |
| Dependency isolation | Provider-independent Packages import no GitHub or Codex Package and no Provider DTO or error crosses their public contracts. | Import review and black-box contract tests |
| Statelessness | No Product Package persists Plan, progress, Assessment, Authorization, request, acknowledgement, session, or loop history. | Design and code review; live proof starts from fresh external facts |

### 2.3 Change Acceptance Criteria

These criteria verify this Change without introducing a Product workflow or durable proof state.

| ID | Acceptance criterion |
|---|---|
| CA-1 | The proof operator selects one real GitHub Milestone target and records an initial GitHub Plan Snapshot whose current Plan and representation progress were established from that target. |
| CA-2 | Codex assesses the exact initial Plan with caller-owned observation material and returns a valid Revise assessment containing one exact proposed Plan. |
| CA-3 | The exact stable target reference, current Plan, and proposed Plan form one Revision; current externally authoritative authorization evidence bound to that Revision establishes Authorized; and one external Actor interaction yields ReceiptAcknowledged. Any other receipt outcome leaves this proof incomplete. |
| CA-4 | A later GitHub Plan Snapshot independently establishes the current external Plan after the Actor interaction and is compared with the proposed Plan without claiming causality from correspondence. |
| CA-5 | The proof record names the external facts used as Goal and acceptance-condition evidence, and Codex returns Complete for the later current Plan and those facts. |
| CA-6 | The record contains references to the selected target, both GitHub Plan Snapshots, both assessments, Revision, the subject-bound externally authoritative authorization evidence, and ReceiptAcknowledged evidence; it is disposable, operator-owned, and establishes neither authoritative external state nor a mandatory operation order. |

## 3. Structural Design

### 3.1 Concept-to-design mapping

The specification concepts and invariants are defined in \`model.md\` and the delta specs. This section records only their physical realization and responsibility placement.

| Specification concept | Design representation | Responsibility / lifecycle placement |
|---|---|---|
| Plan | Existing immutable value in \`plan\`, extended with semantic equality | \`plan\` protects validity and equality; no Arcloom store owns an authoritative Plan. |
| GitHub Milestone Target, GitHub Plan Snapshot, Representation Progress | Provider-independent values and observation contract in \`plansnapshot\`; GitHub binding in \`githubplan\` | \`plansnapshot\` protects result coherence; \`githubplan\` owns provider mapping; the Planning Context owns external facts. |
| Authorization Subject, Policy, Rule Conclusion, Evaluation | Generic typed values and evaluation contract in \`authorization\` | \`authorization\` owns aggregation and subject binding; consumers and external Contexts own subject validity and evidence. |
| External Plan Target Reference, Plan Revision, Application Request, Request Receipt Evidence, Plan Application Result | Immutable values and application contract in \`planapplication\` | \`planapplication\` protects one attempt; the Actor owns receipt evidence and action-time mutation; the Planning Context owns target state. |
| Codex Assessment Interaction and Safe Host Configuration | Plan Control Assessor implementation in \`codexplancontrol\` and an SDK contract in \`codexappserver\` | \`codexplancontrol\` owns Plan assessment translation; the separately developed SDK owns app-server communication and interaction lifecycle; \`plancontrol\` retains provider-independent assessment meaning. |

The disposable proof composition is not a Product Component. Its evidence record is operator-owned verification material, not authoritative Product state or a repeated lifecycle owner.

### 3.2 Responsibility Assignment

| Responsibility / decision | Owner | Information and authority used | Invariant protected | Not owner / reason |
|---|---|---|---|---|
| Validate Plan structure and semantic equality | Plan | Complete Plan values | Only valid immutable Plans participate in revisions and assessments. | Plan Control and application do not duplicate Plan semantics. |
| Map GitHub Milestone facts to Plan meaning | GitHub Plan Representation Scheme | GitHub root, payload, Issue membership, native state | One representation has one coherent mapping. | GitHub Plan Snapshot owns result coherence, not GitHub mapping. |
| Recognize and decode \`arcloom-plan\` versions | Versioned payload format in \`githubplan\` | Payload bytes and declared version | Versions coexist without partial interpretation. | HTTP access and Plan do not own serialization. |
| Establish current Plan eligibility | GitHub Plan Snapshot | One Scheme projection plus Plan invariants | Incomplete or invalid required facts never expose a current Plan. | Host does not assemble validity procedurally. |
| Classify representation progress completeness and states | Plan Representation Progress | Coherent mapped progress facts | Unknown state differs from incomplete membership; repeated member names remain observable; no state establishes Complete. | Plan does not own actual progress. |
| Interpret one authorization fact | Authorization Rule | Exact subject and current external facts | Failure or unavailable fact never contributes Permit. | Policy aggregates but does not acquire facts. |
| Aggregate Rule conclusions and bind the result | Authorization Policy and Evaluation | Rule conclusions for one exact subject | Deny-overrides/all-permit semantics have one owner and the Decision remains associated with its subject. | Each consumer does not reimplement policy. |
| Preserve target/current/proposed identity | Plan Revision | Stable target reference and two Plans | Authorization and Actor request concern the same revision. | Assessment has no target; Authorization does not inspect it. |
| Protect one external application attempt | Plan Application Attempt | Revision, current Policy, and Request Receipt Evidence | No request without current Authorization; once receipt is possible, the attempt is never retried. | Actor owns action-time mutation; Authorization owns only permission. |
| Perform action-time conflict judgment and GitHub mutation | External Actor | Current native state and permissions | Native safety and resulting state stay outside Arcloom. | GitHub Provider is read-only in this Change. |
| Translate Plan assessment material and output | \`codexplancontrol\` | Exact Plan, caller-owned observation material, and one SDK-completed final output | Provider output cannot weaken Plan Control response semantics. | The SDK does not depend on Plan or Plan Control. |
| Complete one read-only Codex app-server turn | Codex app-server Go SDK | SDK configuration, app-server protocol, and process authority | Process, transport, protocol, and shutdown failures never become a completed turn. | \`codexplancontrol\` does not launch or stop processes and does not parse JSON-RPC. |
| Select observation vocabulary | Assessor consumer | Facts needed for its control judgment | No universal Observation model is invented. | Codex adapter only encodes supplied material. |
| Schedule or repeat operations | Host composition | Deployment-specific policy | No Arcloom Package owns a fixed Feedback Loop lifecycle. | No workflow, runtime, or orchestrator Package is introduced. |
| Preserve proof completeness without becoming authoritative | Verification Evidence Record | Native references and disposable contract results selected by the proof operator | Every CA-1 through CA-6 relation is inspectable without treating the record as external state or causality. | Product Components do not own the record. |

#### Codex SDK boundary review

Risk level: High. The change introduces a public Package contract and changes the
boundary between Plan assessment translation and an external Provider protocol.
The user approved the interface-only SDK boundary before construction.

| Candidate | Remove or merge test | Decision | Reason |
|---|---|---|---|
| Codex Assessment Interaction | Merge into Plan Control Assessment | Keep | The external interaction has Provider-owned failure, isolation, safety, and cancellation conditions that do not define Assessment meaning. |
| Codex app-server Go SDK boundary | Keep inside \`codexplancontrol\` | Separate | Process/transport/protocol lifecycle changes independently of Plan material and outcome translation and has another evidenced AI consumer. |
| Generic process controller, helper, or transport utility | Extract from the SDK | Reject | No consumer requires process semantics independent of the Codex app-server contract. |

| Independent scenario / confidence | Primary owner | Expected propagation | Verdict |
|---|---|---|---|
| App-server protocol, launch, or completion representation changes / Evidence-backed plausible | Codex app-server Go SDK | SDK implementation and SDK tests; Client changes only if completed read-only Turn semantics change. | Pass |
| Process exits, blocks, or changes cooperative shutdown / Evidence-backed plausible | Codex app-server Go SDK | SDK lifecycle implementation and bounded process tests. | Pass |
| Assessment concurrency increases / Committed | SDK for interaction isolation; \`codexplancontrol\` for material/result isolation | SDK race/process tests and adapter unit tests. | Pass |
| A second Arcloom consumer needs Codex / Evidence-backed plausible | Its consumer-owned adapter plus the SDK | The SDK is reused without Plan imports; the new consumer owns its judgment translation. | Pass |
| Observation vocabulary or Plan Control outcomes change / Evidence-backed plausible or Speculative | Assessor consumer, Plan, and \`codexplancontrol\` | Plan material/output translation and adapter tests; no SDK lifecycle change. | Pass |
| Startup cost creates pressure for reuse / Evidence-backed plausible | Codex app-server Go SDK | SDK internals may change only while the isolated read-only Turn and bounded-cancellation contract remains true. No session extension point is added now. | Pass |
| Transport changes away from a child stdio process / Speculative | Codex app-server Go SDK | Reconsider SDK internals when supported; no current interface extension. | Risk only |

| Boundary candidate | Consumer and constraint | Dependency direction | Simpler alternative | Decision |
|---|---|---|---|---|
| \`codexappserver.Client\` | \`codexplancontrol\` needs one isolated read-only completed Turn while external protocol and process lifecycle remain hidden. | \`codexplancontrol\` depends on the SDK contract; the SDK never depends on Plan or Plan Control. | A direct function cannot publish the separately verified SDK lifecycle contract. | Accept |
| Generic process abstraction | No current non-Codex consumer or invariant. | N/A | Keep process details inside the future SDK implementation. | Reject |

### 3.3 Package Design

| Package | Responsibilities implemented | Contracts published | Hidden implementation | Permitted dependencies |
|---|---|---|---|---|
| \`plan\` | Plan value, invariants, and semantic equality. | Existing Plan API plus equality. | Validation and comparison details. | Standard library only. |
| \`plansnapshot\` | Provider-independent Snapshot and Plan Representation Progress invariants; consumer-owned observation Port. | Snapshot, Progress Evidence, Task Progress, Observation Failure, Observer, Observe. | Defensive copies, membership matching, contract validation. | \`plan\`, standard library. |
| \`githubplan\` | GitHub target binding, fresh REST reads, Milestone Scheme, payload format, and Snapshot Observer implementation. | Milestone target and Snapshot Observer construction; existing APIs remain. | HTTP/JSON DTOs, pagination, native identity, API version, state mapping. | \`plan\`, \`plansnapshot\`, existing \`planrepresentation\`, standard library. |
| \`authorization\` | Generic Policy, Rule, subject-bound Evaluation, and aggregate Decision semantics. | Rule, Policy, Evaluation, Decision. | Rule iteration and failure localization. | Standard library only. |
| \`planapplication\` | Stable target reference and Plan Revision invariants, Authorization gating, Actor request Port, Application Attempt safety, and request-result classification. | Target Reference, Revision, Request, Actor, Receipt Evidence, Result, RequestApplication. | Exact one-attempt classification and defensive value handling. | \`plan\`, \`authorization\`, standard library. |
| \`plancontrol\` | Existing provider-independent Assessment semantics. | Existing Assessor and Assessment contracts. | Response validation. | \`plan\`, standard library. |
| \`codexappserver\` | Contract surface of the separately developed Codex app-server Go SDK. | Read-only Turn Request, Completed Turn, Client. | Future SDK implementation details including process lifecycle, transport, JSON-RPC, correlation, and bounded shutdown. | Standard library only in this Change. |
| \`codexplancontrol\` | Codex-backed implementation of the Assessor Port through the SDK contract. | Validated assessment Configuration, Observation Encoder, Assessor construction. | Plan material encoding, assessment instructions/schema, and final-output translation. | \`codexappserver\`, \`plan\`, \`plancontrol\`, standard library. |

No production Package is added for proof composition. Opt-in verification code composes public contracts as a test/Host concern and owns no domain decision.

| Source Package | Target Package | Public contract used | Dependency reason | Details that do not cross |
|---|---|---|---|---|
| \`plansnapshot\` | \`plan\` | Plan values and invariants | Snapshot may expose one valid current Plan. | Provider targets and progress do not enter Plan. |
| \`githubplan\` | \`plansnapshot\` | Observer and result values | GitHub implements the GitHub Plan Snapshot observation boundary. | HTTP DTOs, IDs, URLs, payload bytes, and errors. |
| \`githubplan\` | \`plan\` | Plan construction | Scheme reconstructs provider-independent Plan meaning. | GitHub state does not enter Plan. |
| \`planapplication\` | \`authorization\` | Typed Policy and subject-bound Evaluation | The exact Revision is authorized during the request invocation. | Authorization does not depend on Plan or target semantics. |
| \`planapplication\` | \`plan\` | Plan values and equality | Revision protects current/proposed invariants. | Actor protocol and receipt state do not enter Plan. |
| \`plancontrol\` | \`plan\` | Plan values | Existing Assessment concerns one exact current Plan. | AI protocol does not enter Plan Control. |
| \`codexplancontrol\` | \`plancontrol\` | Consumer-owned Assessor Port | Codex provides one concrete external-AI judgment. | SDK contracts, model metadata, and raw output. |
| \`codexplancontrol\` | \`plan\` | Plan serialization input | The adapter supplies the exact current Plan to Codex. | Codex representation does not redefine Plan. |
| \`codexplancontrol\` | \`codexappserver\` | Client, ReadOnlyTurnRequest, and CompletedTurn | The adapter requests one isolated read-only turn without owning Provider communication. | Plan, Observation, AssessorResponse, FailureCode, process, and JSON-RPC details do not cross in the wrong direction. |

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
package codexappserver

type ReadOnlyTurnRequest struct {
    /* private immutable construction material */
}

// NewReadOnlyTurnRequest validates non-blank model, reasoning effort,
// instructions, and input; an absolute working directory; and a JSON object
// output schema. Invalid material returns an error and a zero request without
// beginning an app-server interaction.
func NewReadOnlyTurnRequest(
    model string,
    reasoningEffort string,
    workingDirectory string,
    developerInstructions string,
    input string,
    outputSchema string,
) (ReadOnlyTurnRequest, error)

func (r ReadOnlyTurnRequest) Model() string
func (r ReadOnlyTurnRequest) ReasoningEffort() string
func (r ReadOnlyTurnRequest) WorkingDirectory() string
func (r ReadOnlyTurnRequest) DeveloperInstructions() string
func (r ReadOnlyTurnRequest) Input() string
func (r ReadOnlyTurnRequest) OutputSchema() string

type CompletedTurn struct {
    FinalOutput string
}

// Client is the contract implemented by the separately developed Go SDK.
// CompleteReadOnlyTurn owns one isolated app-server interaction, permits no
// approval, network, tool, or mutation capability, returns only one completed
// final output, honors cancellation, and completes shutdown within the SDK's
// accepted finite bound. The same Client accepts concurrent calls and isolates
// each call's material, session, result, and cancellation; cancelling one call
// does not affect another. Invalid request, protocol, lifecycle, or shutdown
// failure returns an error and no CompletedTurn.
type Client interface {
    CompleteReadOnlyTurn(
        context.Context,
        ReadOnlyTurnRequest,
    ) (CompletedTurn, error)
}
\`\`\`

\`\`\`go
package codexplancontrol

type Model struct { /* non-empty Codex model identifier */ }
func NewModel(value string) (Model, error)

type ReasoningEffort struct { /* validated supported effort */ }
func NewReasoningEffort(value string) (ReasoningEffort, error)

type WorkingDirectory struct { /* existing absolute directory */ }
func NewWorkingDirectory(value string) (WorkingDirectory, error)

type Configuration struct { /* immutable cohesive values */ }
func NewConfiguration(
    model Model,
    reasoningEffort ReasoningEffort,
    workingDirectory WorkingDirectory,
) Configuration

// ObservationEncoder preserves caller vocabulary and must not mutate input.
type ObservationEncoder[O any] func(context.Context, O) (string, error)

// NewAssessor validates Client, including typed nil implementations,
// Configuration, and encoder without beginning an SDK interaction. Each
// invocation supplies one exact read-only Turn to the Client and implements
// the existing Plan Control Assessor contract independently. SDK errors and
// Provider details never cross that contract.
func NewAssessor[O any](
    client codexappserver.Client,
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
| Plan Control Assessor | Existing Plan Controller needs one external AI judgment. | Codex SDK contract and output. | Plan Control must remain independent of the experimental Provider. |
| Observation Encoder | Assessor consumer owns observation meaning while the Codex adapter needs serializable material. | Caller vocabulary. | A universal Observation model would couple unrelated evidence sources. |
| Codex app-server SDK Client | \`codexplancontrol\` needs one completed read-only Turn while process and protocol concerns evolve independently. | Process lifecycle, transport, JSON-RPC, correlation, app-server DTOs, and bounded shutdown. | Keeping the concrete process implementation in \`codexplancontrol\` gives Plan assessment translation and Provider infrastructure different change drivers in one Package. |

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
| \`github-plan-snapshot/*\` | Valid facts produce exact Plan/progress; invalid, incomplete, and unavailable observations do not produce a current Plan. | Table-driven HTTP contract tests and race tests. | \`githubplan\`, \`plansnapshot\` | Passing Go tests. |
| \`authorization/*\` | Exact subject forwarding and aggregate decisions fail closed. | Table-driven unit tests, cancellation tests, import review. | \`authorization\` | Passing Go tests and dependency check. |
| \`plan-application-request/*\` | Only Authorized enters a possibly-sent Attempt; every receipt certainty maps exactly; no state claim. | Table-driven unit tests with the Actor Port only. | \`planapplication\` | Passing Go tests. |
| \`codex-plan-control-assessment/*\` adapter behavior | A completed SDK Turn yields the existing Plan Control response; invalid AI output, cancellation, and concurrency fail safely. | Unit tests against the Client contract; no process and no live model. | \`codexplancontrol\` | Passing Go tests. |
| Codex app-server communication and lifecycle | Protocol, process, cancellation, and concurrent SDK interactions satisfy the Client contract. | Unit and process contract tests owned by the separately developed Go SDK. | Codex app-server Go SDK | SDK verification evidence before live composition. |
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
| Codex structured output | Client returns one completed final output | Assess | Existing AssessorResponse is returned. | Unit | No Provider data crosses. |
| Codex request meaning | Every Plan element and caller-owned observation value varies independently | Assess through a capturing Client | The Client receives the exact semantic Plan and observation material with existing Plan Control outcome meanings. | Unit boundary | Do not assert exact instruction prose. |
| Codex protocol or lifecycle failure | Client returns an error | Assess | Failure follows the Codex failure-classification table without exposing the SDK error. | Unit boundary | JSON-RPC and process cases belong to SDK tests. |
| Codex bounded cancellation | Client blocks until caller cancellation | Cancel one assessment | The adapter propagates cancellation and another concurrent assessment remains unaffected. | Unit boundary | The SDK separately proves its finite process shutdown bound. |

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

### 3.7 Detailed Design and TDD Plan

| Implementation unit | Responsibility / contract | Dependencies | Behavior and test | Representation decision |
|---|---|---|---|---|
| \`codexappserver\` contracts | Publish one validated immutable read-only Turn request, one completed result, and the concurrent Client lifecycle guarantee. | Standard library. | Constructor and accessor unit tests plus compile-time use through adapter tests; SDK behavior remains pending task 5.5. | One small contract Package; no process implementation or generic abstraction. |
| \`codexplancontrol\` Configuration | Validate Model, Reasoning Effort, and Working Directory used in one Turn request. | Standard library. | Existing table-driven validation tests without Shutdown Grace. | Immutable values and one cohesive Configuration value. |
| \`codexplancontrol\` Assessor | Encode exact material, construct one ReadOnlyTurnRequest, call Client, and translate final output. | \`codexappserver\`, \`plan\`, \`plancontrol\`. | Capturing Client tests for all outcomes, failures, cancellation, repetition, and concurrency. | Function-based Assessor plus the approved external Client interface; no internal service or controller. |
| Removed process implementation | No longer an Arcloom Plan adapter responsibility. | None. | Existing process contract tests and protocol stub are removed; equivalent evidence is required from task 5.5. | No compatibility shim because the API is unmerged. |

| Behavior / criterion | Construction mode | Red or baseline | Smallest implementation | Refactor target / evidence |
|---|---|---|---|---|
| Missing or typed-nil Client, invalid request material, or invalid assessment configuration starts no interaction | TDD | Change \`NewAssessor\` tests to require Client and remove process configuration; add failing request-constructor cases. | Add the immutable SDK request contract and validate Client. | Passing focused constructor tests. |
| Exact material and four outcomes cross only the Client boundary | TDD | Replace process-capturing tests with a failing in-memory Client test. | Construct ReadOnlyTurnRequest and decode CompletedTurn.FinalOutput. | Delete process and JSON-RPC code after equivalent adapter behavior passes. |
| Client and final-output failures fail closed | TDD | Replace protocol-mode cases with Client-error and completed-output cases. | Map Client error to boundary failure and retain strict output decoding. | Passing failure classification tests with no SDK error exposure. |
| Cancellation, repetition, and concurrency stay isolated | TDD | Replace child-process cases with controllable Client calls. | Forward context and keep every call stateless. | Race-tested adapter evidence; process shutdown evidence remains task 5.5. |

### 3.8 Construction Log

| Cycle | Evidence |
|---|---|
| Red | In-memory Client tests replaced process-bound assertions; request-constructor tests failed against the zero-value scaffold; typed-nil Client and full concurrent-material cases exposed missing boundary guarantees. |
| Green | Added the immutable validated SDK request contract, adapted one assessment through \`Client\`, rejected nil Client forms, and mapped only the completed final output into Plan Control meaning. |
| Refactor | Removed the child-process launcher, JSON-RPC implementation, protocol stub, shutdown configuration, and process-owned tests from \`codexplancontrol\`; lifecycle verification remains task 5.5 under the SDK owner. |
| Verify | Focused race tests passed 20 repetitions; repository race tests, lint, module tidy diff, strict OpenSpec validation, and three independent boundary reviews passed. |

### 3.9 Design Conformance Review

| Boundary | Result |
|---|---|
| \`codexappserver\` | Contains only the SDK-facing Turn contract and its local value validation; no process, transport, JSON-RPC, Plan, or Plan Control responsibility. |
| \`codexplancontrol\` | Contains only assessment configuration, exact material encoding, Client invocation, and final-output translation; it owns no Codex process lifecycle. |
| Dependency and minimality | Dependency remains \`codexplancontrol\` to \`codexappserver\`; no helper, utility, common, shared-session, process-controller, or speculative transport abstraction was introduced. |
| Verification ownership | Adapter unit and race tests verify Plan Control behavior; SDK lifecycle and protocol tests remain explicitly pending in task 5.5. |

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

### 4.7 Codex app-server lifecycle belongs to its Go SDK

- Adopted: \`codexplancontrol\` consumes a Client contract for one isolated read-only Turn. The separately developed SDK owns process, transport, JSON-RPC, correlation, and bounded shutdown.
- Rejected: Process and protocol implementation inside \`codexplancontrol\`, and a shared long-lived Codex thread or session pool in Plan Control.
- Reason: Plan assessment translation changes with Plan Control meaning; app-server communication and lifecycle change with Codex. The boundary keeps both owners cohesive and permits another Codex consumer without importing Plan semantics.

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
| Codex app-server protocol, launch contract, or transport changes. | Confine the change to the Go SDK and its contract tests; \`codexplancontrol\` continues to consume one completed read-only Turn. |
| The SDK contract is present before its implementation. | Keep live Codex composition incomplete and explicit until the separately developed SDK supplies passing lifecycle and protocol verification evidence. |
| GitHub API version \`2022-11-28\` is retired. | Keep version and DTO handling inside \`githubplan\`; update its contract tests without changing provider-independent Packages. |
| A stable target reference can be mapped to the wrong native target by a Host or Actor. | Require non-empty Context and identity values, preserve them exactly through Revision and Request, and leave action-time native-target validation with the Actor. |
| Live proof depends on real credentials, exact approval, and Actor availability. | Keep it opt-in, require explicit Host inputs, and never infer approval from repository access. |
| An isolated SDK interaction may incur process startup cost. | Accept the cost until the SDK implementation measures a need for another lifecycle without weakening the Client or Assessor contracts. |

## 5. Impact, Migration, and Rollback

### Impact

- Additive Packages: \`plansnapshot\`, \`authorization\`, \`planapplication\`, the \`codexappserver\` SDK contract, and \`codexplancontrol\`.
- Additive read-side API in \`githubplan\`; existing Plan Representation observation and creation-preview contracts remain.
- Additive semantic equality in \`plan\`; \`plancontrol\` delegates its existing comparison rule to that owner.
- \`PRODUCT.md\` and \`ARCHITECTURE.md\` replace Change-specific Authorization and the Plan Change Target with the generic Authorization and Plan Application Request boundaries.

### Migration

- No data migration exists because Arcloom owns no persistent state.
- The unmerged \`codexplancontrol\` API changes before release to require a
  \`codexappserver.Client\`; no released caller compatibility is affected.
- New Hosts opt into each contract independently and retain target/provenance in their composition.

### Rollback

- Remove the additive Packages and GitHub Snapshot API, including the SDK contract, restore the previous canonical-document terminology, and retain the existing read-only observer, dry-run preview, and Plan Control APIs.
- No external data rollback is performed. Any GitHub change is owned and recorded by the external Actor and Planning Context.

## Open Questions

None.
