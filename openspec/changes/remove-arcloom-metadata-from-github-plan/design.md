## Context

| Information | Normative source |
|---|---|
| Product scope and external-source authority | `PRODUCT.md` |
| Components, ownership, dependencies, Ports, and external boundaries | `ARCHITECTURE.md` |
| Accepted and proposed behavior | Main specs and this Change's delta specs |
| Design, test, and review policy | `docs/DEVELOPMENT.md` |
| Requirements and conceptual analysis | `proposal.md` and `model.md` |

The GitHub Plan Adapter currently renders one human narrative after an encoded payload and decodes that payload independently in representation projection and Snapshot current-Plan construction. The public creation, representation-observation, and Snapshot contracts already provide every consumer boundary required by this Change. The implementation must replace the private source-of-meaning decision without changing those public contracts or moving GitHub grammar into Plan Components.

Risk is High because persisted external representations and public observable creation and observation behavior change. Construction requires human approval after this DesignDoc review.

## Goals / Non-Goals

### Goals

- Make creation and both observation paths use one strict native narrative classification.
- Preserve exact accepted Plan values and reject structural collisions before producing a partial Request Plan.
- Remove all runtime authority and production decoding for `arcloom-plan:v1`.
- Preserve existing GitHub target, REST, pagination, Task, failure, cancellation, concurrency, and provider-independent Observation contracts.
- Produce deterministic fixture evidence and separate live Milestone #3 evidence.

### Non-Goals

- Add a Markdown parser, format plugin, Provider-wide interface, public narrative contract, or new package.
- Mutate or backfill GitHub resources.
- Preserve marker-only content as a current Plan.
- Change Plan validity, Task collection behavior, Authorization, application, or Host composition.

## Conceptual-Model-to-Implementation Mapping

| Specification concept / Requirement | Owning component | Physical representation | Notes |
|---|---|---|---|
| Plan | Plan | Existing `controllers/plan.Plan` and value constructors | Unchanged. |
| GitHub Plan Representation | GitHub Plan Adapter | Existing private `scheme` variants and `NewCreationRequestPlan` validation | `scheme` decides supported representation after narrative collision classification. |
| GitHub Plan Narrative | GitHub Plan Adapter | Private exact-source classifier and renderer in `providers/github/plan` | No public type, package, or Port. |
| Acceptance Condition Sequence Establishment | GitHub Plan Adapter | Private outcome value containing exact members and completeness | It represents one call-local classification, not stored state. |
| Observed GitHub Fact | GitHub Plan Adapter | Existing `rootFact` and `githubFactSet` plus one private narrative outcome | Native/legacy precedence is decided once before projection. |
| Creation Request Plan | GitHub Plan Adapter | Existing `RequestPlan` and sealed `Request` values | Public signatures remain unchanged. |
| Provider-independent Observation | Plan Representation Reconciliation | Existing `controllers/plan/representation` values and `Observer` Port | GitHub syntax does not enter the contract. |
| Current Plan Snapshot | Plan Snapshot Observation | Existing `controllers/plan/snapshot` values and `Observer` Port | Eligibility consumes the same narrative outcome as representation projection. |
| `[[github-plan-creation-dry-run/github-plan-narrative]]` | GitHub Plan Adapter | Exact root description or body string | Replaces payload plus narrative bytes. |
| `[[github-plan-representation-observation/native-narrative-meaning]]` | GitHub Plan Adapter | Private narrative classification mapped to existing public Observation states | Legacy payload is generic preamble, not decoded. |

## Responsibility and Boundary Design

### Responsibility Assignment

| Responsibility / decision | Owner | Information and authority used | Invariant protected | Excluded owner and reason |
|---|---|---|---|---|
| Plan validity | Plan | Provider-independent exact values | Plan rules do not depend on GitHub grammar | GitHub Adapter lacks product authority. |
| Native representability | GitHub Plan Representation | Selected scheme and narrative collision classification | Valid unrepresentable Plan returns `UnsupportedRepresentation` without becoming invalid | Narrative classification identifies collision but does not own request acceptance. |
| Exact grammar production and classification | GitHub Plan Narrative responsibility inside the Adapter | Raw source bytes and closed structural markers | Accepted creation output and observed meaning agree exactly | A parser or codec abstraction has no independent consumer or lifecycle. |
| Acceptance ordinal prefix and completeness | Acceptance Condition Sequence Establishment inside the Adapter | Recognized member frames and next expected ordinal | Members equal the longest coherent prefix | Plan collection validity does not own Provider framing. |
| Native/legacy authority and Plan-location mapping | Observed GitHub Facts inside the Adapter | Scheme, native fields, narrative outcome, Task fact set | Legacy content never supplies a fact; independent facts survive localized failure | Observation Outcome owns request lifecycle, not source precedence. |
| Generic available, unavailable, violation, and collection states | Plan Representation Observation | Provider-independent consumer meaning | Provider vocabulary does not leak | GitHub Adapter only maps into these states. |
| Valid Observation versus caller lifecycle failure | GitHub Observation Outcome | Context and REST call result | Cancellation and Provider failures remain isolated | Narrative semantics do not decide request completion. |

### Architecture Boundary Plan

| Boundary candidate | Consumer / evidence | State, data, or policy owner | Constraint protected | Dependency direction | Decision |
|---|---|---|---|---|---|
| Existing `providers/github/plan` Adapter | Creation callers and existing Plan-owned observation Ports | GitHub owns remote facts; Adapter owns call-local mapping and grammar | Confines REST, identifiers, raw content, grammar, pagination, and errors | Adapter depends inward on `controllers/plan`, `representation`, and `snapshot` | Retain. |
| New narrative package or Port | No consumer outside the Adapter | No independent state or substitution owner | No new constraint | Would add Adapter-internal indirection | Reject. |
| New Provider-wide GitHub interface | No consumer needs combined creation and observation operations | Existing consumers own minimal Ports | Would expose Provider-centered surface | Wrong dependency direction | Reject. |
| Existing Plan Representation Observation Port | Plan Representation Reconciliation | Consumer owns provider-independent fact states | Prevents Provider leakage | Adapter implements inward-owned Port | Retain unchanged. |
| Existing Plan Snapshot Observation Port | Plan Attempt through Snapshot component | Consumer owns coherence and eligibility | Prevents Provider from deciding Plan eligibility | Adapter implements inward-owned Port | Retain unchanged. |

Evidence-backed REST, text-serialization, field-limit, relationship, pagination, authorization-scope, transient-failure, and cancellation changes propagate only within the existing Adapter and its tests. A second Provider adds its own Adapter and leaves GitHub narrative rules unchanged. Speculative payload restoration, caching, mutation, and rendered-Markdown changes create no current extension point.

### Package Design

| Package | Responsibility | Carries | Must not carry |
|---|---|---|---|
| `providers/github/plan` | GitHub representation, native narrative, request planning, REST fact acquisition, and mapping | Private narrative outcome, fact sets, existing public construction and Observer contracts | Plan validity policy, Snapshot eligibility, Reconciliation result, Authorization, mutation, or durable state |
| `controllers/plan` | Provider-independent Plan meaning | Existing exact values and validation | GitHub markers, field limits, DTOs, or representability |
| `controllers/plan/representation` | Provider-independent observed representation and reconciliation | Existing location, availability, violation, completeness, evidence, and result contracts | GitHub source syntax or compatibility policy |
| `controllers/plan/snapshot` | Current Plan eligibility, progress, and Snapshot coherence | Existing Observer and Snapshot contracts | GitHub grammar or payload meaning |

| Boundary | Hidden detail | Dependency direction |
|---|---|---|
| `providers/github/plan` | Raw Markdown source, exact framing, preamble, legacy bytes, REST DTOs, identities, API version, pagination, and Provider failures | Toward existing Plan and consumer-owned observation contracts |

## Interface Design

No public signature changes. The following existing contracts remain the complete public surface affected by this Change:

```go
// Existing, unchanged public contracts.
func NewCreationRequestPlan(Repository, Representation, plan.Plan) (RequestPlan, error)
func NewMilestoneObserver(*http.Client, Repository, ResourceNumber) (planrepresentation.Observer, error)
func NewIssueObserver(*http.Client, Repository, ResourceNumber) (planrepresentation.Observer, error)
func NewMilestoneSnapshotObserver(*http.Client, MilestoneTarget) (plansnapshot.Observer, error)
```

The following is planned private Go design, not existing code or a public contract:

```go
// narrativeFor returns false only when exact Plan Text collides with the
// selected scheme's closed native grammar. It performs no I/O.
func (s scheme) narrativeFor(plan.Plan) (string, bool)

// establishNarrativeFacts establishes only exact source structure. Each
// observer invocation calls this shared implementation once for its fact set.
// Plan classification remains in the existing representation/Snapshot mapping.
func (s scheme) establishNarrativeFacts(textFact) narrativeFacts

type narrativeFacts struct {
	goal       establishedText
	conditions establishedConditions
	targetDate establishedTargetDate
}
```

Private outcome zero values represent structurally unavailable source meaning. The private result contains only exact established text, Acceptance Conditions members with Complete or Incomplete membership, and Issue Target Date as Unavailable, Absent, or Present exact text. It never contains a Plan violation. Existing `planrepresentation` classifiers and `plan` constructors classify established text at their current mapping boundaries. Returned slices are immutable to consumers. The exact private field layout may change during TDD if these ownership and single-shared-classifier invariants remain.

Each representation-observer invocation calls `establishNarrativeFacts` exactly once before `scheme.project` maps its result. Each Snapshot-observer invocation calls the same implementation exactly once before `scheme.currentPlan` evaluates eligibility. The two public Observer invocations remain independent and do not share mutable results.

| Public contract | Requirement / external dependency | Owner | Consumer | Why separate |
|---|---|---|---|---|
| `NewCreationRequestPlan` | GitHub request-plan behavior and native representability | GitHub Plan Adapter | Host or request-plan caller | Exposes pure Provider-specific creation planning without mutation. |
| `planrepresentation.Observer` returned by Milestone/Issue constructors | Current GitHub facts through provider-independent vocabulary | Plan Representation Reconciliation | Its Controller | Isolates REST and raw source. |
| `plansnapshot.Observer` returned by Milestone Snapshot constructor | Fresh current Plan and progress | Plan Snapshot Observation | Plan Attempt composition | Keeps eligibility and coherence with the consumer. |

| Contract | Preconditions | Postconditions / result semantics | Errors / side effects |
|---|---|---|---|
| `NewCreationRequestPlan` | Valid Repository, explicit Representation, and valid Plan | Deterministic immutable Request Plan for accepted representable input; zero `RequestPlan` otherwise | Structural collision returns `*ValidationError` with `UnsupportedRepresentation` and `RepresentationField`; simultaneous independent invalid-input precedence is unspecified; no I/O or mutation. |
| `NewMilestoneObserver` / `NewIssueObserver` | Valid no-cookie client, Repository, and positive Resource Number | Return one target-bound `planrepresentation.Observer` | Construction validates without I/O. Invalid input returns existing stable `*ValidationError`. |
| Returned `planrepresentation.Observer` | Caller context and current bound target | Successful calls return one valid provider-independent Observation; malformed narrative is localized inside that Observation | Performs read-only GitHub I/O. Before success, caller cancellation/deadline returns exact `ctx.Err()` and no successful Observation. Other Provider failures do not cross the Port. Concurrent calls remain isolated. |
| `NewMilestoneSnapshotObserver` | Valid no-cookie client and Milestone Target | Returns one target-bound `plansnapshot.Observer` | Construction validates without I/O. Invalid input returns existing stable `*ValidationError`. |
| Returned `plansnapshot.Observer` | Caller context and current bound target | Coherent root/progress returns a Snapshot; invalid or incomplete native Plan meaning returns a valid Snapshot without current Plan when progress is coherent | Root/progress unavailability returns existing `UnavailableObservationFailure`; before success, caller cancellation/deadline returns exact `ctx.Err()`; Provider details do not cross. Concurrent calls remain isolated. |

Illustrative call sites below are written for this DesignDoc and are not executed examples:

```go
requestPlan, err := githubplan.NewCreationRequestPlan(repository, representation, current)
if err != nil {
	var validation *githubplan.ValidationError
	if errors.As(err, &validation) &&
		validation.Code() == githubplan.UnsupportedRepresentation &&
		validation.Field() == githubplan.RepresentationField {
		// The Plan is valid but cannot use this GitHub representation.
	}
}
```

```go
observer, err := githubplan.NewMilestoneObserver(client, repository, number)
if err != nil {
	return err
}
controller, err := planrepresentation.NewController(observer)
if err != nil {
	return err
}
result, err := controller.Reconcile(ctx, expected)
```

```go
observer, err := githubplan.NewMilestoneSnapshotObserver(client, target)
if err != nil {
	return err
}
snapshot, err := plansnapshot.Observe(ctx, observer)
if err == nil {
	current, hasCurrent := snapshot.CurrentPlan()
	_, _ = current, hasCurrent
}
```

## Decisions

### Decision: Use one closed raw-source grammar

- **Choice**: Recognize only the exact LF-framed H2/H3 structure in the delta spec and preserve value bytes without normalization.
- **Rationale**: It accepts Milestone #3, makes behavior deterministic, and avoids interpreting rendered Markdown or arbitrary prose.
- **Alternatives**: A CommonMark parser makes rendering behavior authoritative and broadens scope. A blockquote or encoded format does not match Milestone #3. Permissive heuristics make malformed outcomes non-reproducible.
- **Consequences**: Reserved-line collisions are unsupported. Human-readable but noncanonical content can remain unavailable.

### Decision: Classify narrative once per fact set

- **Choice**: Produce one private narrative outcome from the root content and use it in both `scheme.project` and `scheme.currentPlan`.
- **Rationale**: Representation reconciliation and Snapshot eligibility must not disagree about native/legacy precedence, exact values, or collection completeness.
- **Alternatives**: Separate decoders duplicate policy. Moving classification into Plan Components leaks GitHub grammar inward.
- **Consequences**: The current payload paths in `scheme.go` and `snapshot_observer.go` converge on one private result.

### Decision: Remove payload production and decoding

- **Choice**: Delete production payload encoding, JSON decoding, and marker-specific interpretation after native tests are green.
- **Rationale**: Retaining unused codec code preserves a false authority and encourages fallback behavior prohibited by the spec.
- **Alternatives**: Legacy fallback violates Issue #50. A versioned strategy has no second accepted format.
- **Consequences**: Marker-only resources lose content-backed facts. Legacy plus native content works because generic preamble is ignored.

### Decision: Preserve existing public contracts and package boundary

- **Choice**: Add no exported type, method, Port, Component, or package.
- **Rationale**: Every consumer and external constraint is already protected by the GitHub Plan Adapter and Plan-owned Ports.
- **Alternatives**: A public parser or Provider-wide service has no external consumer and weakens dependency direction.
- **Consequences**: Native grammar changes remain private implementation changes governed by the two affected capabilities.

### Decision: Verify live acceptance as operator evidence

- **Choice**: Add a deterministic Milestone #3 REST fixture for CI and run an opt-in authenticated black-box Snapshot observation against the live target, recording the exact command, timestamp, current facts, result, and drift status in `verification/native-milestone-3.md`.
- **Rationale**: CI remains deterministic while Issue #50 receives real external evidence through public contracts.
- **Alternatives**: A mandatory network test is nondeterministic. A raw `gh api` response alone does not prove the public Snapshot contract.
- **Consequences**: Live drift fails or defers the external check and never substitutes for the deterministic fixture.

## Test Specification

### Requirement Coverage

| Requirement / criterion | Observable behavior | Automated test or verification | Owner | Required evidence |
|---|---|---|---|---|
| `github-plan-creation-dry-run/explicit-github-target-and-representation` | Accepted native plans produce a plan; collisions return exact unsupported validation and no plan | Public table tests through `NewCreationRequestPlan` | `githubplan` | Literal validation code, field, and zero result |
| `github-plan-creation-dry-run/github-plan-narrative` | Exact canonical bytes, round-trip values, no marker or opaque preamble | Public literal producer tests with independent expected strings | `githubplan` | Both schemes, date states, Unicode, CRLF, and whitespace |
| `github-plan-creation-dry-run/milestone-creation-request-plan` | Zero and ordered Tasks retain Milestone request topology with native content | Existing 0/1/multiple Task public request tests updated with literals | `githubplan` | Exact request order, optional date, and Milestone-number references |
| `github-plan-creation-dry-run/issue-creation-request-plan` | 0/100 Tasks succeed, 101 fails, and request pairs retain topology | Existing boundary and public request-graph tests retained | `githubplan` | Exact pair order, parent number, child identity, and unsupported result |
| `github-plan-creation-dry-run/validation-result` and narrative idempotency | Repeated inputs are byte-equal; every invalid or unsupported class returns exact stable evidence | Public validation and repeated-call tests | `githubplan` | Code, Field, zero plan, and exact repeated requests |
| `github-plan-representation-observation/native-narrative-meaning` | Native, malformed, legacy, invalid-value, and incomplete-prefix cases expose exact public Results | Controller-mediated black-box observation matrix | `githubplan` | Determination, Differences, Unavailable Locations, and no Provider detail |
| `github-plan-representation-observation/milestone-representation-meaning` | Native titles, date, closed Issue inclusion, Pull Request exclusion, and relationships map independently | Existing Milestone mapping baselines updated to native fixtures | `githubplan` | Closed/open Issues, Pull Requests, zero/one/page boundaries, absent/invalid `due_on` |
| `github-plan-representation-observation/issue-representation-meaning` | Native title, narrative/date, empty Sub-issues, and parent Pull Request behavior remain exact | Existing Issue mapping baselines updated to native fixtures | `githubplan` | Empty/complete Tasks, parent Pull Request root failure, and date states |
| `conservative-failure-observation` | Narrative failures remain localized and lifecycle failures retain existing contract | Existing failure, cancellation, pagination, and race suites plus focused narrative cases | `githubplan` | Exact public results and `ctx.Err()` where applicable |
| Snapshot current-Plan eligibility | Native narrative creates a current Plan only when exact values and complete collections are valid | Public Snapshot Observer tests | `githubplan` | Current Plan equality and progress preservation |
| SC-3 / Milestone #3 | Frozen fixture is deterministic; live target is observed through the public Snapshot contract | CI fixture plus opt-in external verification | Proof operator | `verification/native-milestone-3.md` |
| Main-spec and Architecture publication | Payload authority terminology is removed without ownership drift | `openspec validate`, Markdown lint, OpenSpec lint after archive | Change owner | Passing command output |

### Behavior Tests

| Behavior | Given | When | Then | Level |
|---|---|---|---|---|
| Canonical producer | Exact representable Plan values | Create each Request Plan | Description/body equals independent literal and contains no marker | Unit, public |
| Collision rejection | Reserved H2 in Goal or condition, or `### ` line in condition | Create either representation | `UnsupportedRepresentation`, `RepresentationField`, no plan | Unit, public |
| Exact text | Unicode, CRLF, leading/trailing whitespace, extra LF | Create then observe literal content | Exact public Plan values are recovered | Unit, public |
| Global structure failure | Missing, duplicate, out-of-order, or malformed required H2 | Reconcile through public Controller | Narrative locations are unavailable/incomplete and independent facts remain | Unit, black-box |
| Condition prefix | Valid members followed by every malformed ordinal/framing class | Observe | Exact prefix remains and membership is Incomplete | Unit, black-box |
| Empty conditions | Canonical zero-member EOF and dated forms | Observe | Complete empty collection yields existing invalid-current-Plan behavior | Unit, black-box and Snapshot |
| Issue date states | Absent, valid, invalid, duplicate, out-of-order, malformed framing | Observe | Exact Absent, Present, violation, or Unavailable state | Unit, black-box |
| Generic preamble | Arbitrary non-marker prose precedes valid native narrative, then is removed | Observe both forms | Public Results are equal and only native sections supply facts | Unit, black-box |
| Legacy compatibility | Marker plus native, conflicting marker plus native, marker-only, marker removed | Observe | Native-only authority and required unavailable states | Unit, black-box |
| Milestone narrative date heading | A Milestone description contains `## Target Date` while native `due_on` is valid | Observe | Narrative locations globally fail; native Target Date remains | Unit, black-box |
| Shared classification | Same current content enters representation and Snapshot observers | Observe | Both use identical exact Goal, conditions, and date meaning | Contract integration |
| Milestone #3 fixture | Captured root and four Task responses | Observe Snapshot | Exact current Plan and complete open progress are established | Deterministic integration |

### Plan Value Classification Matrix

| Established native value | Representation reconciliation evidence | Snapshot evidence |
|---|---|---|
| Empty or Unicode-whitespace Goal | `InvalidObservedDifference` at Goal with `InvalidText`; unrelated locations remain | No current Plan; coherent progress remains |
| Empty or Unicode-whitespace Acceptance Condition | Exact member is established; `InvalidObservedDifference` at Acceptance Conditions with `InvalidText` | No current Plan; coherent progress remains |
| Duplicate exact Acceptance Conditions | Complete exact collection is established; duplicate-condition violation is reported | No current Plan; coherent progress remains |
| Valid members mixed with one invalid member | Complete membership and every exact member remain; violation identifies the invalid member under existing Plan rules | No current Plan; coherent progress remains |
| Incomplete prefix with individually valid members | Prefix remains observable and collection is Incomplete | No current Plan; coherent progress remains |
| Invalid framed Issue date | `InvalidObservedDifference` at Target Date with `InvalidTargetDate` | Not applicable to the current Milestone-only Snapshot contract; representation behavior remains authoritative |

### Cross-Consumer Source-Authority Matrix

Milestone representation and Snapshot observers receive the same literal root and Task fixtures. Tests assert only their respective public contracts.

| Source case | Representation Observer | Snapshot Observer |
|---|---|---|
| Valid native narrative | Exact native facts; no narrative Unavailable locations | Exact current Plan when progress is complete |
| Malformed required H2 | Narrative locations unavailable/incomplete; native name/date/tasks preserved | No current Plan; coherent progress preserved |
| Incomplete Acceptance ordinal prefix | Exact prefix and Incomplete membership | No current Plan; coherent progress preserved |
| Invalid Goal or Acceptance member | Exact Plan violation | No current Plan; coherent progress preserved |
| Arbitrary non-marker preamble plus valid native narrative | Native facts only; removing the preamble yields the same Result | Same current Plan; removing the preamble yields the same Snapshot |
| Legacy preamble plus conflicting payload and valid native narrative | Native facts only | Native current Plan only |
| Marker-only content | Narrative-backed facts unavailable/incomplete | No current Plan; coherent progress preserved |
| Same native narrative after marker removal | Same public Result as before removal | Same current-Plan presence and exact value as before removal |
| Milestone description with unique ordered `## Target Date` and valid native `due_on` | Narrative locations globally fail; native Target Date remains present | No current Plan from the malformed narrative; native progress remains coherent |

### Representability Boundary Matrix

| Plan Text line | Location | Expected creation result |
|---|---|---|
| Exact `## Goal`, `## Acceptance Conditions`, or `## Target Date` at start, middle, or end | Goal or Acceptance Condition | Unsupported representation |
| Any exact full line beginning `### ` | Acceptance Condition | Unsupported representation |
| `### 1` | Goal | Accepted; Goal has no ordinal-member grammar |
| `## Goal `, `###\t1`, indented markers, embedded substrings, and non-reserved H2/H3 | Goal or Acceptance Condition | Accepted and preserved exactly |
| Marker-like line terminated with CRLF rather than the structural LF form | Goal or Acceptance Condition | Accepted unless its exact LF-delimited bytes meet a reserved rule |

### Ordinal and Date Partitions

| Partition | Exact public expectation |
|---|---|
| Wrong, duplicate, gap, out-of-order, zero, leading-zero, or nonnumeric ordinal | Preserve only members before the first defect; Incomplete; a later canonical member is not resumed |
| Missing member opening separator or missing undated final LF | Preserve previously closed members; Incomplete |
| Issue date absent under valid structure | Known Absent |
| Empty, Unicode-whitespace, noncanonical, multiline, or impossible date with canonical framing | Known `InvalidTargetDate` violation |
| Unique ordered date H2 with malformed opening that cannot close conditions | Target Date Unavailable and conditions Incomplete |
| Unique ordered date H2 with valid opening but missing final LF | Target Date Unavailable; already closed conditions remain Complete |
| Duplicate or out-of-order date H2 | Global narrative structural failure |

### Error and Edge Cases

| Case | Expected result |
|---|---|
| Body or description unavailable | Every narrative-backed location Unavailable; independent facts remain. |
| Reserved H2 embedded in value | Global structural failure. |
| Non-reserved heading in condition | Exact condition text. |
| Wrong, duplicate, gapped, zero, leading-zero, out-of-order, or nonnumeric ordinal | Longest valid prefix and Incomplete membership. |
| Missing EOF framing | Last unclosed member omitted; prior prefix Incomplete. |
| Empty or impossible framed Issue date | Known `InvalidTargetDate`. |
| Malformed Issue date boundary or final LF | Target Date Unavailable; conditions Incomplete when the boundary cannot close them. |
| Concurrent observers and cancellation between pages | Existing isolated results and exact caller lifecycle outcome. |

Tests do not assert private function names, call order, parser state layout, or codec removal as behavioral outcomes. Removal is verified by production-reference search and code review after public tests are green.

### Milestone #3 Evidence Protocol

| Item | Required value or rule |
|---|---|
| Target | Repository `kotokumu/arcloom`, Milestone `3` |
| Public entry point | `NewMilestoneSnapshotObserver` with a Host-supplied authenticated no-cookie `http.Client` |
| Frozen expectation | Title `Arcloom Reconciliation Model and Module Finalization`; exact captured Goal; five exact ordered Acceptance Conditions; absent Target Date; Tasks #47, #48, #57, and #58 in stable native order with captured states |
| Deterministic prerequisite | The checked-in REST fixture passes through the public Snapshot Observer with no network access |
| Live prerequisite | Explicit GitHub credential with read access; operator records runtime and GitHub API version |
| Freshness | Evidence is captured after the final implementation commit and within 24 hours before the completion review |
| Pass | One fresh public Snapshot observation establishes a current Plan equal to the then-recorded native root and complete Task progress; marker absence is independently recorded from the same fetched root |
| Fail | Public observation disagrees with the same current native facts or still requires payload meaning |
| Defer | Access or external drift prevents comparison; defer does not satisfy SC-3 and blocks completion until rerun |
| Accountable owner | Change implementer records the evidence; human reviewer approves it before construction completion |
| Evidence | Exact command with secret values omitted, UTC timestamp, root and Task response SHA-256 hashes, observed Plan/progress, drift result, and reviewer identity in `verification/native-milestone-3.md` |

### Metadata Dependency Inventory

Before implementation, freeze the current non-archive file list returned by `rg -l 'arcloom-plan:v1|decodePayload|payloadFor'`. Completion requires:

- zero production producer or decoder references;
- no main spec, current fixture, or current verification text that defines the marker as authoritative;
- explicit allowlisting of each remaining occurrence as a legacy compatibility input or immutable archived history;
- a recorded command, result, and reviewer decision in the verification record.

## Detailed Design and TDD Plan

### Detailed Design Gate

| Implementation unit | Implements concept / responsibility / contract | Dependencies | Behavior / test | Migration or rollback impact |
|---|---|---|---|---|
| Private narrative facts in `providers/github/plan` | GitHub Plan Narrative and Acceptance Condition Sequence Establishment | Raw source only; `plan` classification remains at existing mapping boundaries | Producer, grammar, prefix, date, and compatibility matrices | Replaces payload outcome; no persisted state |
| `scheme.creationRequests` and root narrative construction | GitHub Plan Representation and Creation Request Plan | Private narrative rendering | Exact literals and collision rejection | Output bytes break intentionally |
| `scheme.project` | Observed GitHub Facts to representation Observation | Shared narrative outcome and existing Observation constructors | Controller-mediated public Results | Marker-only facts become unavailable |
| `scheme.currentPlan` | Snapshot current-Plan eligibility | Same narrative outcome, existing Plan constructors, existing Task/date facts | Snapshot tests | No separate compatibility policy |
| Main specs and `ARCHITECTURE.md` | Published concepts and durable Adapter responsibility | Approved model and delta specs | OpenSpec/Markdown validation | Rollback must restore matching behavior and specs together |
| Verification fixture and evidence record | Milestone #3 acceptance | Public Snapshot Observer and operator-supplied GitHub access | Deterministic and opt-in live checks | Disposable evidence only |

| Implementation unit | Simplest viable representation | Need | Rejected simpler alternative | Verdict |
|---|---|---|---|---|
| Narrative classification | Private immutable value and functions | Multiple consumers require one consistent classification with localized states | Primitive booleans lose Goal/collection/date distinctions | Accept |
| Acceptance sequence | Private slice plus completeness value | Must preserve exact prefix and completeness | Plain slice cannot represent omitted remainder | Accept |
| Public API | Existing functions and Ports | All consumers already served | New interface adds no constraint | Reuse |
| Legacy handling | Generic preamble rule | Compatibility without authority | Retained decoder creates forbidden fallback risk | Use no decoder |

### TDD Plan

| Behavior / criterion | Construction mode | Red test or baseline / verification | Smallest implementation | Refactor target / evidence |
|---|---|---|---|---|
| Native creation bytes | Red-Green-Refactor | Replace payload literal expectation with exact native literal for Milestone | Render canonical Goal and condition frames | Share scheme-independent framing only where rules are identical |
| Issue date and both schemes | Red-Green-Refactor | Add dated/undated Issue and collision tables | Extend private rendering under existing `scheme` | Keep representation-specific date ownership explicit |
| Native observation happy path | Red-Green-Refactor | Controller-mediated native Milestone case fails with unavailable facts | Classify required H2, Goal, and complete conditions | Introduce one private outcome only after test pressure |
| Structural and prefix matrix | Red-Green-Refactor per row family | One failing public result case at a time | Add minimal state needed for that family | Consolidate duplicated framing checks without general parser abstraction |
| Issue date outcomes | Red-Green-Refactor | Absent, valid, invalid, and unavailable cases | Add date classification to shared narrative outcome | Preserve Plan date validation ownership |
| Invalid native Plan values | Red-Green-Refactor | Goal/member invalid, duplicate, and mixed-member public Results plus Snapshot absence | Establish exact text only, then reuse existing Plan classifiers | Keep validation out of narrative grammar |
| Snapshot shared meaning | Red-Green-Refactor | Cross-consumer matrix initially diverges or yields no current Plan | Consume shared narrative facts in `currentPlan` | Remove duplicate source-authority logic |
| Preamble and legacy compatibility | Red-Green-Refactor | Arbitrary preamble, removed preamble, marker plus native, conflicting, marker-only, removed-marker, and Milestone narrative-date cases | Ignore generic preamble; add no payload decoder; keep Milestone `due_on` independent | Delete marker-specific production paths after green |
| Retained creation and mapping baselines | Baseline-green plus focused Red-Green | 0/100/101 Tasks, request topology, closed Issue/PR, parent PR, Milestone date, pagination, idempotency | Update only root native fixtures and assertions required by changed behavior | Preserve unchanged public guarantees |
| Payload removal | Baseline-green refactor | All new behavior and retained suites green | Remove payload files and stale tests/references | `rg arcloom-plan:v1` has only intentional archived or compatibility evidence references |
| Milestone #3 proof | Approved non-automated criterion | Deterministic fixture green first | Run opt-in public Snapshot observation | Record exact evidence and drift status |

## Risks / Trade-offs

- Strict framing rejects some otherwise valid Plan Text. → Keep rejection Provider-specific, stable, and fully tested.
- GitHub may normalize raw text or change field limits. → Keep exact-source behavior at the Adapter boundary and treat incompatible external facts conservatively.
- Marker-only resources no longer establish a current Plan. → Make the break explicit; perform no silent fallback or mutation.
- Representation and Snapshot paths can diverge. → Produce one call-local narrative outcome consumed by both.
- Removing payload code can erase edge coverage accidentally. → Replace each relevant behavior with native grammar and compatibility matrices before deletion.
- Live Milestone #3 can drift. → Separate deterministic fixture acceptance from current external evidence; drift does not count as a pass.

## Migration / Rollback

No GitHub resource is mutated or backfilled by this Change.

| Compatibility window | Validation | Failure handling | Rollback or forward-fix strategy | Operational owner | Approval evidence |
|---|---|---|---|---|---|
| Before any native-only Request Plan is externally applied | Full tests and fixture verification | Do not publish failing artifacts | Code and specs can be reverted together | Change owner | Human DesignDoc approval before construction |
| Existing legacy plus native resources | Compatibility matrix | Native facts win; preamble ignored | Revert only before depending on native-only observation | Host/operator | Recorded test evidence |
| Existing marker-only resources | Observation returns unavailable content-backed facts | No automatic mutation or fallback | External Actor may add native narrative under separate authority; Arcloom does not backfill | External resource owner | Breaking behavior accepted with this Change |
| After a native-only Request Plan has been externally applied | Live observation and evidence record | Old code cannot establish its content-backed facts | Approved forward fix is restore the native observer; do not reintroduce hidden payload storage | Host/operator | Release and live-proof record |

`PRODUCT.md` remains unchanged. `ARCHITECTURE.md` replaces the GitHub Adapter's `payload versions` detail with native narrative grammar, framing, and legacy-content compatibility while retaining the same boundary and dependency direction.

## Open Questions

None.
