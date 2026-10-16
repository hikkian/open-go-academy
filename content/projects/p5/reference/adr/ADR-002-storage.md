# ADR-002: tenant-scoped relational source of truth

- Status: accepted as a hypothesis
- Date: 2026-10-16
- Requirements: FR-1, FR-2, NFR-3
- Revisit when: measured storage/retention or multi-region requirements invalidate one primary.

## Context

We need unique tenant-scoped idempotency, atomic outbox, retention and recovery semantics. Workload, region and deletion obligations are not confirmed.

## Alternatives

| Option | Benefits | Cost/failure mode | Suitable when |
|---|---|---|---|
| PostgreSQL relational tables | Transactions/constraints, operator familiarity, point-in-time recovery | Primary writes, index/vacuum pressure, payload bloat | Moderate scale and relational tenant checks |
| Document DB | Flexible envelopes and horizontal distribution | Cross-record transaction/unique semantics vary; outbox still needed | Schema evolution dominates and guarantees proven |
| Broker-only log | Natural replay and partition order | Tenant authorization, compaction/deletion and external state become harder | High measured streaming throughput with platform team |

## Decision and consequences

Use PostgreSQL for event metadata, outbox and receipts; keep payload bounded and separate large blobs only if measurement supports a safe transactional pointer protocol. Every query and unique constraint includes tenant scope. Backups need WAL/PITR and restore exercises; retention deletion has an audited policy. One primary is an availability limit.

## Verification

Run schema migration from empty state and upgrade state; test same-key concurrent insert, cross-tenant attempts and delete/retention. Restore a backup into an isolated environment and measure RPO/RTO.
