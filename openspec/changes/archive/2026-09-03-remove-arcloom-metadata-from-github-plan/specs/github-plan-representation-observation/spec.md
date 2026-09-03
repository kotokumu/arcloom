## ADDED Requirements

### Requirement: native-narrative-meaning

GitHub Plan observation MUST establish narrative-backed Plan facts only from the exact raw UTF-8 GitHub Plan Narrative grammar and MUST treat legacy or conflicting Arcloom metadata as nonfact content rather than an authority or fallback.

- **Input and Acceptance**: Observation reads the current raw Milestone description or parent Issue body. Structural markers are exact LF-delimited full lines. The required H2 structure is one `## Goal` followed by one `## Acceptance Conditions`; a parent Issue may contain one `## Target Date` only after Acceptance Conditions, while a Milestone admits no narrative Target Date section. Optional preamble before the unique Goal heading supplies no Plan fact and contains no reserved H2 line.
- **Behavioral Rules**:

  | Global narrative state | Goal | Acceptance Conditions | Issue Target Date | Independent facts |
  |---|---|---|---|---|
  | Root content unavailable | Unavailable | Incomplete and empty | Unavailable | Preserve established name, Tasks, and Milestone Target Date. |
  | Required Goal or Acceptance H2 is missing, duplicated, out of order, or not canonically framed | Unavailable | Incomplete and empty | Unavailable | Preserve established name, Tasks, and Milestone Target Date. |
  | Issue Target Date H2 is duplicated or precedes Acceptance Conditions, or a Milestone contains that H2 | Unavailable | Incomplete and empty | Unavailable | Preserve established name, Tasks, and Milestone Target Date. |
  | Required H2 structure is established | Classify the exact framed Goal under `plan` rules | Apply Acceptance Condition establishment below | Apply Issue Target Date establishment below | Preserve every independently established fact. |

  Goal framing is exact `## Goal\n\n`, followed by the exact Goal and `\n\n## Acceptance Conditions`. The Acceptance Conditions heading then uses the canonical member, EOF, or Issue Target Date boundary below.

  | Acceptance Conditions source | Established members | Membership |
  |---|---|---|
  | Exact `\n\n### 1\n\n` through `\n\n### n\n\n` members followed by canonical EOF or Issue Target Date boundary | Every exact member in ordinal order | Complete |
  | Exact zero-member EOF form ending `## Acceptance Conditions\n` | None | Complete |
  | Exact zero-member Issue Target Date boundary | None | Complete |
  | First full line beginning `### ` has a wrong, duplicate, gapped, out-of-order, zero, nonnumeric, or leading-zero ordinal | Longest preceding fully framed contiguous prefix from 1 | Incomplete |
  | A member delimiter, final EOF, or Issue Target Date boundary has altered or missing framing | Members completed before that boundary | Incomplete |

  A line that does not begin exact `### ` and a non-reserved heading remain part of the current member. Establishment stops after the first member ordinal or framing defect and does not use later bytes as member facts. Each established member is classified independently under `plan` rules.

  | Issue Target Date source | Target Date meaning |
  |---|---|
  | No Target Date H2 under established global structure | Known Absent |
  | One exact `\n\n## Target Date\n\n` section containing a valid `YYYY-MM-DD` value and one final framing LF | Present exact value |
  | One canonically framed section containing empty, whitespace, noncanonical, impossible, or multiline text | Known `InvalidTargetDate` Plan Validation Violation |
  | One correctly ordered H2 with malformed local opening or final framing | Unavailable |

  A malformed local Target Date boundary also leaves Acceptance Conditions Incomplete when that boundary cannot establish the end of the collection. Extra value bytes are never normalized or discarded as Plan text; only the exact declared framing bytes are excluded from established values.

  | Compatibility input | Narrative-backed result |
  |---|---|
  | Valid native narrative without legacy metadata | Use the native narrative. |
  | Nonfact preamble, including a legacy `arcloom-plan:v1` block, followed by valid native narrative | Ignore the preamble and use the native narrative. |
  | Legacy marker or payload content without the required native narrative | Goal Unavailable, Acceptance Conditions Incomplete and empty, and Issue Target Date Unavailable. |
  | Legacy payload values conflict with valid native narrative | Ignore the payload values and use only the native narrative. |
  | Nonfact preamble is removed while native narrative is unchanged | Produce the same narrative-backed facts. |

- **Invariants**: Legacy or opaque content never supplies, overrides, supplements, or recovers a Plan fact. A known exact native value that violates Plan meaning becomes a Plan Validation Violation. A structurally unestablished value becomes Unavailable rather than a guessed value, authoritative absence, or violation. Accepted creation output from `[[github-plan-creation-dry-run/github-plan-narrative]]` establishes the same exact Plan values.
- **Failure Handling**: Structural and local failures produce the tabled localized Observation states without exposing content, parser detail, or Provider failure detail outside the Observation contract.
- **References**: [related] `[[github-plan-creation-dry-run/github-plan-narrative]]` for canonical native representation; [related] `plan` Conceptual Model for exact Plan Text, collection validity, and Target Date; [related] `plan-representation-reconciliation` Conceptual Model for Observation and Unavailable Information.

#### Scenario: Native Milestone narrative is established [happy]

- **GIVEN** a Milestone description contains one canonically framed Goal and five contiguous numbered Acceptance Conditions without metadata
- **WHEN** GitHub Plan observation interprets the current content
- **THEN** the exact Goal and five ordered Acceptance Conditions are available

#### Scenario: Canonical creation output round-trips [happy]

- **GIVEN** a Creation Request Plan contains an accepted GitHub Plan Narrative
- **WHEN** the same exact native content is observed for the selected representation
- **THEN** observation recovers every narrative-backed Plan value exactly

#### Scenario: Legacy marker precedes native narrative [compatibility]

- **GIVEN** a legacy metadata block precedes a valid native narrative and its encoded values conflict with the narrative
- **WHEN** GitHub Plan observation interprets the content
- **THEN** only the native Goal, Acceptance Conditions, and applicable Issue Target Date are established

#### Scenario: Legacy marker is the only content [compatibility]

- **GIVEN** the root content contains a legacy metadata block but no required native Goal and Acceptance Conditions structure
- **WHEN** GitHub Plan observation interprets the content
- **THEN** Goal is Unavailable, Acceptance Conditions are Incomplete and empty, and Issue Target Date is Unavailable while independent native facts remain usable

#### Scenario: Required H2 is duplicated inside value text [error]

- **GIVEN** raw content contains a second reserved Goal or Acceptance Conditions H2 line within otherwise readable text
- **WHEN** GitHub Plan observation interprets the content
- **THEN** narrative-backed facts have the global structurally unavailable states while independent native facts remain usable

#### Scenario: Acceptance numbering becomes malformed [boundary]

- **GIVEN** the first two members are canonically framed as `### 1` and `### 2` and the next member is `### 4`
- **WHEN** GitHub Plan observation establishes Acceptance Conditions
- **THEN** the first two exact members are preserved and membership is Incomplete

#### Scenario: Acceptance Conditions are explicitly empty [boundary]

- **GIVEN** established Goal and Acceptance Conditions headings end with the declared zero-member EOF form
- **WHEN** GitHub Plan observation establishes Acceptance Conditions
- **THEN** membership is Complete and empty, producing the applicable Plan collection violation when a complete current Plan is evaluated

#### Scenario: Issue Target Date text is invalid [error]

- **GIVEN** an otherwise valid Issue narrative has a canonically framed Target Date value `2027-02-29`
- **WHEN** GitHub Plan observation classifies Target Date
- **THEN** the Target Date location contains the known `InvalidTargetDate` violation rather than Unavailable Information or absence

#### Scenario: Issue Target Date final framing is absent [error]

- **GIVEN** an otherwise valid Issue narrative has one correctly ordered Target Date heading and value but no final framing LF
- **WHEN** GitHub Plan observation interprets the content
- **THEN** Target Date is Unavailable and the already established Goal remains usable

## MODIFIED Requirements

### Requirement: milestone-representation-meaning

GitHub Plan observation MUST project a Milestone representation into the Plan meaning assigned by the Conceptual Model.

- **Behavioral Rules**: Milestone title supplies Plan name; its GitHub Plan Narrative supplies Goal and Acceptance Conditions; its native target date supplies present or absent Target Date; and the Complete assigned-Issue collection supplies Tasks. Open and closed Issues are included and Pull Requests are excluded. Narrative failure does not suppress independently established name, Target Date, or Task facts.
- **Failure Handling**: A present native target date that cannot represent a valid Plan date becomes a known target-date violation. Absence remains known absence. Narrative-backed facts follow `[[github-plan-representation-observation/native-narrative-meaning]]`.
- **References**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model; [related] `[[github-plan-representation-observation/native-narrative-meaning]]`.

#### Scenario: Complete Milestone representation is observed [happy]

- **GIVEN** all native facts required by the selected Milestone representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning without requiring Arcloom-specific metadata

#### Scenario: Closed Issue and Pull Request are assigned [boundary]

- **GIVEN** one closed Issue and one Pull Request are assigned to the bound Milestone
- **WHEN** GitHub Plan observation establishes Task membership
- **THEN** the closed Issue is an observed Task and the Pull Request is not

#### Scenario: Milestone target date is absent or invalid [boundary]

- **GIVEN** the Milestone native target date is absent or cannot represent a valid Plan date
- **WHEN** GitHub Plan observation classifies Target Date meaning
- **THEN** the Observation distinguishes known absence from a known Plan Validation Violation

### Requirement: issue-representation-meaning

GitHub Plan observation MUST project a parent Issue representation into the Plan meaning assigned by the Conceptual Model.

- **Behavioral Rules**: Parent Issue title supplies Plan name; its GitHub Plan Narrative supplies Goal, Acceptance Conditions, and optional Target Date; and the Complete Sub-issue collection supplies Tasks. A completely established empty Sub-issue collection is known Complete and empty. Narrative failure does not suppress independently established name or Task facts.
- **Failure Handling**: A selected parent resource that is a Pull Request makes the Plan root Unavailable and contributes no descendant state. Narrative-backed facts follow `[[github-plan-representation-observation/native-narrative-meaning]]`.
- **References**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model; [related] `[[github-plan-representation-observation/native-narrative-meaning]]`.

#### Scenario: Complete Issue representation is observed [happy]

- **GIVEN** all native facts required by the selected parent Issue representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning without requiring Arcloom-specific metadata

#### Scenario: Parent is a Pull Request [error]

- **GIVEN** the selected parent resource is a Pull Request
- **WHEN** a caller requests observation
- **THEN** the Plan root is Unavailable and no descendant state is asserted

### Requirement: conservative-failure-observation

A GitHub Plan observation caller MUST receive unavailable GitHub facts as localized Unavailable Information without Provider detail, except for its own cancellation or deadline outcome before success.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Root unavailable | No coherent root fact is established | GitHub cannot establish the root | Plan root is Unavailable; no descendant state is asserted | No authoritative absence is claimed. |
  | Localized unavailable | Coherent facts exist outside one affected location | One native field, relationship, or narrative fact cannot be established | Only the locations affected under their owning rules are Unavailable or Incomplete | Other coherent facts remain usable. |
  | Partial collection | Some coherent members are established | Complete membership cannot be established | Members remain observable and membership is Incomplete | Omitted members do not become authoritative absence. |
  | Non-authoritative absence | GitHub does not unambiguously establish absence | A resource or narrative fact cannot be selected conclusively | The affected location is Unavailable | No authoritative Plan absence is claimed. |

- **Invariants**: Provider failure detail never crosses the observation boundary, and inability to establish a fact never becomes authoritative absence, opaque recovery, or a fabricated Plan violation.
- **Side Effects**: Target binding remains unchanged. Provider errors, response content, credentials, rate-limit information, request identifiers, redirect targets, and target identity do not enter the Observation.
- **Failure Handling**: Caller cancellation or deadline expiration before success returns the supplied caller-lifecycle outcome and no successful Observation. No other Provider failure crosses the observation boundary.
- **References**: [related] `plan-representation-reconciliation` Conceptual Model for localized Unavailable Information and covering rules; [related] `[[github-plan-representation-observation/native-narrative-meaning]]` for narrative-specific localization.

#### Scenario: Root facts are inaccessible [error]

- **GIVEN** GitHub cannot establish the selected root representation
- **WHEN** GitHub Plan observation completes
- **THEN** the Plan root is Unavailable and no Provider failure detail crosses the boundary

#### Scenario: Narrative is malformed but title and Tasks are coherent [boundary]

- **GIVEN** the selected root title and complete Task collection are established but required narrative H2 structure is not
- **WHEN** GitHub Plan observation completes
- **THEN** name and Tasks remain usable while narrative-backed locations follow their unavailable or incomplete states

#### Scenario: GitHub does not establish authoritative absence [boundary]

- **GIVEN** a GitHub outcome may represent either absent or inaccessible data
- **WHEN** GitHub Plan observation classifies the affected fact
- **THEN** the affected location is Unavailable rather than authoritatively absent

#### Scenario: Caller cancels observation [error]

- **GIVEN** no successful Observation has been established
- **WHEN** the caller cancels or its deadline expires
- **THEN** the supplied caller-lifecycle outcome is returned and no successful Observation is returned

## REMOVED Requirements

### Requirement: versioned-payload-meaning

- **Reason**: Legacy payload bytes are no longer an authoritative or fallback source of observed Plan meaning.
- **Migration**: Use `[[github-plan-representation-observation/native-narrative-meaning]]`. Marker-only stored content leaves narrative-backed facts unavailable until a human-readable native narrative exists through an external change outside this capability.
