## 1. Plan and Snapshot Meaning

- [x] 1.1 Add Plan semantic equality through table-driven tests and implementation, and verify `go test -race ./plan` passes.
- [x] 1.2 Implement immutable Task Progress and Progress Evidence values, including duplicate-name preservation, completeness, accessors, and defensive copies, and verify `go test -race ./plansnapshot` passes.
- [x] 1.3 Implement Snapshot eligibility, Observation Failure, and the consumer-owned Observer boundary, and verify every Snapshot outcome-classification row plus cancellation and concurrency with `go test -race ./plansnapshot`.

## 2. GitHub Milestone Snapshot

- [ ] 2.1 Extend the private GitHub Milestone representation mapping to establish current native state and member progress from one observation, and verify mapping tests cover open, closed, unknown, duplicate-name, and incomplete membership cases.
- [ ] 2.2 Implement the GitHub Milestone Snapshot Observer against the `plansnapshot` Port, and verify exact target binding, complete current Plan reconstruction, partial progress, pagination, and provider-failure isolation with `go test -race ./githubplan`.
- [ ] 2.3 Verify repeated and concurrent GitHub observations use current facts without mixed target, response, cancellation, or Provider detail, and make `go test -race ./githubplan ./plansnapshot` pass.

## 3. Generic Authorization

- [ ] 3.1 Implement immutable non-empty Policy and Rule evaluation for exact consumer-established subjects, and verify the deny-overrides, all-permit, and otherwise-undecidable matrix with `go test -race ./authorization`.
- [ ] 3.2 Implement subject-bound Evaluation and verify semantic subject immutability for the Evaluation lifetime, missing or failed evidence, cancellation, and repeated or concurrent Policy evaluation with `go test -race ./authorization`.

## 4. Plan Application Request

- [ ] 4.1 Implement stable Target Reference and exact Plan Revision values with public immutable accessors, and verify invalid, equal, and exact-value cases with `go test -race ./planapplication`.
- [ ] 4.2 Implement passive Request, Request Receipt Evidence, and Result value contracts, and verify all authorization and receipt outcomes remain inspectable without exposing target state.
- [ ] 4.3 Implement the invocation-local Application Attempt and stable invalid-input failures, and verify every Plan Application precedence row, Actor cancellation LSP behavior, exact subject-bound Authorization, at-most-once Actor invocation, and no retry with `go test -race ./planapplication`.

## 5. Codex Plan Control Adapter

- [ ] 5.1 Implement cohesive validated Model, Reasoning Effort, Working Directory, and bounded Shutdown Grace values; keep executable selection and read-only/no-approval constraints fixed inside the adapter; and verify invalid inputs start no interaction.
- [ ] 5.2 Implement one disposable Codex app-server assessment and verify through a capturing helper that every Plan element and caller-owned observation value reaches the boundary semantically unchanged and maps to the existing four Plan Control outcomes.
- [ ] 5.3 Fail closed according to the Codex failure-classification table without exposing Provider details, and verify every protocol-drift and FailureCode case with `go test -race ./codexplancontrol`.
- [ ] 5.4 Implement bounded caller cancellation and concurrent-assessment isolation, and verify only observable material/result isolation and an unaffected concurrent assessment with `go test -race ./codexplancontrol`.

## 6. Structural and Repository Verification

- [ ] 6.1 Review imports and public contracts to verify provider-independent Packages depend only inward, Provider DTOs and errors do not cross Ports, Provider Modules do not implement application Ports, and no workflow, runtime, orchestrator, persistence, utility, helper, or common package was introduced.
- [ ] 6.2 Run `go mod tidy -diff`, `go test -v -race ./...`, `golangci-lint run`, `openspec validate --specs --strict`, and `openspec validate --changes --strict`, and resolve every failure.

## 7. One Real Plan Proof

- [ ] 7.1 Select one real GitHub Milestone target, establish and record its initial current Snapshot, and obtain a Codex Revise assessment for the exact Plan and caller-owned observation material, satisfying CA-1 and CA-2.
- [ ] 7.2 Establish current externally authoritative authorization evidence for the exact Revision, obtain ReceiptAcknowledged from one external Actor interaction, and record both references, satisfying CA-3.
- [ ] 7.3 Re-observe the target, establish the resulting current Plan independently, record named Goal and acceptance-condition evidence, obtain a Codex Complete assessment, and verify the disposable evidence record satisfies CA-4 through CA-6 without asserting causality or authority.
