# ADR-001: PostgreSQL outbox and fenced worker leases

- Status: accepted for the illustrative single-region baseline
- Date: 2026-10-16
- Requirements: FR-1, FR-3, NFR-1
- Revisit when: event volume/lag exceeds measured primary capacity, multi-region writes are required, or downstream cannot deduplicate effects.

## Context

The client needs a durable acknowledgement while event processing happens asynchronously. A process-local queue loses accepted work on crash. A database commit followed by broker publish has a dual-write gap.

## Alternatives

| Option | Benefits | Cost/failure mode | Suitable when |
|---|---|---|---|
| In-process channel | Low latency and simple | Lost on restart; no durable acknowledgement | Best-effort local tasks only |
| DB outbox + bounded workers | Atomic with business row; one operational datastore; queryable recovery | Primary bears queue load; polling/retention and vacuum require care | Moderate throughput with SQL as source of truth |
| External durable broker | Independent queue scaling, consumer groups | Dual-write still needs outbox; extra cluster, ACLs, lag and replay operations | Measured queue isolation/throughput need justifies ops cost |

## Decision and consequences

Use one transaction for event and outbox, claim with `FOR UPDATE SKIP LOCKED`, persist lease expiry and monotonic token, release transaction before network call, then conditionally record completion with the token. Retries are at-least-once. The receiver must deduplicate on stable key or expose reconciliation. Do not claim exactly-once.

## Verification

Kill a disposable worker before call, after receiver commit, and before local completion; prove recovery and duplicate handling. Run concurrent claimers against a disposable PostgreSQL instance and show no two current tokens. Measure queue lag and DB lock/connection pressure at stated workload.
