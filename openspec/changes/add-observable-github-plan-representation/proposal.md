## Why

The Plan Representation Controller can reconcile a provider-independent Plan observation, but no GitHub Provider can reconstruct that observation from GitHub facts. The current human-readable GitHub narrative accepts arbitrary Plan text without escaping, so its Goal and acceptance conditions cannot be decoded unambiguously after publication.

## What Changes

- **BREAKING** Add a versioned, machine-readable Plan payload to GitHub Milestone descriptions and parent Issue bodies produced by the creation dry-run while retaining the human-readable narrative.
- Add read-only observation of a target-bound GitHub Milestone or parent Issue and translate the current GitHub facts into `planrepresentation.Observation`.
- Map inaccessible or incomplete facts, malformed machine-readable payloads, pagination, and detected incoherence to the existing provider-independent observation states. This GitHub contract does not claim an authoritative absence signal because GitHub can conceal inaccessible resources as not found.
- Keep external mutation, Change Authorization, persistence, GitHub Enterprise Server, and Host deployment outside this Change.

## Capabilities

### New Capabilities

- `github-plan-representation-observation`: Defines target binding, GitHub REST reads, decoding, coherence, and translation of GitHub Milestone and Issue representations into a Plan observation.

### Modified Capabilities

- `github-plan-creation-dry-run`: Adds the reversible machine-readable Plan payload required to observe a created GitHub representation without changing the dry-run's non-mutating boundary.

## Impact

- Affects the public GitHub creation request-plan output and adds a GitHub-specific implementation of the Plan Representation Controller's observation Port.
- Uses GitHub.com REST API contracts for Milestones, Issues, and Sub-issues.
- Preserves the existing `plan`, `reconciliation`, and `planrepresentation` ownership boundaries and adds no Arcloom-owned durable state.
