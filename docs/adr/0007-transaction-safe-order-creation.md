## Title
Implement Transaction-Safe Order Creation

## Status
Accepted

## Context
Orders reduce stock and create multiple dependent records. Concurrent orders must not oversell inventory. Financial correctness and data integrity are critical for user trust.

## Decision
Use database transactions with row locking (SELECT FOR UPDATE) during order creation to ensure atomic stock updates and order persistence.

## Consequences
- Positive: Prevents race conditions and overselling; ensures consistent order records.
- Tradeoffs: Locking can reduce throughput under heavy contention.
- Limitations: Requires careful transaction scope to avoid long-lived locks.

## Alternatives Considered
- Optimistic concurrency without locks: Rejected due to higher complexity and risk of oversell under contention.
- In-memory locks: Rejected because it does not scale across multiple instances.