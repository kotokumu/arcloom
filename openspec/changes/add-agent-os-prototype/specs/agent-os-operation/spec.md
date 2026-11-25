## Purpose

Let an operator run, inspect and resume replaceable business applications with explicit authority, durable evidence and conservative handling of interrupted external actions within one local owner scope.

## ADDED Requirements

### Requirement: replaceable-business-progression

The Agent OS MUST execute application-owned business transitions and resolve open capability invocations from explicit provider and dependency declarations without embedding a provider-specific or business-specific workflow.

- **Preconditions**: The operator supplies an application and driver configuration.
- **Input and Acceptance**: Missing or ambiguous capability selection is rejected before work starts.
- **Behavioral Rules**: The application determines its next state and whether to wait, complete or request a named ability through a valid staged ticket. Providers receive opaque application input and explicit invocation metadata, without unconditional context injection. The kernel does not determine business success criteria or retry policy, or classify an invocation into fixed execution/evaluation slots.
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

The Agent OS MUST require explicit grants for each requested root and provider dependency capability and MUST confine requested evidence to the configured operator and current-Signal scopes.

- **Preconditions**: The run has an immutable configuration binding and one operator-owned state directory.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Granted dispatch | Selected provider satisfies capability | Exact requested capability is explicitly granted | Dispatch is eligible subject to the remaining limits | Authorized action may be recorded and invoked |
  | Missing grant | Selected provider satisfies capability | Requested capability is absent from grants | Permission refusal | No process or action-result artifact |
  | Invalid provider | Missing or ambiguous provider selection | Any grant set | Configuration rejection | No external dispatch |

- **Invariants**: There are no default grants. Context never crosses owner state directories. Applications receive their own run Signal and request knowledge or artifact evidence through separately granted services, with no arbitrary filesystem or CAS query access. Applications cannot commit Intent or write Knowledge in this increment. Native command drivers remain explicitly trusted host programs, not sandboxed principals.
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
  | No pending action for transition | Application requests capability invocation | Ticket valid, authorized, within budget and resource available | Pending action | Next application state, selected Mailbox acknowledgement, resource reservation and pending action become durable together before dispatch |
  | No pending action for transition | Application requests capability invocation | Required resource already reserved | Unchanged | No committed transition, acknowledgement or new action; input remains available |
  | Pending action | Driver returns an outcome | Corresponding run and action match | Completed action | Immutable result artifact and correlated completion Signal are recorded; only the matching reservation is released |
  | Pending action | Operator resumes after interruption | No completion recorded | Uncertain action | Stop automatic work and expose uncertainty without replay; retain a reconciling reservation |
  | Completed action | Operator resumes | Completion Signal not yet consumed | Completed action | Deliver persisted result for subsequent application progress without rerunning the driver |
  | Uncertain action | Operator resumes | Existing uncertain record | Uncertain action | Preserve uncertainty without dispatch |

  Transitions not listed here confer no dispatch authority.
- **Invariants**: Consumed input and committed application state cannot disagree after restart. Recorded action results remain correlated to their originating run, action, generation and reservation. Staged guest resources are not durable effects until the reservation transition commits. Durable records do not establish exactly-once external effects or authoritative external business state.
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

- **Preconditions**: A consumer has a subscription filter and an explicit delivery mode. The application runner consumes this Mailbox for its actual transitions, including configured coalescing selections.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Queue | Unacknowledged matching Signals exist | Consumer requests queue delivery | Oldest matching occurrence is delivered | Acknowledgement advances only that consumer's progress |
  | Coalescing | Explicitly selected replaceable notifications exist | Consumer requests coalescing delivery | Latest matching occurrence for the selected kind and target represents that pending selection | Acknowledgement covers only that delivered/subsumed selection for that consumer |
  | No match | No unacknowledged matching Signal | Consumer receives | No delivery | Other subscriptions are unchanged |

- **Invariants**: Delivery does not mutate the Signal log. Coalescing never implicitly replaces queue delivery. Acknowledgement for one consumer does not acknowledge another consumer's events. An interleaved queued event or a newer notification arriving during delivery remains available after a coalesced acknowledgement.
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

- **Preconditions**: Knowledge ingestion is an operator operation. Draft knowledge requires finite expiry; source references identify artifacts. A guest knowledge query requires the explicit knowledge-read grant.
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

The Agent OS MUST reserve the actual invocation resource atomically with accepted dispatch progress and reject stale release or completion against newer ownership.

- **Preconditions**: A lease request names a resource key and holder and supplies a finite future expiry.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Available resource | No protected reservation | Valid dispatch reservation | Grant ownership with a new fencing token | Atomically record Lease, pending action, next application state and selected Mailbox acknowledgement |
  | Held resource | Active unexpired reservation or pending/uncertain associated action | Competing dispatch reservation | Refuse ownership | Existing ownership, application state and input acknowledgement unchanged |
  | Expired resource | Existing expiry is at or before current time and no unresolved external action protects it | Acquire request | Grant new ownership with a newer token | Replace expired ownership |
  | Uncertain outcome | Pending external outcome becomes uncertain | Recovery or failed dispatch | Reservation becomes reconciling | Expiry does not permit reuse |
  | Operator reconciliation | Reservation is reconciling | Explicit operator release decision | Resource becomes available | Preserve historical uncertainty evidence |
  | Current completion | Action, generation, holder and token match current ownership | Confirmed completion | Release succeeds | Record result and Signal together with resource release |
  | Stale release | Holder or token does not match current ownership | Release request | Release refused | Current ownership unchanged |

- **Invariants**: One resource has at most one current local owner. Pending or uncertain effects prevent resource reuse even after nominal expiry until explicitly reconciled. A fencing token distinguishes successive ownership grants; this prototype does not claim that external systems enforce it.
- **Concurrency and Idempotency**: Competing acquisitions cannot both establish current ownership. Repeating a stale release cannot clear newer ownership.
- **Failure Handling**: Expiry or stale ownership cannot be treated as authority to release another holder's lease.
- **References**: [decision] `openspec/changes/add-agent-os-prototype/design.md`.

#### Scenario: Holder acquires and releases an available resource [happy]

- **GIVEN** an application requests an authorized capability whose resource is available
- **WHEN** its transition reserves and subsequently completes that invocation
- **THEN** ownership and input progress commit together, and exact completion records its result and releases ownership

#### Scenario: Stale holder cannot release a replacement lease [concurrency]

- **GIVEN** a first holder's lease expired and a second holder acquired a newer lease
- **WHEN** the first holder releases using its old token
- **THEN** release is refused and the second holder retains ownership

#### Scenario: Expiry makes resource available at the boundary [boundary]

- **GIVEN** a resource lease expires at the current time without a pending or uncertain external action
- **WHEN** a new holder requests ownership
- **THEN** acquisition may succeed with a newer fencing token

#### Scenario: Expiry cannot duplicate uncertain work [boundary]

- **GIVEN** an application's external outcome is uncertain and its resource reservation is reconciling
- **WHEN** another invocation requests that resource after the nominal lease expiry
- **THEN** reservation is refused until explicit operator reconciliation, and the new input remains unacknowledged

### Requirement: scoped-capability-composition

The Agent OS MUST resolve explicitly declared provider dependencies without granting an application or provider access to undeclared abilities.

- **Preconditions**: The operator declares application requirements, capability providers, provider requirements and explicit grants.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Resolved graph | Each required ability has one provider; graph is acyclic | Authorized root invokes only declared dependencies | Request reaches selected implementations | Only scoped invocations may start |
  | Missing dependency | Required ability has no provider | Startup | Reject configuration | No external work |
  | Ambiguous dependency | Required ability has multiple providers | Startup | Reject configuration | No external work |
  | Cyclic dependency | Provider requirements form a cycle | Startup | Reject configuration | No external work |
  | Incompatible resource scope | Dependency edge has a different effective resource than its consumer | Startup | Reject configuration | No external work |
  | Undeclared or ungranted dependency | Provider attempts an ability outside its declared scope or explicit grants | Invocation | Refuse dependency access | No unauthorized dependency work |

- **Invariants**: Application roots are limited to their own declared requirements. Each provider sees only its declared dependencies. Transitive resolution never creates a grant. Kernel policy is independent of whether the implementation is a command, direct computation, delegation or another conforming provider. One composed invocation uses one effective resource and reservation; all dependency edges must agree on that resource. A provider's explicit resource key applies when present; otherwise the invoked capability is the resource key. The guest cannot override the selected resource.
- **Failure Handling**: Resolution or authority failure cannot silently select another provider or broaden scope.
- **References**: [[agent-os-operation/explicit-dispatch-authority]]; [[agent-os-operation/replaceable-business-progression]].

#### Scenario: Delegated capability reaches its declared implementation [happy]

- **GIVEN** an operator configures a root provider that requires a distinct provided capability, explicitly grants both and assigns the same resource key to both providers
- **WHEN** the application invokes the root and it invokes that declared dependency
- **THEN** the dependency receives the request without kernel business or provider branching

#### Scenario: Missing dependency grant prevents delegation [permission]

- **GIVEN** an operator grants the root capability but omits its dependency grant
- **WHEN** the root would require that dependency
- **THEN** the OS refuses unauthorized work without invoking the dependency

#### Scenario: Cyclic composition is rejected before work [error]

- **GIVEN** an operator configures two providers requiring each other's abilities
- **WHEN** the operator starts work
- **THEN** resolution rejects the cycle before an external effect starts

#### Scenario: Dependency cannot escape its reserved resource [error]

- **GIVEN** an operator configures a dependency with a resource key different from its consumer's
- **WHEN** the operator starts work
- **THEN** configuration is rejected before either provider runs

### Requirement: requested-guest-services

The Agent OS MUST provide typed guest service and outcome contracts for requested evidence and staged invocation, and MUST commit no staged effect from an unsuccessful callback.

- **Preconditions**: The application is executing a callback for its own delivered Signal.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Knowledge requested | Explicit knowledge-read grant | Guest queries knowledge | Return owner-scoped nonexpired knowledge in separate confirmed/draft collections | No mutation |
  | Artifact requested | Explicit artifact-read grant and requested identity equals current Signal artifact | Guest reads artifact | Return verified bytes | No mutation |
  | Evidence denied | Required grant or evidence scope missing | Guest requests evidence | Return typed refusal | No evidence disclosed |
  | Invocation staged | Guest requests a permitted capability | Guest acquires lease proposal and stages dispatch | Return callback-local resources | No durable Lease, acknowledgement or external effect yet |
  | Valid invocation returned | Guest returns a valid ticket in a successful transition | Callback completes | Invocation is eligible for atomic reservation | Only the returned invocation may commit |
  | Invalid callback | Trap, invalid result or invalid resource | Callback fails | Reject proposed transition | No staged dispatch, durable Lease or acknowledgement |

- **Invariants**: OS-owned signal structure, service outcomes and resource lifetimes use the typed contract; opaque business content does not define OS authority. Context is not injected into every guest callback or provider invocation. Guest resources cannot be reused across callbacks. Guest artifact or knowledge mutation remains unavailable in this increment.
- **Concurrency and Idempotency**: Staging is not dispatch and does not reserve a durable resource. Reservation rechecks actual resource availability before committing progress.
- **Failure Handling**: Invalid guest handles and insufficient grants fail closed without durable effects.
- **References**: [interface] `agent-os/wit/business-app.wit`; [[agent-os-operation/filtered-knowledge-context]]; [[agent-os-operation/immutable-artifact-evidence]]; [[agent-os-operation/local-resource-leases]].

#### Scenario: Guest requests only the knowledge it needs [happy]

- **GIVEN** an application has a knowledge-read grant and receives a Signal without unsolicited context
- **WHEN** it requests knowledge through the imported service
- **THEN** it receives eligible owner-scoped confirmed and draft knowledge separately

#### Scenario: Artifact scope cannot expand to another run's evidence [permission]

- **GIVEN** an application has an artifact-read grant but requests an artifact other than the current Signal's artifact
- **WHEN** it calls the artifact-read service
- **THEN** it receives refusal and no bytes from that artifact

#### Scenario: Trap after staging commits no invocation [error]

- **GIVEN** a guest has staged a dispatch ticket during its callback
- **WHEN** it traps before returning a valid transition
- **THEN** no invocation or durable Lease is created, and the input remains unacknowledged

### Requirement: pending-invocation-handoff

The Agent OS kernel MUST return submitted pending work to its host without waiting for the external transport to finish, and MUST associate completion with the recorded action and resource reservation.

- **Preconditions**: An authorized invocation has durably reserved its resource and committed its transition.
- **Behavioral Rules**:

  | Current state | Trigger or event | Guard | Next state | Output or Side Effects |
  |---|---|---|---|---|
  | Reserved action | Dispatch accepts submission | Recorded invocation identity matches | Pending operation | Return pending handle to host without collecting completion |
  | Pending operation | Host collects completed result | Action, generation and reservation match | Completed action | Record evidence and Signal and release matching reservation |
  | Pending operation | Host cannot establish outcome | Dispatch error or interruption | Uncertain action | Preserve reconciling resource protection and inspection evidence |
  | Any action | Late, repeated or mismatched completion | Recorded identity or current reservation does not match | Unchanged | Reject completion without releasing another holder |

  Unlisted transitions confer no authority to repeat an external action.
- **Invariants**: The host chooses when to collect pending work. Transport workers and blocking collection remain outside kernel decision execution. A pending handle does not replace durable action identity or authorize a second submission after restart.
- **Failure Handling**: Uncertain completion is not converted into confirmed failure, success or automatic retry.
- **References**: [[agent-os-operation/durable-action-outcomes]]; [[agent-os-operation/local-resource-leases]].

#### Scenario: Slow provider returns pending control to the host [happy]

- **GIVEN** an application invokes a provider whose external operation has not completed
- **WHEN** the kernel submits its reserved invocation
- **THEN** the host receives pending work before completion and may collect it outside the kernel

#### Scenario: Stale completion cannot release current ownership [concurrency]

- **GIVEN** a host holds an old pending handle whose reservation is no longer current
- **WHEN** it submits that handle's completion
- **THEN** the completion is rejected without recording it against another action or releasing current ownership
