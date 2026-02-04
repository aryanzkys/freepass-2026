## Title
Support Dual Payment Methods (Cash + Cashless QRIS)

## Status
Accepted

## Context
Campus canteens accept both traditional cash payments and QRIS-based digital transfers. The system must support both without excluding users or vendors who prefer cash.

## Decision
Implement two payment methods: CASH and CASHLESS_QRIS, with distinct order and payment state handling.

## Consequences
- Positive: Realistic fit for campus operations and inclusive user experience.
- Tradeoffs: More complex state management and verification flows.
- Limitations: Digital payments rely on manual verification rather than automatic settlement.

## Alternatives Considered
- Cash-only: Rejected due to limited user convenience and reduced digital adoption.
- Cashless-only: Rejected because it excludes vendors and users who rely on cash.