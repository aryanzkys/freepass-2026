## Title
Use PostgreSQL as Primary Database

## Status
Accepted

## Context
Order creation and payment verification require strong consistency, transactional safety, and reliable constraint enforcement. The data model is relational with clear entities and relationships. The project also depends on sqlc, which expects a SQL-centric workflow.

## Decision
Use PostgreSQL as the primary relational database.

## Consequences
- Positive: Strong ACID guarantees, robust constraints, and proven reliability for financial and inventory data.
- Tradeoffs: Requires schema migrations and careful indexing; operational setup is heavier than document databases.
- Limitations: Horizontal scaling requires additional tooling beyond the current scope.

## Alternatives Considered
- MongoDB: Rejected due to weaker relational integrity and transaction complexity for multi-entity order workflows.
- MySQL: Rejected because PostgreSQL offers stronger feature depth for constraints and query flexibility that align with sqlc usage.