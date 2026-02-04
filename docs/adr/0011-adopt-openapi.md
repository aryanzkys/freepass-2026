## Title
Adopt OpenAPI for Documentation

## Status
Accepted

## Context
The project requires consistent API documentation for onboarding, testing, and integration. Documentation must be consumable by common tools and reviewers.

## Decision
Maintain an OpenAPI 3.0 YAML specification in the repository.

## Consequences
- Positive: Standardized, tool-compatible documentation with examples and schemas.
- Tradeoffs: Requires ongoing maintenance to keep spec aligned with implementation.
- Limitations: Does not replace integration tests or runtime validation.

## Alternatives Considered
- Ad-hoc Markdown docs only: Rejected due to limited tooling support.
- Auto-generated docs without curated examples: Rejected because the project needs explicit examples and error contracts.