## Title
Use Golang + Gin Framework

## Status
Accepted

## Context
The backend must handle concurrent order creation, verification workflows, and multiple roles without sacrificing latency. The team needs predictable performance, fast startup, and easy deployment on commodity infrastructure. A minimal framework is preferred to keep business logic explicit and maintainable.

## Decision
Use Go as the primary language and Gin as the HTTP framework.

## Consequences
- Positive: High performance and efficient concurrency; minimal framework overhead; strong standard library; easy static builds.
- Tradeoffs: Smaller ecosystem than NodeJS for web-specific utilities; requires Go expertise for contributors.
- Limitations: Less built-in structure than full-stack frameworks, so conventions must be enforced by the team.

## Alternatives Considered
- NodeJS: Rejected due to less predictable performance under high concurrency and heavier reliance on external libraries for structure.
- Java Spring: Rejected due to heavier runtime footprint and slower startup compared to Go for this project scope.
- PHP Laravel: Rejected due to lower concurrency performance and a larger runtime stack for a backend focused on API services.