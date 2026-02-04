## Title
Use REST API Architecture

## Status
Accepted

## Context
The system needs a simple, reliable API surface for mobile and web clients with limited integration overhead. The team required tooling compatibility for manual testing and documentation, and the backend endpoints are resource-oriented (canteens, menus, orders, payments) which map naturally to HTTP verbs. The environment favors a straightforward client-server model suitable for a campus canteen.

## Decision
Adopt a RESTful API style using HTTP methods and resource-based routes, documented with OpenAPI.

## Consequences
- Positive: Easy onboarding, clear resource boundaries, broad tool support (Swagger UI, Postman, Apidog), and low operational complexity.
- Tradeoffs: Less flexible querying than GraphQL and less efficient for chatty clients in some cases.
- Limitations: Multiple round-trips may be required for complex views; versioning is handled through documentation and routes rather than a schema gateway.

## Alternatives Considered
- GraphQL: Rejected due to higher operational complexity, schema management overhead, and a larger learning curve for a small team.
- gRPC: Rejected because browser compatibility and tooling for manual testing are less convenient for the target users and reviewers.