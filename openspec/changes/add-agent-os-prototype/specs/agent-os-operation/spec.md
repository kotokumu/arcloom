## Purpose

Let an operator run, inspect and resume replaceable business applications with explicit authority, durable evidence and conservative handling of interrupted external actions within one local owner scope.

## ADDED Requirements

### Requirement: replaceable-business-progression

The Agent OS MUST execute application-owned business transitions and select execution and feedback providers from explicit capability declarations without embedding a provider-specific or business-specific workflow.

- **Preconditions**: The operator supplies an application and driver configuration.
- **Input and Acceptance**: Missing or ambiguous capability selection is rejected before work starts.
- **Behavioral Rules**: The application determines its next state and whether to wait, complete or request execution or feedback. Drivers receive opaque application payloads, the run Intent and filtered context. The host does not determine business success criteria or retry policy.
- **Invariants**: Replacing the application changes its business progression without changing host business rules. Replacing a driver preserves the application's consumer contract. No default provider or fixed execution/evaluation sequence is imposed.
- **Failure Handling**: Invalid application output cannot dispatch an external action.
- **References**: [decision] `openspec/changes/add-agent-os-prototype/design.md`; [external] `docs/design/agent-os-v0.1.original.md`.

#### Scenario: Different applications determine different progressions [happy]

- **GIVEN** an operator has two applications, one requesting feedback after execution and one completing without an external action
- **WHEN** the operator starts each with its declared dependencies
- **THEN** each follows its own progression without a host workflow change

#### Scenario: Execution provider is replaced [compatibility]

- **GIVEN** an operator has two drivers satisfying the same requested capability
- **WHEN** the operator selects the second driver for a new run
- **THEN** the same application's request reaches the selected driver without a provider-specific application branch

#### Scenario: Ambiguous provider configuration is rejected [error]

- **GIVEN** an operator configures multiple providers without an unambiguous selection for a required capability
- **WHEN** the operator starts work
- **THEN** startup rejects the configuration before any driver performs work

### Requirement: explicit-dispatch-authority

The Agent OS MUST require an explicit grant for each requested driver capability and MUST confine supplied application context to the configured operator scope.

- **Preconditions**: The run has an immutable configuration binding and one operator-owned state directory.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Granted dispatch | Selected provider satisfies capability | Exact requested capability is explicitly granted | Dispatch is eligible subject to the remaining limits | Authorized action may be recorded and invoked |
  | Missing grant | Selected provider satisfies capability | Requested capability is absent from grants | Permission refusal | No process or action-result artifact |
  | Invalid provider | Missing or ambiguous provider selection | Any grant set | Configuration rejection | No external dispatch |

- **Invariants**: There are no default grants. Context never crosses owner state directories. Applications receive their own run Intent and Signal and the owner's filtered context, with no arbitrary filesystem, artifact or knowledge query access. Applications cannot commit Intent or write Knowledge in this increment. Native command drivers remain explicitly trusted host programs, not sandboxed principals.
- **Concurrency and Idempotency**: Grants remain bound for the run; changing them requires a new run rather than live revocation.
- **Failure Handling**: Unavailable authority does not become permission.
- **References**: [[agent-os-operation/filtered-knowledge-context]]; [decision] `openspec/changes/add-agent-os-prototype/design.md`.

#### Scenario: Explicit grant permits selected driver [happy]

- **GIVEN** an application requests a capability provided by its selected driver and the operator granted that exact capability
- **WHEN** the OS considers dispatch within the run limits
- **THEN** it permits that driver request

#### Scenario: Missing grant prevents dispatch [permission]

- **GIVEN** an application requests a driver capability absent from its grants
- **WHEN** the OS considers the request
- **THEN** it reports refusal without starting the driver or creating an action-result artifact

#### Scenario: Context belongs to one operator scope [permission]

- **GIVEN** an operator has separate state directories containing different knowledge
- **WHEN** an application runs in the first directory
- **THEN** its supplied context contains no knowledge from the second directory

### Requirement: durable-action-outcomes

The Agent OS MUST preserve committed application progress and action outcomes across restart without automatically repeating an action whose external outcome is uncertain.

- **Input and Acceptance**: Resume requires the same application content and configuration identities as the existing run; a mismatch is rejected.
- **Behavioral Rules**:

  | Current state | Trigger or event | Guard | Next state | Output or Side Effects |
  |---|---|---|---|---|
  | No pending action for transition | Application requests external action | Output valid, authorized and within budget | Pending action | Next application state, consumed input acknowledgement and pending action become durable together before dispatch |
  | Pending action | Driver returns an outcome | Corresponding run and action match | Completed action | Immutable result artifact and correlated completion Signal are recorded |
  | Pending action | Operator resumes after interruption | No completion recorded | Uncertain action | Stop automatic work and expose uncertainty without replay |
  | Completed action | Operator resumes | Completion Signal not yet consumed | Completed action | Deliver persisted result for subsequent application progress without rerunning the driver |
  | Uncertain action | Operator resumes | Existing uncertain record | Uncertain action | Preserve uncertainty without dispatch |

  Transitions not listed here confer no dispatch authority.
- **Invariants**: Consumed input and committed application state cannot disagree after restart. Recorded action results remain correlated to their originating run and action. Durable records do not establish exactly-once external effects or authoritative external business state.
- **Concurrency and Idempotency**: One local runner owns a state directory at a time; a competing runner cannot start work there. Resume never replays a completed or uncertain external action.
- **Failure Handling**: An interrupted pending action is not reported as confirmed success or confirmed failure. Inspection exposes its uncertain disposition.
- **References**: [[agent-os-operation/immutable-artifact-evidence]]; [[agent-os-operation/independent-signal-delivery]].

#### Scenario: Committed result survives restart [happy]

- **GIVEN** an operator's run has a recorded driver result whose Signal has not been consumed
- **WHEN** the operator resumes the run with the same binding
- **THEN** the application receives that recorded result without invoking the driver again

#### Scenario: Interrupted action is not replayed [idempotency]

- **GIVEN** an operator's run was interrupted after recording a pending action and before recording its completion
- **WHEN** the operator resumes the run
- **THEN** inspection reports uncertainty and no automatic repeat of that external action occurs

#### Scenario: Changed application cannot resume historical state [compatibility]

- **GIVEN** an operator replaces the application's content after a run was created
- **WHEN** the operator tries to resume that historical run with the replacement
- **THEN** the OS rejects the mismatched binding without external dispatch

#### Scenario: Competing runner cannot own the same state [concurrency]

- **GIVEN** one operator process is running work in a state directory
- **WHEN** a second process tries to run work in that directory
- **THEN** the second process fails to acquire runner ownership and performs no work there

### Requirement: bounded-local-work

The Agent OS MUST enforce configured application resource limits, external-command limits and a finite accepted-action budget.

- **Preconditions**: The operator supplies valid finite limits for application memory and computation, command duration and output bytes, and an action budget.
- **Behavioral Rules**:

  | Partition | Condition or range | Acceptance or result |
  |---|---|---|
  | Budget available | Accepted actions are fewer than the budget | An otherwise valid and authorized action may be accepted |
  | Budget exhausted | Accepted actions equal or exceed the budget | No further action; persist terminal budget_hit disposition and Signal |
  | Command within bounds | Command finishes before its deadline and accepted output bytes do not exceed the configured bound | Process its result |
  | Command limit exceeded | Deadline is reached while command runs, or output exceeds its bound | Terminate the local command group and expose terminal uncertainty |

- **Invariants**: Application resource exhaustion and invalid output cannot authorize external dispatch. The application has no ambient host capabilities. Limits constrain local work, not the external business effects already performed by a trusted driver.
- **Side Effects**: Driver failure records terminal uncertainty and a stall Signal for operator inspection without another application callback or automatic retry. Command termination kills the local process group and reaps the direct child before reporting command handling complete; grandchild reaping belongs to their parent or init.
- **Failure Handling**: Application resource exhaustion, timeout and output overflow produce observable failures, not business success.
- **References**: [[agent-os-operation/explicit-dispatch-authority]]; [[agent-os-operation/durable-action-outcomes]].

#### Scenario: Work within configured limits returns its result [happy]

- **GIVEN** an application has remaining action budget and its selected driver finishes within its configured duration and output limits
- **WHEN** the application requests that authorized action
- **THEN** its correlated result becomes available for the next application transition

#### Scenario: Exhausted budget stops further actions [boundary]

- **GIVEN** an application's accepted action count equals its configured budget
- **WHEN** it requests another action
- **THEN** no driver starts and inspection shows terminal budget_hit with its Signal

#### Scenario: Timed-out command is terminated [error]

- **GIVEN** an application's driver command continues running until its configured deadline
- **WHEN** the deadline is reached
- **THEN** the OS terminates the local command group, reaps the direct child and records terminal uncertainty

### Requirement: independent-signal-delivery

The Agent OS MUST preserve immutable Signals and independent durable acknowledgement progress for each consumer, with explicit queue or coalescing selection.

- **Preconditions**: A consumer has a subscription filter and an explicit delivery mode.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Queue | Unacknowledged matching Signals exist | Consumer requests queue delivery | Oldest matching occurrence is delivered | Acknowledgement advances only that consumer's progress |
  | Coalescing | Explicitly selected replaceable notifications exist | Consumer requests coalescing delivery | Latest matching occurrence represents the pending selection | Acknowledgement covers the coalesced selection for that consumer |
  | No match | No unacknowledged matching Signal | Consumer receives | No delivery | Other subscriptions are unchanged |

- **Invariants**: Delivery does not mutate the Signal log. Coalescing never implicitly replaces queue delivery. Acknowledgement for one consumer does not acknowledge another consumer's events.
- **Concurrency and Idempotency**: Acknowledged progress survives restart. An application transition and its input acknowledgement commit together.
- **Failure Handling**: An uncommitted acknowledgement cannot cause a Signal to disappear from the consumer's durable progress.
- **References**: [[agent-os-operation/durable-action-outcomes]].

#### Scenario: Queue preserves separate occurrences [happy]

- **GIVEN** a consumer has two matching unacknowledged Signals in queue mode
- **WHEN** it receives and acknowledges each in turn
- **THEN** it observes both in occurrence order

#### Scenario: Consumers progress independently [concurrency]

- **GIVEN** two consumers subscribe to the same Signal
- **WHEN** the first consumer acknowledges it
- **THEN** it remains available to the second consumer

#### Scenario: Coalescing selection survives restart [idempotency]

- **GIVEN** a consumer has acknowledged the latest occurrence of an explicitly coalesced selection
- **WHEN** the operator restarts and that consumer receives again without new matching Signals
- **THEN** the acknowledged selection is not delivered again

### Requirement: immutable-artifact-evidence

The Agent OS MUST preserve immutable content-addressed artifacts and verify their content identity when reading them.

- **Behavioral Rules**: Equal bytes identify the same content; stored content is not overwritten with different bytes under the same identity.
- **Invariants**: A successful read returns bytes matching the requested content identity. Artifact evidence remains separate from mutable knowledge interpretation.
- **Concurrency and Idempotency**: Repeating a put of identical bytes preserves their content identity.
- **Failure Handling**: Missing or corrupt bytes fail explicitly and are not returned as verified evidence.
- **References**: [decision] `openspec/changes/add-agent-os-prototype/design.md`.

#### Scenario: Stored content is reusable evidence [happy]

- **GIVEN** an operator has stored an artifact
- **WHEN** the operator puts the same bytes again and reads their identity
- **THEN** the identity is unchanged and the original bytes are returned

#### Scenario: Corrupted content is rejected [error]

- **GIVEN** an operator's artifact bytes no longer match their recorded identity
- **WHEN** the operator reads that artifact
- **THEN** the OS reports failed verification instead of returning verified evidence

### Requirement: filtered-knowledge-context

The Agent OS MUST exclude expired knowledge from active context and provide confirmed human/stable knowledge separately from provisional drafts within the run's owner scope.

- **Preconditions**: Knowledge ingestion is an operator operation. Draft knowledge requires finite expiry; source references identify artifacts.
- **Behavioral Rules**:

  | Partition | Condition or range | Acceptance or result |
  |---|---|---|
  | Expired | Knowledge expiry is at or before current time | Exclude from active context |
  | Unexpired | Knowledge expiry is after current time | Eligible for its separate confirmed or draft context category |
  | No expiry | Knowledge has no expiry | Eligible only where its category permits no expiry; drafts are rejected |

- **Invariants**: Human/stable and draft context remain physically separate supplied collections. No context query crosses owner state directories. Knowledge interpretation does not replace artifact evidence. Applications cannot ingest or promote knowledge in this increment.
- **Side Effects**: Expiry affects active-context eligibility without rewriting original artifact bytes.
- **Failure Handling**: Invalid knowledge or invalid source references are rejected rather than silently promoted to trusted context.
- **References**: [[agent-os-operation/explicit-dispatch-authority]]; [[agent-os-operation/immutable-artifact-evidence]].

#### Scenario: Confirmed context stays separate from drafts [happy]

- **GIVEN** an operator has ingested human/stable knowledge and an unexpired draft in the same owner scope
- **WHEN** the OS assembles context for work in that scope
- **THEN** the confirmed knowledge and draft appear in separate collections

#### Scenario: Knowledge expires at its deadline [boundary]

- **GIVEN** an operator has a draft whose expiry equals the current time
- **WHEN** the OS assembles active context
- **THEN** that draft is excluded

### Requirement: local-resource-leases

The Agent OS MUST provide time-limited exclusive local resource ownership and reject a stale holder's attempt to release newer ownership.

- **Preconditions**: A lease request names a resource key and holder and supplies a finite future expiry.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Available resource | No active unexpired lease | Acquire request | Grant ownership with a new fencing token | Record holder and expiry |
  | Held resource | Existing expiry is after current time | Competing acquire request | Refuse ownership | Existing ownership unchanged |
  | Expired resource | Existing expiry is at or before current time | Acquire request | Grant new ownership with a newer token | Replace expired ownership |
  | Current release | Holder and token match current ownership | Release request | Release succeeds | Resource becomes available |
  | Stale release | Holder or token does not match current ownership | Release request | Release refused | Current ownership unchanged |

- **Invariants**: One resource has at most one unexpired local owner. A fencing token distinguishes successive ownership grants; this prototype does not claim that external systems enforce it.
- **Concurrency and Idempotency**: Competing acquisitions cannot both establish current ownership. Repeating a stale release cannot clear newer ownership.
- **Failure Handling**: Expiry or stale ownership cannot be treated as authority to release another holder's lease.
- **References**: [decision] `openspec/changes/add-agent-os-prototype/design.md`.

#### Scenario: Holder acquires and releases an available resource [happy]

- **GIVEN** a resource user names an available resource and a future expiry
- **WHEN** it acquires and then releases with its current token
- **THEN** ownership is granted and subsequently released

#### Scenario: Stale holder cannot release a replacement lease [concurrency]

- **GIVEN** a first holder's lease expired and a second holder acquired a newer lease
- **WHEN** the first holder releases using its old token
- **THEN** release is refused and the second holder retains ownership

#### Scenario: Expiry makes resource available at the boundary [boundary]

- **GIVEN** a resource lease expires at the current time
- **WHEN** a new holder requests ownership
- **THEN** acquisition may succeed with a newer fencing token
