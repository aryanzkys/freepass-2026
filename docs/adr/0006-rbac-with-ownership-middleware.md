## Title
Implement RBAC with Ownership Middleware

## Status
Accepted

## Context
The system has distinct roles (USER, OWNER, ADMIN) with different permissions, and canteen data must be restricted to owners of the canteen. Preventing horizontal privilege escalation is a core security requirement.

## Decision
Enforce RBAC at the handler layer and apply ownership checks for owner-scoped resources.

## Consequences
- Positive: Clear security boundaries, consistent authorization checks, and reduced risk of data leakage.
- Tradeoffs: Additional authorization logic in handlers and middleware.
- Limitations: Requires careful testing of role and ownership combinations.

## Alternatives Considered
- Role-only checks without ownership validation: Rejected due to risk of cross-tenant access.
- Database-level row security: Rejected due to added operational complexity and tighter coupling to database features.