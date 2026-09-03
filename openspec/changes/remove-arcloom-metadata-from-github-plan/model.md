# Specification Analysis: remove-arcloom-metadata-from-github-plan

## 1. Boundary

| Item | Decision | Evidence |
|---|---|---|
| Product capabilities | Existing `github-plan-creation-dry-run` and `github-plan-representation-observation` capabilities | These capabilities already own creation request content and GitHub-to-Plan fact establishment. |
| Change classification | Breaking observable behavior change | Creation bytes, representability, stored-content compatibility, and observed fact sources change. |
| Risk | High | Persisted external representations and public creation and observation contracts change. |
| Included behavior | Native narrative creation, native narrative observation, representability, localized malformed-content outcomes, and legacy compatibility | Proposal Scope and SC-1 through SC-5. |
| Excluded behavior | External mutation, backfill, Authorization, application, another representation, general Markdown interpretation, and Plan validity changes | Proposal Out of Scope. |

---

## 2. Consumers and Observable Events

| Consumer / actor | Trigger / prior state | Interaction or event | Observable result |
|---|---|---|---|
| Creation Request Plan caller | One valid Plan, Repository, and explicit Milestone or Issue representation exist | Requests a creation dry-run | Receives a deterministic request plan containing only the canonical native narrative, or an unsupported-representation result and no plan. |
| GitHub Plan observation caller | One exact GitHub Plan Target is bound | Requests a current observation | Receives native Plan facts, localized unavailable information or known Plan violations, or the caller lifecycle outcome before success. |
| Existing GitHub resource owner | A description or body contains legacy metadata, native narrative, both, or neither | The resource is observed | Native narrative alone supplies content-backed facts; legacy metadata supplies none. |
| Verification operator | A versioned Milestone #3 fixture and the current live target are available | Runs deterministic and live verification | Receives a deterministic fixture result and separate evidence that the current live target either satisfies or fails the native representation contract. |

---

## 3. Conceptual Model

| Concept | Meaning | Identity / relevant values | Owner capability and rationale |
|---|---|---|---|
| Plan | Existing provider-independent planning meaning | Exact name, Goal, ordered Acceptance Conditions, ordered Tasks, and optional Target Date | `plan`; it owns validity independently of Provider representation limits. |
| GitHub Plan Representation | Existing explicit Milestone or Issue mapping plus the decision whether one valid Plan is representable without native structural ambiguity | Milestone or Issue; supported or unsupported for one Plan | `github-plan-creation-dry-run`; it owns selection and GitHub representability. |
| GitHub Plan Narrative | Exact human-readable raw UTF-8 content that represents Goal, ordered Acceptance Conditions, and, for Issue, optional Target Date | Canonical exact-LF structure when created; structurally established or unavailable when observed | `github-plan-creation-dry-run` defines represented form; `github-plan-representation-observation` references that form and owns fact establishment from current external content. |
| Acceptance Condition Sequence Establishment | Ordered native narrative members established from a contiguous ordinal prefix | Exact members plus Complete or Incomplete membership | `github-plan-representation-observation`; it owns the distinction between a known prefix, a complete collection, and an unestablished remainder. |
| Observed GitHub Fact | Existing localized Plan meaning established from native fields, relationships, or the GitHub Plan Narrative | Available exact value, known Plan violation, authoritative absence where defined, or Unavailable; collections also carry Complete or Incomplete | `github-plan-representation-observation`; it owns GitHub fact-source authority and mapping into provider-independent Observation vocabulary. |
| Creation Request Plan | Existing passive deterministic GitHub operation description | Immutable ordered requests and typed dependencies | `github-plan-creation-dry-run`; narrative format does not change request-topology ownership. |
| GitHub Plan Target | Existing immutable Repository, representation, and positive Resource Number binding | One exact bound external subject | `github-plan-representation-observation`; it owns observation target acceptance. |
| Task Collection Establishment | Existing coherent native Task resources and membership state | Exact members plus Complete or Incomplete membership | `github-plan-representation-observation`; task pagination, identity, and conflict behavior remain unchanged. |
| GitHub Observation Outcome | Existing result of one read-only request | Valid Observation or caller cancellation or deadline before success | `github-plan-representation-observation`; it owns request lifecycle and failure containment, not fact-source precedence. |

### 3-1. GitHub Plan Narrative

Creation uses exact UTF-8 bytes without normalization. Structural bytes use LF.

| Representation | Canonical bytes |
|---|---|
| Milestone or undated Issue | `"## Goal\n\n" + G + "\n\n## Acceptance Conditions" + Σ("\n\n### " + decimal(i) + "\n\n" + Cᵢ) + "\n"` |
| Dated Issue | `"## Goal\n\n" + G + "\n\n## Acceptance Conditions" + Σ("\n\n### " + decimal(i) + "\n\n" + Cᵢ) + "\n\n## Target Date\n\n" + D + "\n"` |

`i` starts at 1 and is contiguous. A complete Plan always supplies at least one Acceptance Condition. Parsing removes only the literal framing bytes. Every additional byte, including LF, CR, Unicode, and surrounding whitespace, remains part of the represented value.

A valid Plan is unsupported by a GitHub Plan Representation when a Goal or Acceptance Condition contains a full LF-delimited line equal to `## Goal`, `## Acceptance Conditions`, or `## Target Date`, or when an Acceptance Condition contains a full line beginning `### `. Unsupported representation is not Plan invalidity.

### 3-2. Narrative Fact Establishment

| Established source shape | Goal | Acceptance Conditions | Issue Target Date |
|---|---|---|---|
| Root content unavailable | Unavailable | Incomplete, empty | Unavailable |
| Required Goal or Acceptance H2 inventory, order, or exact framing fails | Unavailable | Incomplete, empty | Unavailable |
| Required H2 structure is established | Exact framed value classified by Plan rules | Establish members under section 3-3 | Establish under section 3-4 |

The required H2 structure contains exactly one `## Goal` followed by exactly one `## Acceptance Conditions`. A parent Issue contains zero or one `## Target Date` after Acceptance Conditions. Duplicate or out-of-order Target Date headings make the H2 structure fail. A Milestone admits no Target Date heading in its narrative. Reserved H2 lines within a value therefore make the structure unavailable.

Preamble before the unique Goal heading supplies no Plan fact and contains no reserved H2 line. Legacy `arcloom-plan:v1` content has no special status, authority, or fallback behavior. Marker-only content cannot establish narrative-backed facts. Removing nonfact preamble leaves an otherwise equal Observation unchanged.

### 3-3. Acceptance Condition Sequence Establishment

| Source after the Acceptance Conditions heading | Established members | Membership |
|---|---|---|
| Canonical `### 1` through `### n` members and canonical EOF or Issue Target Date boundary | Every exact member | Complete |
| Canonical zero-member EOF form `## Acceptance Conditions\n` | None | Complete |
| Canonical zero-member Issue date boundary | None | Complete |
| First `### ` line has a wrong, duplicate, gapped, out-of-order, zero, nonnumeric, or leading-zero ordinal | Longest preceding fully framed contiguous prefix | Incomplete |
| Member delimiter, final EOF, or Issue date boundary framing is malformed | Members completed before that boundary | Incomplete |

Lines that do not begin exact `### ` and non-reserved headings remain member text. Establishment never resumes after the first member defect.

### 3-4. Issue Target Date Establishment

| Native date section | Target Date meaning |
|---|---|
| No Target Date heading under established H2 structure | Absent |
| One canonically framed section with valid `YYYY-MM-DD` | Present exact value |
| One canonically framed section with empty, whitespace, noncanonical, impossible, or multiline text | Known `InvalidTargetDate` Plan violation |
| Unique correctly ordered heading with malformed local opening or final framing | Unavailable |
| Duplicate or out-of-order heading | H2 structural failure under section 3-2 |

Milestone Target Date continues to come only from native `due_on`.

### 3-5. Relationships and Invariants

| Concept A | Concept B | Cardinality / direction | Lifecycle ownership | Consistency constraint |
|---|---|---|---|---|
| Plan | GitHub Plan Representation | One Plan to one explicitly selected representation per dry-run | Caller selects; Plan remains provider-independent | Representation is never inferred. |
| GitHub Plan Representation | GitHub Plan Narrative | One root request to one exact narrative | Request Plan owns passive output; GitHub owns persisted content | Accepted creation output is observable by the same native grammar. |
| GitHub Plan Narrative | Acceptance Condition Sequence Establishment | One narrative to one sequence result | Disposable per observation | Members are exactly the longest fully framed contiguous prefix. |
| Legacy content | Observed GitHub Fact | No fact relationship | GitHub owns stored bytes | It cannot override, supplement, or recover native facts. |
| GitHub Plan Target | Observed GitHub Fact | One target to one disposable fact set per invocation | Target binding outlives each fact set | Calls and targets never mix. |
| Observed GitHub Fact | Provider-independent Observation | One fact set to one Observation | Caller owns the returned value; GitHub remains authoritative | Provider DTOs, identity, errors, and source syntax do not cross the boundary. |

### 3-6. Minimality Check

| Candidate | Semantic remove / merge test | Decision | Reason |
|---|---|---|---|
| Versioned Plan Narrative | Remove payload authority and retain the native representation meaning | Remove and replace | The version and payload no longer exist. |
| GitHub Plan Narrative | Merge into primitive description/body strings | Keep | Grammar, representability, round-trip, and producer-observer agreement would lose one owner. |
| Acceptance Condition Sequence Establishment | Merge into generic Observed GitHub Facts | Keep | Ordinal-prefix continuity and completeness differ from Plan collection validity and from Task establishment. |
| Parser, decoder, producer, codec, or format strategy | Give parsing mechanics an independent concept | Reject | They are procedures with no independent identity, lifecycle, or authority. |
| General Markdown document | Broaden the closed grammar | Reject | General Markdown interpretation is outside scope. |
| Legacy payload | Preserve a compatibility concept | Reject | It has no authority, fallback, state, or lifecycle. |
| Representability Policy | Split from GitHub Plan Representation | Merge | The representation has the Plan values and authority required for the support decision. |
| Milestone #3 | Treat the verification target as a concept | Reject | It is versioned evidence, not reusable product meaning. |

### 3-7. Main Spec Conceptual Model Replacements

#### `github-plan-creation-dry-run`

Replace `GitHub Plan Representation` with:

> A GitHub Plan Representation is exactly Milestone or Issue. It is selected explicitly and is never inferred from Plan content. A selected representation supports one valid Plan only when the exact Goal and Acceptance Condition text does not collide with the GitHub Plan Narrative structural lines. Unsupported GitHub representation identifies the representation input and returns no Creation Request Plan. Representability is a GitHub constraint and does not change provider-independent Plan validity.

Replace `Versioned Plan Narrative` with `GitHub Plan Narrative`:

> A GitHub Plan Narrative is the exact human-readable UTF-8 representation of Goal and ordered Acceptance Conditions in a Milestone description or parent Issue body, and of an optional Target Date in an Issue body. Creation emits no preamble or opaque block. It starts with `## Goal`, the exact Goal, `## Acceptance Conditions`, and contiguous `### 1` through `### n` members with exact Acceptance Condition values. A dated Issue then contains `## Target Date` and the canonical date. Structural lines and framing use the exact LF form defined by `[[github-plan-creation-dry-run/github-plan-narrative]]`; values remain exact and are not normalized.
>
> A Goal or Acceptance Condition collides when one full LF-delimited line equals `## Goal`, `## Acceptance Conditions`, or `## Target Date`; an Acceptance Condition also collides when one full line begins `### `. Colliding text is not represented through escaping or opaque metadata. It makes the selected GitHub Plan Representation unsupported without changing Plan validity.

#### `github-plan-representation-observation`

Replace `Observed GitHub Facts` with:

> An Observed GitHub Fact is meaning established from native GitHub fields, relationships, or the exact raw UTF-8 GitHub Plan Narrative grammar. Milestone native facts provide Plan name, Target Date, and Task membership; its native narrative provides Goal and Acceptance Conditions. Issue native facts provide Plan name and Task membership; its native narrative provides Goal, Acceptance Conditions, and Target Date. Legacy marker or payload text is generic nonfact preamble. It is neither authoritative nor a fallback, and conflicting payload content is ignored. Marker-only content leaves narrative-backed facts unavailable.
>
> Required Goal and Acceptance Conditions H2 inventory, order, and exact framing are global structural constraints. Their failure makes Goal Unavailable, Acceptance Conditions Incomplete and empty, and Issue Target Date Unavailable while independently established name, Tasks, and Milestone Target Date remain usable. With established global structure, exact values are classified under the `plan` capability. A known value that violates Plan meaning becomes a Plan Validation Violation. An Issue Target Date is Absent only when its section is authoritatively omitted, Present when exact framed text is valid, and Unavailable when its local boundary or final framing cannot be established.

Add:

> Acceptance Condition Sequence Establishment preserves the longest fully framed contiguous sequence of exact `### 1` through `### n` members. Ordinals start at 1, have no leading zero, and remain contiguous in source order. The first wrong ordinal or member-framing defect ends establishment and makes membership Incomplete without discarding the preceding prefix or resuming later. Membership is Complete only when the declared EOF or Issue Target Date boundary establishes the end of the sequence; defined zero-member forms can therefore be Complete and empty. Each established member retains its exact text and is classified under `plan` rules.

---

## 4. Requirement Candidates

| Requirement slug | Actor and event | Guarantee | Concepts used | Normative representations | Important scenario classes |
|---|---|---|---|---|---|
| `explicit-github-target-and-representation` | Creation caller requests a dry-run | Accept only a valid and natively representable Plan for the selected representation | Plan, GitHub Plan Representation | Decision table | happy, error, compatibility |
| `github-plan-narrative` | Creation caller produces a request plan | Emit deterministic exact native narrative without metadata | GitHub Plan Narrative | Literal grammar, invariant | happy, boundary, idempotency |
| `milestone-creation-request-plan` | Creation caller selects Milestone | Place native narrative and existing Task/date inputs in the request plan | Creation Request Plan, GitHub Plan Narrative | Prose | happy, boundary |
| `issue-creation-request-plan` | Creation caller selects Issue | Place native narrative, optional date, and existing Task relationships in the request plan | Creation Request Plan, GitHub Plan Narrative | Partition table, prose | happy, boundary |
| `validation-result` | Creation input is unrepresentable | Return stable unsupported representation and no partial plan | GitHub Plan Representation | Decision table | error |
| `native-narrative-meaning` | Observation caller reads current root content | Establish narrative-backed facts, malformed-content outcomes, and legacy compatibility | GitHub Plan Narrative, Acceptance Condition Sequence Establishment, Observed GitHub Fact | Decision tables, invariants | happy, error, boundary, compatibility |
| `milestone-representation-meaning` | Observation caller reads a Milestone | Map native title, narrative, date, and Issue membership | Observed GitHub Fact | Prose | happy, boundary |
| `issue-representation-meaning` | Observation caller reads a parent Issue | Map native title, narrative, date, and Sub-issues | Observed GitHub Fact | Prose | happy, error, boundary |
| `conservative-failure-observation` | A fact cannot be established | Preserve independent facts and expose no Provider failure detail | Observed GitHub Fact, GitHub Observation Outcome | Decision table, invariant | error, boundary |

---

## 5. Independent Evolution Scenarios and Responsibility Impact

| Scenario / confidence | Primary decision owner | Expected propagation | Verdict |
|---|---|---|---|
| Section names, order, addition, or removal / Evidence-backed plausible | GitHub Plan Narrative | Creation, observation, specs, fixtures, and grammar tests | Pass |
| Reserved-collision support changes / Evidence-backed plausible | GitHub Plan Representation | Narrative collision classification, validation, and boundary tests | Pass |
| Acceptance numbering style changes / Evidence-backed plausible | GitHub Plan Narrative and Acceptance Condition Sequence Establishment | Lexical recognition, sequence rules, and tests | Pass |
| Malformed-content tolerance changes / Evidence-backed plausible | GitHub Plan Narrative | Fact localization and malformed-content tests | Pass |
| Target Date source or precision changes / Evidence-backed plausible | Observed GitHub Facts | Representation mapping and tests; Plan validity remains independent | Pass |
| Consumers require richer fact states / Evidence-backed plausible | Provider-independent Observation | Provider mapping, reconciliation, and tests | Pass |
| Legacy metadata policy or migration changes / Evidence-backed plausible | Observed GitHub Facts | Compatibility rules, fixtures, and tests | Pass |
| Task membership or ordering changes / Evidence-backed plausible | Task Collection Establishment | Collection behavior and mapping tests | Pass |
| Milestone #3 drifts / Evidence-backed plausible | External GitHub facts | Fresh verification evidence only | Pass |
| Issue #50 expectations change / Evidence-backed plausible | Requirement owner, then the affected concept | Affected specification, behavior, and tests | Pass |
| A second Provider uses different native concepts / Evidence-backed plausible | That Provider adapter | Provider mapping and contract tests; Plan meaning unchanged | Pass |
| GitHub REST shapes, text serialization, limits, relationships, pagination, access, failures, or cancellation change / Evidence-backed plausible | GitHub Plan Adapter and the applicable existing fact-establishment owner | Provider boundary, observation behavior, and tests | Pass |

Speculative metadata restoration, marker-only recovery, Plan-validity coupling, mutation or backfill, stateful observation, and atomic observation are risks only. They do not justify a current concept or extension point.

---

## 6. Unresolved Decisions

None.

---

## 7. Sources

Evidence packet: `GH-NATIVE-PLAN-2026-09-02-v1`

| Fact | Source | Relevance |
|---|---|---|
| Issue #50 requires removal of metadata storage, native observation, explicit compatibility, and Milestone #3 proof. | GitHub `kotokumu/arcloom` Issue #50, fetched 2026-09-02 | Change authority and acceptance boundary. |
| Plan is provider-independent and external systems retain authority. | `PRODUCT.md` sections 4.1, 7.3, and 7.7 | Product boundary. |
| The GitHub Plan Adapter maps external facts and confines Provider details. | `ARCHITECTURE.md` sections 5.3, 5.5, and 6 | Component and dependency boundary. |
| Plan Text, collections, and Target Date preserve exact provider-independent meaning. | `openspec/specs/plan/spec.md` | Value and validity owner. |
| Creation currently owns payload-backed narrative, request topology, integrity, and dry-run behavior. | `openspec/specs/github-plan-creation-dry-run/spec.md` | Affected creation guarantees. |
| Observation currently owns payload mapping, localization, Task completeness, and stateless failure isolation. | `openspec/specs/github-plan-representation-observation/spec.md` | Affected and retained observation guarantees. |
| Production mapping and codec depend on payload creation and decoding. | `providers/github/plan/scheme.go`, `payload.go`, `payload_decode.go`, `milestone.go`, `issue.go` | Current implementation evidence. |
| Literal tests establish payload, native text, and response-shape behavior. | `providers/github/plan/payload_producer_test.go`, `payload_observation_test.go`, `native_text_observation_test.go`, `response_shape_observation_test.go` | Current executable evidence. |
| Milestone #3 contains native Goal and five numbered Acceptance Conditions, no marker, and open Tasks #47, #48, #57, and #58. | GitHub REST capture for `kotokumu/arcloom` Milestone #3, fetched 2026-09-02 | Live acceptance target and future versioned fixture. |
| Payload previously protected exact reversibility and no-payload resources left content-backed facts unavailable. | Archived change `2026-08-29-add-observable-github-plan-representation`, design sections 4.5 and 5 | Compatibility and regression evidence. |
