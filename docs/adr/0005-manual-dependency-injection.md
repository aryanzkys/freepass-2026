## Title
Implement Manual Dependency Injection

## Status
Accepted

## Context
The project is a medium-sized backend with a limited number of services and handlers. The team prefers explicit wiring to keep dependencies visible and to avoid heavy frameworks. Testing requires straightforward construction of handlers with mocks.

## Decision
Use manual dependency injection by constructing handlers and services explicitly in the application setup.

## Consequences
- Positive: Transparent dependencies, easy to understand initialization, and no framework lock-in.
- Tradeoffs: Wiring code grows as the project scales; no automatic lifecycle management.
- Limitations: Requires discipline to keep constructor signatures manageable.

## Alternatives Considered
- DI frameworks: Rejected due to added complexity and reduced transparency for the project size.
- Service locator pattern: Rejected because it hides dependencies and complicates testing.