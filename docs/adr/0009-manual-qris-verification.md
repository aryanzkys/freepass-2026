## Title
Use Manual QRIS Verification Instead of Payment Gateway

## Status
Accepted

## Context
The project scope prioritizes a realistic campus workflow where owners verify QRIS payments manually. Integrating a full payment gateway would increase complexity and cost beyond current requirements.

## Decision
Record QRIS confirmations and require owner verification, storing approvals or rejections in a verification table with optional refund records.

## Consequences
- Positive: Lower integration complexity, clear audit trail, and alignment with offline operational practice.
- Tradeoffs: Verification is slower and depends on human action.
- Limitations: No automatic settlement or reconciliation with external payment processors.

## Alternatives Considered
- Midtrans: Rejected due to integration scope and operational overhead.
- Xendit: Rejected for similar reasons and unnecessary gateway complexity.
- Stripe: Rejected because it is not tailored to local QRIS workflows and exceeds project scope.