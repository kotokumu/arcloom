## Why

Arcloom currently verifies the consistency of a Plan with its external representation, but it does not control the Plan itself toward completion using progress and other delivery observations. Plan Control must remain independent of the experimental Change vocabulary and must not reduce AI control to a fixed set of Change-shaped outcomes.

## What Changes

- Add Plan Control that uses the current Plan and delivery observations to obtain an AI assessment and any resulting Plan adjustment needed to control the Plan toward completion.
- Clarify in `PRODUCT.md` and `ARCHITECTURE.md` that Plan Control is the primary Plan reconciliation behavior, while Plan Representation Reconciliation verifies an external representation and does not control the Plan.
- Keep Change, Authorization, field-specific AI control policy, concrete external application, and Provider-specific mutations outside this capability. Plan Control defines no Change Concept or dependency and does not require a closed product-wide taxonomy of AI outcomes.

## Capabilities

### New Capabilities

- `plan-control`: Reconciliation of a current Plan and delivery observations so an external AI can control the Plan toward completion.

### Modified Capabilities

None.

## Impact

- `PRODUCT.md` Plan and Delivery Reconciliation descriptions
- `ARCHITECTURE.md` Plan Controller, Plan Representation Controller, AI Agent Provider Module, and their dependency boundaries
- New Go contracts and behavior for Plan Control only
- No dependency on Change or Authorization; no external Provider mutation, persistence, AI model implementation, or authoritative Arcloom state
