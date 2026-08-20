# Arcloom Development Guidelines

## 1. Purpose and Scope

### 1.1 Purpose and Applicability

This document defines the criteria used for design, implementation, and review in Arcloom. The product document defines the value and capabilities provided by Arcloom, and the Architecture document defines system boundaries and Component responsibilities. This document defines how to design when those are changed or made concrete.

This document applies to development that adds or changes Concepts, responsibilities, state owners, Component boundaries, public contracts, or behavior observable from outside.

This document does not prescribe a specific programming language, framework, or deployment form.

### 1.2 Required before Starting Design

Before starting design, review this document. Review `PRODUCT.md` when the Change affects the product concept, scope, value, capabilities, or product principles. Review `ARCHITECTURE.md` when the Change affects system boundaries, Component responsibilities, ownership, dependency rules, Ports, or external Contexts. Design takes as input the rules in the documents selected for the Change.

During design, create or update a DesignDoc for each Change. Do not start implementation until the DesignDoc review is complete.

Do not omit a DesignDoc even for a small Change that follows an existing design. Record `N/A` with a reason for sections that do not apply. A separate DesignDoc is not required for Changes that do not involve implementation, such as wording corrections.

### 1.3 Responsibility of a DesignDoc

A DesignDoc records the design needed to implement one Change and reach a state where its functional and non-functional aspects can be verified. It includes applicable content among the Change purpose, out of scope, behavior, conceptual model, responsibility assignment, Package design, Interface design, test specification, and design decisions specific to the Change.

When a Change affects Architecture principles, system boundaries, Component responsibilities, ownership relationships, or dependency rules, record the decision and impact in the DesignDoc and update the Architecture document at the same time.

---

## 2. Basic Design Policy

### 2.1 Design Structure from Responsibilities and Behavior from Tests

Structural design defines Concepts, relationships, states, constraints, invariants, responsibilities, owners, boundaries, dependency directions, and contracts rather than processing procedures. Do not split user operations or processing flows directly into Components, Classes, or Interfaces.

Behavioral design defines externally observable rules, state changes, errors, and boundary conditions as tests. Tests do not make internal call order or private implementation structure part of the specification.

Implementation is the minimal representation that satisfies both structural and behavioral design.

### 2.2 Do Not Mix Modeling Levels

| Level | Defines | Criterion for determining a boundary |
|---|---|---|
| Concept | Meaning, state, relationships, and invariants in the problem domain | Whether it can be integrated with another Concept without losing meaning |
| Component | Related responsibilities, decisions, owned contracts, and protected boundaries | Whether responsibilities, decision owners, and protected constraints cohere as one boundary |
| Package | Public code surface and dependency direction that implement Component responsibilities | Whether hidden implementation and permitted dependencies align |
| Interface/Port | Public contract required by a Consumer | Whether there is a real Consumer and a concrete constraint that direct dependency cannot protect |
| Value/Function/Class | Implementation representation of a Concept or responsibility | Whether state, identity, lifecycle, invariant, or substitutability is required |

A Concept does not mean a Class. Capabilities, use cases, and processing steps do not imply one-to-one correspondence with Components.

`Reconciliation Module`, `Change Target Module`, and `Provider Context Module` in the Architecture document represent logical Component types. This document calls implementation units in source code Packages and distinguishes them from Modules in the Architecture.

### 2.3 Design Order

```mermaid
flowchart LR
    A[Requirements and constraints] --> B[Conceptual model]
    B --> C[Confirm Concept minimality]
    C --> D[Assign responsibilities]
    D --> E[Boundaries and dependency direction]
    E --> F[Interface and Port]
    F --> G[Behavioral tests]
    G --> H[Implementation with TDD]
    H --> I[Design conformance review]
```

When subsequent design reveals a need for a new Concept, responsibility, state owner, boundary, or public contract, do not add it at that point. Return to the corresponding earlier stage and design it there.

---

## 3. Conceptual Modeling

### 3.1 Conditions for Introducing a Concept

Introduce a Concept when it has a stable meaning based on current requirements, constraints, or observed facts and represents one of the following:

- A coherent rule, decision, or Policy
- Identity, lifecycle, or permissions for state
- The meaning of an invariant or Protocol
- A relationship that loses meaning when integrated

When introducing a Concept, show the supporting requirement, constraint, invariant, or external fact. Do not introduce a Concept solely for future extensibility.

### 3.2 Make Ownership of Concepts Explicit

The conceptual model makes at least the following explicit.

| Concept | Meaning | State represented | Owner of state authority/lifecycle | Behavior/decision | Constraint/invariant | Basis |
|---|---|---|---|---|---|---|
| Target Concept | What it represents in the problem domain | State used for decisions | Arcloom or an external system | What it does based on state and information | Rule that must always hold | Requirement, constraint, or external fact |

For relationships between Concepts, define direction, multiplicity, lifecycle owner, and consistency constraints. When an external system owns meaning, state, permissions, or lifecycle, do not re-own the representation inside Arcloom as authoritative state. State held internally is temporary state during execution or a Disposable Projection reconstructable from the external source of truth.

### 3.3 Confirm Concept Minimality

For each Concept, attempt to remove it or integrate it with another Concept. If meaning, state, identity, lifecycle, invariant, decision authority, or relationship meaning is not lost, do not make it an independent Concept.

Judge the existence of a Concept separately from its implementation representation. Even a meaningful Concept may not need a dedicated Class; it may be represented by a Value, Function, Package, or existing type.

### 3.4 Do Not Treat Procedures as Concepts

The following names or classifications alone do not justify a Concept:

- A processing order turned into a noun
- A list of capabilities or use cases
- `Manager`, `Processor`, `Handler`, `Resolver`, `Registry`, or `Factory`
- Generic `Execution`, `Runtime`, or `Service`
- A DTO that only carries data

Introducing any of these still requires the same conditions as any other Concept. Justify it by the meaning, state, invariants, decisions, or boundaries it owns, not by its name.

---

## 4. Responsibility Assignment Based on SOLID

### 4.1 Decide the Owner of a Responsibility First

Assign a responsibility to the Concept or Component that has the information and authority required for the decision and preserves the invariant through that decision. Use reasons for change to check responsibility cohesion after deciding the owner.

Responsibility assignment makes the following explicit.

| Responsibility/decision | Owner | Information and authority used | State/invariant preserved | Reason for change | Candidate excluded and reason |
|---|---|---|---|---|---|
| Target responsibility | Concept or Component | Facts and authority required for the decision | What the decision protects | Actor or factor that changes the rule | Candidate that does not own it and why |

A decision with no owner indicates a missing conceptual model or boundary. If multiple owners make the same decision, a Policy or invariant is duplicated.

### 4.2 Do Not Make SOLID the Purpose of Abstraction

Use SOLID to verify responsibilities and dependencies. Do not use it as a reason to add Classes or Interfaces.

| Principle | Application in Arcloom | Avoid |
|---|---|---|
| SRP | Group responsibilities that use the same information and authority and change for the same reason | Creating a Class for every operation or collecting decisions in a `Manager` |
| OCP | Enclose currently existing or well-founded changes within a boundary | Creating a Strategy, Factory, or Plugin when there is only one implementation |
| LSP | Implementations preserve preconditions, postconditions, results, and Error contracts | Requiring Consumers to branch specially for each implementation |
| ISP | Make Consumers depend only on the minimum contracts they use | Publishing a large Provider-centered Service or Repository |
| DIP | Separate higher-level decisions from Framework, SDK, DB, Filesystem, and other details | Exposing Provider-specific DTOs or Errors to the Core |

### 4.3 Verify Responsibilities with Independent Change Scenarios

For design that adds or changes a Concept, responsibility, state owner, or boundary, after the initial conceptual model and minimality check are complete, a person or Agent other than the model author identifies changes in requirements or external constraints. Do not give the scenario author the proposed Class, Interface, or Package structure as a premise.

Use observed, agreed, or well-founded Change scenarios to confirm the primary decision owner and Change propagation. Reconsider responsibility assignment when unrelated responsibilities change, the same Policy decision is duplicated in multiple places, or propagation cannot be explained.

Use unfounded future scenarios only to record risks. Do not use them as grounds for adding extension points.

---

## 5. Component and Package Design

### 5.1 Boundaries Require a Consumer and a Constraint

For a Component or Package boundary, define the Consumer that uses it and the concrete constraint that it protects. Use one of the following as the basis for a boundary:

- Ownership of state, a Policy, a decision, or an invariant
- A boundary of trust, permissions, transactions, consistency, concurrency, or lifecycle
- Isolation of an external dependency, volatile detail, or real substitutability
- An organizational ownership boundary

Do not introduce a boundary if either the Consumer or protected constraint cannot be shown. Do not create a new boundary when an existing boundary, Value, Function, or Package can protect the same constraint.

### 5.2 Record Package Design

A Package implements responsibilities and contracts owned by a Concept or Component and hides internal implementation details. Do not treat the Package itself as the owner of decisions in the problem domain. Record the following for Package design.

| Package | Concept/Component responsibilities implemented | Contracts published | Implementation hidden | Dependencies permitted |
|---|---|---|---|---|
| Target Package | Implementation target and responsibilities | Contracts provided to other Packages | Internal types, processing, and Provider-specific details | Public contracts it may depend on |

Record dependencies between Packages in the following form.

| Source Package | Target Package | Public contract used | Why dependency is required | Details that must not cross the boundary |
|---|---|---|---|---|
| Package using a public contract | Package containing the public contract | Interface, Port, or public type | Responsibility required by the consumer | Internal types, Provider-specific DTOs, private decisions |

### 5.3 Do Not Map Components and Packages Mechanically

Make the relationship between Components and Packages explicit, but do not assume one-to-one correspondence. When one Component is implemented by multiple Packages, separate the implementation hidden and dependency reason for each Package. When one Package implements multiple Concept or Component responsibilities, show that they change for the same reason and cohere as one public boundary.

Package names represent the hidden Concept, Component responsibility, or external boundary, not the beginning, middle, or end of processing. Do not create Packages mechanically from lists of capabilities, use cases, or processing steps.

### 5.4 Determine Dependency Direction from Owners and Consumers

- Core decisions do not depend on Frameworks, external SDKs, or Provider-specific contracts.
- The Component that uses an external dependency owns the minimum Port it needs.
- A Package implementing a Provider Context Module implements the Port owned by the Consumer and confines Provider-specific representations within the boundary.
- The Composition Root only creates and wires Components; it does not own business decisions.
- Do not turn the capability classification directly into the Package dependency structure.
- Do not create circular dependencies between Packages. When a cycle appears necessary, reconsider responsibilities, contract ownership, or dependency direction.
- Do not express execution order or a fixed workflow through dependencies between Components or Packages.

---

## 6. Interface and Port Design

### 6.1 Design Contracts after Responsibilities and Boundaries

Design Interfaces and Ports after the conceptual model, responsibility assignment, and boundaries are settled. Public contracts define:

- Purpose and Consumer
- Owned Concept, Component, or boundary
- Inputs and preconditions
- Outputs, postconditions, and meaning of the result
- Error contract
- Side effects
- Implementation details hidden
- Requirements, invariants, or external dependencies that require the contract

Introduce an Interface or Port only when there is both a real Consumer and a concrete constraint that direct dependency cannot protect. Constraints include external dependencies, trust boundaries, transactions, consistency, lifecycle, volatile details, or real substitutability. Do not use the current number of implementations alone as the basis for deciding whether to introduce one.

Record public contracts in the following form.

| Interface/Port | Consumer | Owner | Inputs/preconditions | Outputs/postconditions | Error/side effects | Constraint protected |
|---|---|---|---|---|---|---|
| Target contract | Component using it | Concept, Component, or boundary | Received information and conditions | Meaning and guarantees of the result | Failure and state changes | Basis for separating the contract |

### 6.2 The Consumer Owns the Port

Define a Port in the problem-domain vocabulary required by the Consumer. A Package implementing a Provider Context Module implements that Port. Do not expose Provider-specific APIs, DTOs, Errors, identifiers, or state vocabulary outside the Port.

Even when one Provider implements multiple Ports, do not merge Consumer-specific contracts into one large Provider Interface.

### 6.3 Choose the Minimum Implementation Representation

| Representation | Select when |
|---|---|
| Value | It has meaning and invariants but no independent identity or lifecycle |
| Function | It represents behavior with no state, identity, or lifecycle |
| Stateful Object/Class | It has independently changing state, or identity/lifecycle, and protects invariants concerning them |
| Interface/Port | There is a real Consumer and a concrete constraint that direct dependency cannot protect |
| Package | It groups the implementation and public contracts of a Concept or Component and hides internal details |

Avoid the following designs:

- Contract names such as `Process`, `Execute`, or `Handle` that do not express problem-domain meaning
- Numerous Primitive arguments
- Boolean Flags that switch behavior
- Large Interfaces containing operations unused by Consumers
- Abstractions created only for tests
- Interfaces based only on assumed future implementations
- One abstraction for each stage of a procedure

---

## 7. Behavioral Design through Tests

### 7.1 What Tests Specify

Tests specify the following behavior in an externally observable form.

- Rules and decision results
- State changes and invariants
- Errors and state remaining after failure
- Boundary values and Edge Cases
- Contracts at external boundaries

Tests do not fix Private Methods, internal call counts, implementation Class structures, or processing order. Limit Mocks to external boundaries or substitutable contracts; do not use them to reproduce internal structure.

For external constraints or quality conditions that cannot be verified by automated tests, define a reproducible verification method and verification result. Do not omit verification because it cannot be automated.

### 7.2 Implement with TDD

Implement new or changed behavior that can be verified automatically in the following order.

1. Red: Write a failing test that represents externally observable behavior.
2. Green: Write the minimum implementation that makes the test pass.
3. Refactor: Improve naming, duplication, responsibility assignment, and structure without changing behavior.

When Refactor requires a new Concept, responsibility, boundary, or public contract, update the design document and reconfirm from the corresponding design stage. Do not add an undesigned responsibility only in implementation.

---

## 8. Design and Implementation Review

### 8.1 Confirm Design Conformance

| Aspect | Pass condition |
|---|---|
| Correspondence to requirements | Each Concept, responsibility, boundary, and public contract maps to a requirement, constraint, invariant, or external fact |
| Concept minimality | The meaning lost by removing or integrating a Concept is clear, and the implementation representation is not excessive |
| Responsibility ownership | Each decision has one owner, and the required information and authority are gathered by that owner |
| Dependency direction | Higher-level decisions do not depend on Provider or Framework details, and Consumers own Ports |
| Interface | A real Consumer and concrete protected constraint exist, and the contract, Error, side effects, and hidden targets are clear |
| Behavior | Tests protect automatically verifiable rules, invariants, failures, and boundary conditions; other conditions have verification methods and results |
| Implementation representation | The minimum representation among Value, Function, Class, Interface, and Package is selected |
| Architecture | Component responsibilities and prohibited dependencies defined by the Architecture document are not violated |

### 8.2 Detect Procedure-Centered Implementation

The following conditions are signals to reconfirm responsibility assignment.

- Decisions are concentrated in a Handler, Controller, Use Case, or Application Service.
- A Model holds only data, while decisions using that state are scattered outside it.
- A long procedure, deep conditional branching, or multiple independent reasons for change are concentrated in one Component or Package.
- The same Policy or invariant is decided in multiple Components.
- `Manager`, `Processor`, `Helper`, or `Util` collects processing with no clear owner.
- Infrastructure DTOs, SDK Models, or Provider-specific Errors leak into the Core.
- An Interface exposes all Provider capabilities and makes the Consumer depend on unused operations.
- A Pattern, Class, or Interface is added where an existing Function or Value is sufficient.
- Tests reproduce internal call order or Private structure.

When a problem is found, before splitting the procedure, confirm the information, authority, and protected invariant required for the decision. Move behavior to the appropriate owner and remove duplication that represents the same rule.

### 8.3 Completion Conditions

Design and implementation are complete when all of the following conditions hold.

- There are no decisions without owners or duplicated decisions in multiple locations.
- Change propagation across boundaries can be explained from responsibilities and contracts.
- There are no Concepts, Classes, Interfaces, or Packages for unfounded future extensions.
- Provider- and Framework-specific details do not cross their defined boundaries.
- Automatically verifiable behavior and invariants are protected by tests, and conditions that cannot be automated have verification methods and results.
- Implementation matches the design document, and design Changes are reflected in the document.
