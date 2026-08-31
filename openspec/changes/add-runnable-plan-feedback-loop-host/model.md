# Specification Analysis: add-runnable-plan-feedback-loop-host

## 1. Boundary

| Item | Decision | Evidence |
|---|---|---|
| Product capability | Add `plan-feedback-loop-operation`. | The accepted Controller publishes in-process Reports but does not deliver target-semantic Results; the accepted Plan Attempt returns a Plan-owned result but does not operate a repeated caller-facing loop. |
| Change classification | Observable behavior change. | A Host can accept explicit Plan wake-ups, observe each resulting Report, and deliver assessed Plan Results through one caller-scoped operation. |
| Included behavior | One exact Plan target, ordinary identity-only Requests, Report consumption, Plan-specific Result delivery, destination failure, cancellation, deterministic multi-cycle verification, and real GitHub/Codex verification. | Proposal Intended Outcomes and SC-1 through SC-7. |
| Excluded behavior | Authorization, application, Task execution, external mutation, automatic retry or polling, durable state, generic cross-target Result contracts, and final Reconciliation remodelling. | Proposal Out of Scope; `PRODUCT.md`; `ARCHITECTURE.md`; Milestone #3 boundary. |

The new capability owns the observable relationship between a Host's explicit Plan wake-up and delivery of a semantic Plan Result. It does not take ownership of Request scheduling from `reconciliation-control-loop`, Plan evaluation from `plan-reconciliation-loop`, GitHub facts from `github-plan-snapshot`, or Codex judgment from `codex-plan-control-assessment`.

---

## 2. Consumers and Observable Events

| Consumer / actor | Trigger / prior state | Interaction or event | Observable result |
|---|---|---|---|
| Host operator | One complete operation configuration identifies one self-identifying Plan Target | Starts a Plan Feedback Loop Operation | The operation uses the exact identity established from that target, or rejects the configuration before accepting a wake-up. |
| Plan Trigger Source | The operation is Running | Supplies an Explicit Plan Trigger | The operation submits an ordinary Request containing only the configured Target Identity; Controller coalescing remains authoritative. |
| Plan Result Destination | A published Report contains Current Plan Assessed | Receives the exact Current Plan Assessed value | One delivery invocation succeeds, or the operation stops and exposes its delivery failure. |
| Host operator | A Report contains Current Plan Not Established or Attempt Failure | Observes operation progress | No Plan Result delivery is invoked. |
| External Actor | A cycle has exposed a Plan Control Assessment | Changes authoritative Planning Context outside Arcloom and later causes or precedes another explicit trigger | The later Attempt reacquires current facts without prior Report, Result, trigger payload, or delivery outcome as facts. |
| Host operator | Operation is Running or a delivery is in flight | Ends the Caller Lifecycle | Trigger intake stops and the operation reaches Stopped after active boundaries return. |
| Test author (implementation verification) | A deterministic local Planning Context and controlled collaborators exist | Drives an initial trigger and at least two later triggers while changing only external facts between cycles | The accepted operation contract is verified without adding a product concept or production Interface. |

---

## 3. Conceptual Model

| Concept | Meaning | Identity / relevant values | Owner capability and rationale |
|---|---|---|---|
| Plan Feedback Loop Operation | One caller-scoped, disposable operation that connects explicit Plan wake-ups, target-bound Controller Reports, and Plan Result delivery for one exact target. | One Target Identity; Configured, Running, Stopping, or Stopped. | `plan-feedback-loop-operation`; it owns this consumer-visible relationship, not the meaning of the reused stages. |
| Explicit Plan Trigger | A Host-owned occurrence meaning that the configured Plan target should be evaluated again. Its cause, payload, and count are not Attempt facts. | An occurrence for the operation's one configured target; no target-state payload. | `plan-feedback-loop-operation`; it defines what enters the operation before reduction to the existing Request. |
| Deliverable Plan Result | The exact Plan Control Assessment contained by the only Plan Attempt Result branch that establishes semantic Reconciliation. | The Assessment inside Current Plan Assessed: Complete, Retain, Revise, or Insufficient Information. | `plan-control` owns Assessment meaning; `plan-feedback-loop-operation` owns only the rule selecting it for delivery. |
| Plan Result Destination | The target-specific external recipient of the exact Plan Control Assessment selected as a Deliverable Plan Result. It is neither a generic Result abstraction nor an External Actor application authority. | Exactly one destination for the operation. | `plan-feedback-loop-operation`; it defines the recipient and its non-application boundary while `plan-control` retains Assessment meaning. |
| Plan Result Delivery | One invocation that offers the exact Plan Control Assessment contained by one published Current Plan Assessed Report to the configured Plan Result Destination. | Not Started, In Flight, Delivered, Delivery Failed, or Cancelled; at most one invocation per qualifying Report. | `plan-feedback-loop-operation`; its ordering, cardinality, and failure meaning are the new capability's contract. |

The operation reuses the exact meanings of Target Identity, Request, Controller, Completion, Attempt Failure, and Report from `reconciliation-control-loop`; Current Plan Not Established and Current Plan Assessed from `plan-reconciliation-loop`; GitHub Plan Snapshot from `github-plan-snapshot`; and Codex Assessment Interaction from `codex-plan-control-assessment`.

### 3-1. Operation and Delivery States

| State | Meaning |
|---|---|
| Configured | All required target, lifecycle, Controller, Trigger Source, Report-consumption, and Plan Result Destination relationships have been established and validated; no trigger is accepted. |
| Running | Explicit Plan Triggers may be accepted, Reports may be consumed, and qualifying Results may be delivered. |
| Stopping | Caller cancellation has committed; no new trigger is accepted and active Controller or destination work is being cancelled and awaited. |
| Stopped | Trigger intake, Controller work, Report consumption, and delivery work have ended; runtime state may be discarded. |

| Plan Attempt Report classification | Semantic delivery classification | Required delivery behavior |
|---|---|---|
| Completion containing Current Plan Assessed | Deliverable Plan Result | Invoke the destination with the exact result after Report publication commits. |
| Completion containing Current Plan Not Established | Observation-only success | Do not invoke the destination. |
| Attempt Failure | Operational failure | Do not invoke the destination. |

### 3-2. Structural Invariants

- One Plan Feedback Loop Operation is bound to exactly one valid Target Identity, one target-bound Plan Attempt, one Report stream consumer, and one Plan Result Destination.
- Every Explicit Plan Trigger is reduced to an ordinary Request containing only that exact Target Identity.
- Report publication and Plan Result Delivery are distinct: delivery begins only after publication of the corresponding Report commits.
- A Plan Result Delivery preserves the exact Plan Control Assessment and target association from Current Plan Assessed; the operation does not expose the enclosing Snapshot or reconstruct or reinterpret the Assessment.
- Each qualifying published Report causes at most one destination invocation. A failed invocation stops the operation and a cancelled invocation follows caller cancellation; neither creates an implicit retry. A trigger in a newly started operation begins ordinary fresh evaluation rather than retrying an earlier delivery.
- Runtime state is disposable and is not authoritative Planning Context, Observation, Reconciliation, Result, scheduling, or delivery state.
- Only an External Actor may change authoritative Planning Context between cycles. Neither the operation nor the destination is granted mutation authority.

---

## 4. Requirement Candidates

| Requirement slug | Actor and event | Guarantee | Concepts used | Normative representations | Important scenario classes |
|---|---|---|---|---|---|
| exact-plan-operation-configuration | Host starts an operation | Reject incomplete relationships before accepting triggers and use the identity established from one self-identifying target binding. | Plan Feedback Loop Operation, Target Identity, Plan Result Destination | Decision Table; Invariants | valid, invalid target, missing relationship |
| explicit-plan-trigger | Trigger Source supplies a trigger | Submit only the exact Target Identity through the ordinary Request contract; preserve Controller coalescing. | Explicit Plan Trigger, Request, Operation | State Transition Table; Invariants | initial, repeated, duplicate/active, cancellation race |
| assessed-plan-result-delivery | Operation consumes a published Report | Deliver only Current Plan Assessed, after publication, exactly as returned, with at most one invocation. | Deliverable Plan Result, Report, Plan Result Delivery | Decision Table; Invariants | assessed, not established, attempt failure, ordering |
| delivery-failure-and-reentry | Destination rejects, fails, or a later operation starts | Stop with an observable delivery failure, create no retry or mutation, and make any newly started operation evaluate fresh facts. | Plan Result Delivery, Explicit Plan Trigger | State Transition Table; Invariants | failure, uncertain external receipt, later recovery, cancellation |
| caller-scoped-operation-lifecycle | Caller cancels the operation | Stop intake and reach Stopped after active boundaries return, without treating cancellation as success. | Operation lifecycle, Caller Lifecycle, Delivery | State Transition Table | idle, active Attempt, in-flight delivery, publication race |

Deterministic local verification and the real GitHub/Codex proof remain required implementation evidence for these candidates. They do not add Requirements because they are verification environments for the same observable guarantees, not independently owned product behavior.

---

## 5. Unresolved Decisions

None.

The proposal decisions are resolved as follows:

- Only the exact Plan Control Assessment contained by Current Plan Assessed is a Deliverable Plan Result. The enclosing Snapshot remains Plan Attempt material. Current Plan Not Established and Attempt Failure remain observable Reports but cause no semantic delivery.
- Delivery is one invocation at most per qualifying published Report. A destination success establishes Delivered; an error establishes Delivery Failed and stops the operation with that distinct failure; caller cancellation before success establishes Cancelled and preserves the caller lifecycle outcome. Failure or cancellation creates no implicit retry and does not claim whether an external side effect occurred before the destination reported failure.
- The operation accepts explicit triggers only while Running, consumes each published Report as the sole operation consumer, performs qualifying delivery after publication, and stops intake when caller cancellation or delivery failure commits. Later triggers accepted by a Running operation, including a newly started operation after failure, use the ordinary fresh Request contract and never carry prior cycle material. Controller coalescing remains authoritative, so trigger occurrences are not promised a one-to-one Attempt count.

---

## 6. Sources

- `PRODUCT.md`
- `ARCHITECTURE.md`
- `openspec/specs/reconciliation-control-loop/spec.md`
- `openspec/specs/plan-reconciliation-loop/spec.md`
- `openspec/specs/github-plan-snapshot/spec.md`
- `openspec/specs/codex-plan-control-assessment/spec.md`
- Evidence packet `M2-HOST-REQ-v1`, derived from GitHub Milestone #2 and issues #44-#56
