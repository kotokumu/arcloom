## 1. Public Contract and First Behavior

- [x] 1.1 Add the `plancontrol` declarations from the approved DesignDoc without adding Provider, application, Change, Authorization, persistence, or `reconciliation` dependencies; verify the exported surface with `go doc ./plancontrol`.
- [x] 1.2 Run `gotests -w -all -use_go_cmp` for the public implementation file, preserve the generated table scaffold, and add a failing public `Assess` case for one valid current Plan, typed observations, and a Retain response; verify the Red failure is caused by missing behavior rather than a compile or test-data defect.
- [x] 1.3 Implement the smallest immutable Assessment and stateless `Assess` behavior that passes the Retain case; run the focused test with `go test -v ./plancontrol -run TestAssess`.

## 2. Supported Assessments and Plan Relationship

- [x] 2.1 Add failing Complete and InsufficientInformation cases, including all-Tasks-complete with missing outcome evidence and one obsolete incomplete Task with a Complete AI response; implement only the behavior needed to preserve the AI-owned judgment and pass the focused tests.
- [x] 2.2 Add failing Revise cases for every exact Plan difference in the DesignDoc: scalar values, collection values, collection order, collection cardinality, and optional target-date presence and value; implement exact Plan comparison through public `plan.Plan` accessors and pass the focused tests.
- [x] 2.3 Add a failing separately-constructed exact-equivalent Plan case; reject it with `AIContractFailure` and a publicly observable zero Assessment, then pass the focused test.
- [x] 2.4 Add the remaining failing response-form cases for absent, unknown, duplicate, multiple, contradictory, non-Revise-with-proposal, missing/multiple Revise proposals, and invalid zero proposed Plan values; implement deterministic response validation and pass the complete response matrix.

## 3. Input and External AI Boundary Contract

- [x] 3.1 Add failing input cases for nil context, pre-cancelled and pre-expired context, invalid current Plan, nil Assessor, and their simultaneous combinations; implement the approved precedence, assert zero Assessor calls, and pass the focused tests.
- [x] 3.2 Add failing Assessor error cases for untranslatable, wrapped and joined untranslatable, Provider unavailability, Provider-side timeout, unrelated context errors, ordinary Provider errors, and response-plus-error; implement stable Failure mapping without unwrapping Provider errors and pass the focused tests.
- [x] 3.3 Add channel-synchronized failing cases for cancellation and deadline expiry while the Assessor is blocked, cancellation combined with valid response, Provider error, or untranslatable response, and cancellation after success; implement both approved context samples and pass the focused tests without asserting private call order.
- [x] 3.4 Verify through public behavior that schedule/progress, coverage, and Task-granularity observation values reach their typed Assessor unchanged and that Plan Control introduces no fixed Observation type or semantic reinterpretation.

## 4. Immutability, Statelessness, and Concurrency

- [x] 4.1 Add failing public-accessor tests proving Outcome, assessed Plan, optional proposed Plan, and the zero Assessment contract; snapshot successful Assessments so post-return mutation of Assessor response slices cannot change them.
- [x] 4.2 Add successive-call and discard-and-repeat tests proving no prior Assessment, Failure, observation, or AI session state becomes authoritative input to another call.
- [x] 4.3 Add correlation-value and barrier-based concurrent cases proving per-call Plan, observation, response, and Assessment isolation; run `go test -v -race ./plancontrol` and remove all shared per-call state or races.

## 5. Conformance and Review

- [x] 5.1 Run `go test -v -race ./...`, `go vet ./...`, `golangci-lint run`, and `go mod tidy -diff`; resolve every failure without weakening assertions or expanding the approved design.
- [x] 5.2 Capture `go doc ./plancontrol`, `go list -deps ./plancontrol`, and source-import evidence that the exported API and dependencies contain no external application, Authorization, Change, persistence, Task execution, Delivery Acceptance, Provider SDK/DTO/error, `planrepresentation`, or universal Reconciliation result contract.
- [x] 5.3 Obtain post-implementation architecture, SOLID, interface, procedural-code, code-quality, and Go test-specification reviews; resolve every Blocking and High finding and record any consciously accepted lower-severity risk.
- [x] 5.4 Run `openspec validate add-plan-control --strict` and `git diff --check`, then confirm every PLC-1 through PLC-8 accepted scenario is backed by the public tests or the static evidence required by the DesignDoc.
