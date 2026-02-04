## Title
Use sqlc Instead of ORM

## Status
Accepted

## Context
The system relies on precise SQL behavior for stock locking, transactional boundaries, and predictable query plans. The team requires compile-time safety while keeping SQL explicit for review and optimization.

## Decision
Use sqlc to generate type-safe query code from handwritten SQL.

## Consequences
- Positive: Clear visibility into SQL, strong type safety, and predictable performance.
- Tradeoffs: More upfront effort to maintain SQL files and schema migrations.
- Limitations: Less abstraction for rapid prototyping compared to ORMs.

## Alternatives Considered
- GORM: Rejected due to opaque query generation and potential performance surprises.
- Ent ORM: Rejected because the project prefers SQL-first control and simpler build tooling.