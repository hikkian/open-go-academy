# P3 — TaskFlow

TaskFlow is the Middle backend project: a single Go HTTP service for users, workspaces, projects, tasks, comments, and durable background work. It uses PostgreSQL as the source of truth and a transactional outbox; the base project intentionally needs no Redis or broker. The project has a clean student starter and a separate runnable reference.

## Project files

- `starter/` is the student workspace. Its [README](starter/README.md) contains the requirements, checkpoints, and acceptance contract.
- `reference/` is the complete, testable learning solution. Do not copy it as a substitute for explaining your own design.
- [`openapi.yaml`](openapi.yaml), [`rubric.md`](rubric.md), and [`threat-model.md`](threat-model.md) define the contract and evaluation boundaries.
- [`manifest.json`](manifest.json) pins the Go, pgx, and PostgreSQL versions and records the compatibility limits.
- [`compose.yaml`](compose.yaml) starts a loopback-only disposable local environment. It is for this repository's fixed services, never for user-supplied manifests.

## Run the reference

Requirements: Go 1.27.2, Docker Compose v2, and an available linux/amd64 image registry for the pinned PostgreSQL image. Compose binds ports only to `127.0.0.1` and uses a dedicated named volume. Set a random local password before startup:

```sh
cp .env.example .env
# Put a locally generated random value in POSTGRES_PASSWORD.
docker compose up --build
curl http://127.0.0.1:8098/livez
curl http://127.0.0.1:8098/readyz
```

To deliberately remove the TaskFlow schema/data and then recreate it on next API start, run `docker compose run --rm api migrate-down`. This is destructive; use only the named disposable project database.

The API applies migration 0001 at startup. `/livez` checks only process liveness. `/readyz` has a 500 ms PostgreSQL ping bound and reports ready only while the worker is running and not draining. It is a classroom sample with no TLS termination, account recovery, email verification, pending invitation workflow, backup strategy, or production deployment guarantee. Do not publish it on a public host or reuse credentials.

Run deterministic checks without a database and run the real PostgreSQL suite against a **disposable test database only**:

```sh
cd reference
go test ./...
go vet ./...
go test -race ./...
go build ./...
go mod verify

# Example only after creating a disposable database named taskflow_test:
TASKFLOW_TEST_DATABASE_URL='postgres://taskflow:local-only-password@127.0.0.1:55433/taskflow_test?sslmode=disable' go test -run TestPostgresTaskOutboxAndTenantIsolation -count=1
```

The integration test creates and drops a random schema inside that database. Never point it at a database containing data you need. It verifies real PostgreSQL migrations, tenant checks, project-scoped idempotency, role-revocation serialization, logout revocation, outbox lease fencing/retries, terminalization of an expired final attempt when the queue has no other jobs, and concurrent worker delivery. The no-database unit tests do not claim to replace it.

## Authentication and API flow

Register with `POST /api/v1/users` using an email and a password of at least 12 characters. The API creates the initial workspace and sets a random, 24-hour, `HttpOnly`, `Secure`, `SameSite=Strict` cookie; the database stores only its SHA-256 hash. For a plain HTTP local curl session, curl must be told explicitly to retain secure cookies only when using HTTPS; the Compose reference binds to HTTP for local learning, so use a local HTTPS reverse proxy if you want to exercise cookies with a browser. Do not disable `Secure` on an exposed service.

Create a workspace, then a project, then submit a task with an `idempotency_key`. Retrying the same key in the same authorized project returns the original task. Task-state updates use the returned `version` for optimistic concurrency. Owners/editors may mutate; viewers may list tasks and comments. All authorization is derived from the server session and checked in the SQL statement; request bodies never select an acting user.

## Learning decision: SQL outbox first

Task creation and its `task.created` outbox record commit in one PostgreSQL transaction. Idempotency keys are unique within a project, so separate projects can use the same value. Membership is locked through commit to serialize task creation with role revocation. A bounded worker claims jobs with `FOR UPDATE SKIP LOCKED`, persists an attempt and fresh fencing token, and commits a unique receipt with `done` only if that token is still current. Failures receive capped exponential delay and eventually enter `dead`; stale workers cannot change a newer claim. Graceful shutdown bounds HTTP close and worker join to five seconds each; it skips drain when workers do not join, otherwise it drains currently available jobs for up to five seconds. A future lesson may compare RabbitMQ for brokered delivery or Redis for cache/rate-limit workloads, after explaining operational costs and delivery semantics. Neither is required to run this product.

The receipt is intentionally a small, local demonstration side effect. It is not a guarantee of exactly-once delivery to arbitrary external systems. External effects still need idempotency at the destination and an operational recovery process.
