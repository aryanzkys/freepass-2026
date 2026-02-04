## Title
Use JWT Authentication

## Status
Accepted

## Context
The system needs stateless authentication to support REST APIs and horizontal scaling. Tokens must carry role claims for RBAC checks at the handler layer.

## Decision
Use JWTs for authentication and authorization, issued on login and validated on each request.

## Consequences
- Positive: Stateless design, easy API integration, supports scaling without session storage.
- Tradeoffs: Token revocation is not immediate without additional infrastructure.
- Limitations: Requires secure secret management and careful token expiry policies.

## Alternatives Considered
- Session-based auth: Rejected due to server-side state management overhead.
- Full OAuth flow: Rejected because it adds complexity beyond current requirements.