## Title
Introduce AI Feedback Analyzer Using Gemini

## Status
Accepted

## Context
Owners and admins need faster insight from aggregated feedback to identify trends and issues. The project requires AI assistance without exposing personal data or altering transactional logic.

## Decision
Use Gemini to generate read-only insights from aggregated feedback and order data, with explicit timeouts and safe prompts.

## Consequences
- Positive: Faster qualitative insights, improved owner/admin situational awareness.
- Tradeoffs: AI outputs are advisory and may be imperfect; requires API key management.
- Limitations: AI is not used for decisions that change data; features are disabled when API key is missing.

## Alternatives Considered
- Rule-based analytics: Rejected due to limited expressiveness for open-ended feedback.
- No AI solution: Rejected because it provides less actionable summary insight for stakeholders.