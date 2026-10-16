# TaskFlow student starter

This directory is your clean project workspace. It contains only a runnable liveness endpoint and safe HTTP server limits. The product routes, persistence, authentication, readiness check, and worker are yours to implement from the acceptance contract; this starter deliberately does not pretend those features already work.

## Outcome and boundaries

Build one Go 1.27.2 HTTP service for a small team task tracker. A user owns one or more workspaces; workspace members have `owner`, `editor`, or `viewer` roles; projects belong to a workspace; tasks and comments belong to a project/task. Use PostgreSQL 18 and versioned migrations. Preserve server-owned identity: never accept a user ID or role from a request as authority. Bind local services to loopback and do not run student Go code with database credentials outside the disposable local Compose stack.

The service is a classroom project. Passing the checker demonstrates the listed Go Academy behaviors; it does not establish production security, public multi-tenant safety, a service-level objective, a job qualification, or employment experience.

## Required behavior

1. Register and log in users. Derive the actor from a server-side session; store only a hash of the random token. Use a vetted password derivation function with per-user salt and documented work factor. Logout revokes the session.
2. Create/list workspaces and projects with role-aware membership. Let an owner assign `editor` or `viewer` to an existing user without allowing an owner role to be granted or removed through that route. A user cannot read or mutate another workspace by substituting UUIDs. Use the same tenant condition in the SQL operation that reads or mutates the row.
3. Create/list/update/delete tasks and create/list comments. Validate lengths and enum states. Use parameterized SQL, stable bounded pagination, foreign keys, indexes, and optimistic version checks for concurrent updates.
4. Create a durable `task.created` job in the same database transaction as its task. Repeated idempotency keys return the same result only within the authorized project. Explain collision handling and what happens after a process crash.
5. Process jobs with a bounded worker and database leases/`SKIP LOCKED`, persisted attempts, capped retry delay, terminal failure state, unique effect receipt, and graceful bounded drain. A local PostgreSQL outbox is sufficient. Add Redis or RabbitMQ only after explaining the measured need and the different failure semantics.
6. Bound request bodies, list sizes, concurrent worker count, retries, header/time limits, and per-client request rate. Do not trust forwarded-IP headers from an unspecified proxy. Return stable errors without SQL diagnostics, secrets, or stack traces.
7. Provide process-only `/livez` and dependency-aware `/readyz` that checks PostgreSQL and worker state with a short timeout; add structured request/job observations, OpenAPI, migrations with rollback, pinned dependencies, local Compose, threat notes, a repeatable database-backed test and a bounded load/failure test.

## Checkpoints

- **C1 — design/schema:** state cardinalities, role matrix, table keys/constraints/indexes, migration rollback, API contract, and threat boundaries. Implement a clean migration from an empty database.
- **C2 — tenant API:** registration/session lifecycle, workspaces, projects, tasks, comments. Demonstrate cross-user denied/not-found results for every resource path, plus a safe SQL/value-boundary test.
- **C3 — concurrency:** version conflict, same idempotency key twice and concurrently, transaction rollback between task and outbox, two workers claiming a finite queue, worker crash/reclaim, bounded retries, and shutdown drain.
- **C4 — release evidence:** unit/integration/race/vet/build checks, short fixed-size load case, migration down/up, pinned image/module manifest, OpenAPI validation, and an honest limitations note.

## Acceptance contract

The checker can verify compilation, API status/data shapes, schema constraints, deterministic identity ownership, role checks, SQL parameterization rules, idempotency, persisted event receipts, lease recovery, retry bounds, request/rate limits, migration behavior, and clean startup/stop against its own disposable database. The checker does not score personality, unobserved professional experience, job titles, years employed, or unstated production guarantees. If the checker cannot establish a requirement from a reproducible artifact, report it as unverified.

## Run your starter and checks

```sh
go run .
curl http://127.0.0.1:8098/livez
go test ./...
go vet ./...
go test -race ./...
go build ./...
```

See the parent `README.md` for the pinned local PostgreSQL Compose stack and the separate complete reference implementation. Use only an explicitly disposable database for integration tests.
