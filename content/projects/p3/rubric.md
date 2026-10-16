# P3 TaskFlow objective rubric

This rubric checks reproducible project behavior, not the learner's identity, personality, employment, job title, years of experience, or professional competence outside these artifacts. A missing human review is reported as unreviewed; the automated score is not a job credential.

| Area | Points | Observable evidence |
|---|---:|---|
| Data model and migrations | 15 | Versioned up/down migration, FK/check/unique/index invariants, fresh-database setup, documented cardinalities |
| Identity and tenant RBAC | 20 | Hashed random session, server-owned actor, owner/editor/viewer matrix, denied cross-tenant read/write paths |
| HTTP and SQL boundary | 15 | Strict bounded JSON, parameterized values, safe errors, stable OpenAPI, list/body/time limits |
| Task correctness | 15 | CRUD slice, optimistic version conflict, comment ownership, idempotency-key replay and collision behavior |
| Transactional outbox | 15 | Atomic task+event, concurrent claim exclusion, unique effect receipt, crash/lease recovery |
| Worker operations | 10 | Bounded workers/batches/retries, terminal failure, shutdown drain and cancellation checks |
| Verification and operations | 10 | Test/vet/race/build, real disposable-Postgres integration, fixed-size load/failure evidence, pinned local Compose, threat notes |

**Automated acceptance gate:** each invariant must pass against a clean disposable database; cross-tenant data must not appear; wrong role and stale version must be denied; same authorized idempotency key must not duplicate a task/event; repeated worker delivery must not duplicate the receipt; an invalid event must retry then reach `dead`; student code is not assumed safe merely because the trusted reference passed. The exact checker provides the authoritative machine-readable tests. This rubric does not turn static review observations into claims of production readiness.

**Evidence record:** record Go version, exact commands, test output, local database image digest/platform, fixed workload size/timeout, and any failed/skipped test. Performance results are valid only for their stated machine and workload; no universal throughput threshold is implied.
