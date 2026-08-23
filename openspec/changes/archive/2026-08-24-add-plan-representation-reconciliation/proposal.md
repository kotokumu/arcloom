## Why

Arcloom can represent a Plan and preview its GitHub creation requests, but it cannot determine whether an externally observed Plan representation matches the expected Plan. Without this determination, the Delivery Feedback Loop cannot distinguish a satisfied representation from a known difference or missing observation.

## What Changes

- Add read-only reconciliation between an expected Plan and a provider-independent observation of its external representation.
- Distinguish known semantic differences from information that could not be observed.
- Derive the determination from evidence as satisfied, not satisfied, or undecidable.
- Define the observation boundary required by the Plan Representation Controller without adding a concrete Provider implementation.
- Keep Plan sufficiency, external mutation, Change Authorization, AI evaluation, orchestration, and persistence outside this Change.

## Capabilities

### New Capabilities

- `plan-representation-reconciliation`: Determines whether an observed external representation matches an expected Plan and reports known differences separately from unavailable information.

### Modified Capabilities

- `plan`: Exposes the existing Plan-name and collection-validation rules as reusable Plan-owned contracts so observation construction does not duplicate Plan invariants.

## Impact

- Adds the first concrete reconciliation behavior using the existing provider-independent Plan.
- Adds public contracts for a disposable observed representation, evidence, a reconciliation determination, and a consumer-owned observation boundary.
- Adds reusable Plan-name and collection-validation contracts without changing Plan validity semantics.
- Does not change `PRODUCT.md`, `ARCHITECTURE.md`, external systems, or existing Plan and GitHub dry-run behavior.
