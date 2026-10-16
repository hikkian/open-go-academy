# Operations and release plan (illustrative)

## SLI/SLO proposals

| SLI | Draft target | Window / response |
|---|---|---|
| Durable acceptance availability | 99.9% monthly | Count only after commit; reject overload explicitly, never false-accept |
| Acceptance latency | p99 < 300 ms at agreed peak | 15-minute peak profile; record body distribution and hardware |
| Delivery freshness | 99.9% committed events terminal within 5 minutes, measured over the agreed monthly window | Exclude only owner-approved maintenance; alert on oldest pending age |
| Unintended duplicate business effect | 0 observed | Requires receiver dedupe or reconciliation; zero metric does not prove universal exactly-once |

These are proposals pending product owner, security and operations approval. Error budget drives release caution, not silent SLO changes. Alert on acceptance failures, oldest queue age, retry/quarantine rate, DB disk/connection/lock saturation, backup age and restore-test failure. Use bounded labels; link alerts to runbook actions.

## Capacity and scaling

Use the example calculation in `system-design.md` only as a scenario. Before capacity approval: capture representative payloads without PII; test steady and burst arrival, retries, DB failover, and catch-up; publish measured per-worker service rate and the exact machine profile. Scale vertically/batch workers first while preserving DB headroom. Add broker or partitioning only with an ADR and migration/rollback plan.

## Rollout / rollback

1. Apply additive schema migration; deploy code that reads old and new schema; verify constraints and shadow metrics.
2. Enable ingest for one internal partner at low quota; compare duplicate/conflict counts and latency.
3. Canary a small agreed partner cohort; pause on any cross-tenant signal, false acknowledgement, unbounded lag, or error budget burn threshold.
4. Increase cohort only after owner sign-off. Persist a compatible event version and keep old consumers until backlog drains.
5. Roll back application binary while schema remains backward-compatible. Do not drop new columns or replay effects blindly. For irreversible downstream effects, use reconciliation/compensation and incident approval.

No public rollout is authorized by this learning package.

## Backup and restore

Back up encrypted DB snapshots plus WAL to an access-controlled separate failure domain; backup encryption keys through the approved key-management process, never beside the database. Daily restore exercise in an isolated network: restore to a point, compare event/outbox/receipt invariants and tenant counts, record elapsed recovery and data-loss interval. Draft targets RPO ≤5 min and RTO ≤60 min must be replaced if actual measurements miss. A backup job success metric is not proof of recoverability.

## Runbook snippets

### Queue age rising

1. Declare incident and capture queue age, DB headroom and downstream status without payloads.
2. If downstream is failing, lower partner quotas / apply backpressure; avoid unbounded worker retries.
3. If DB is saturated, preserve ingest truth: reject with retryable response before commit; do not return accepted.
4. Restore downstream or capacity, then increase worker concurrency gradually within DB pool budget.
5. Review quarantined messages; redrive only after schema/receiver correction and two-person approval.

### Suspected credential compromise or cross-tenant access

1. Revoke/rotate credential, restrict affected tenant and preserve audit logs.
2. Stop unsafe reads/writes and page security incident lead; do not erase evidence.
3. Establish exposure scope from access logs and immutable event IDs; avoid copying payload into incident channels.
4. Follow approved notification and retention obligations; verify access control fix with negative tests before restore.

### Recovery exercise

Quarterly game day: kill a worker during downstream timeout; expire its lease and start a new worker; prove stale completion is rejected and receiver deduplicates retry. Separately restore DB into isolated environment. Record participants, build hash, timings, missed targets and action owners.
