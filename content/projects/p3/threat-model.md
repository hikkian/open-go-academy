# TaskFlow threat notes

## Assets and trust boundaries

- User passwords, session bearer secrets, workspace membership, project/task/comment data, PostgreSQL credentials, and queued job payloads are sensitive.
- The browser/curl client is untrusted. A request body, path UUID, idempotency key, and forwarded-IP header are never proof of identity or membership.
- The Go API, its database role, migration files, and fixed Compose manifest are trusted local project components. The project runner executes student code separately; it must never receive this API's database credential or share its sockets.
- The background worker reads only PostgreSQL outbox rows and writes a unique local receipt. It does not call arbitrary URLs, execute supplied commands, or send a job to an external broker.

## Controls implemented in the reference

- Registration accepts bounded email/password input, derives a per-account salted PBKDF2-HMAC-SHA256 hash (210,000 iterations), and never stores the password. Session tokens have 256 bits from `crypto/rand`; only SHA-256 token hashes are stored; cookies are `HttpOnly`, `Secure`, `SameSite=Strict`, and expire after 24 hours. Logout deletes the hashed session record.
- The API derives `user_id` from the server-side session. Each project/task/comment query joins `memberships`; caller IDs and roles are never accepted as authority. Write operations restrict owner/editor roles; task deletion is owner-only. Foreign keys and workspace-scoped checks constrain tenant relationships. Missing and unauthorized objects use the same `404` response.
- SQL values are passed as pgx parameters. Request bodies are capped at 1 MiB, unknown JSON fields and trailing JSON documents are rejected, text fields are length-limited, returned lists cap at 100, server timeouts/header bytes are bounded, worker concurrency/batch/retries are fixed, and a bounded per-remote-IP request counter returns `429`. The proxy forwarding headers are ignored.
- Task plus outbox write use one transaction. Project membership is held with `FOR SHARE` through task/outbox commit: a role downgrade either commits first and the task is denied, or waits until the authorized create commits. Outbox keys are unique per project. The worker commits its attempt and a fresh lease UUID before applying its effect, then commits unique receipt plus `done` only while that lease token remains current. Stale failure transitions are conditional too. A lost worker can be reclaimed after the 30-second lease, while retry delay is capped and failures become `dead`. Shutdown bounds HTTP close and worker join to five seconds; it skips drain if workers fail to join, otherwise it bounds drain to five seconds.
- `/livez` reports process liveness only. `/readyz` checks PostgreSQL with a 500 ms request context and reports ready only while the worker has started and is not draining.
- Client-facing errors use fixed HTTP text and never include SQL errors or secrets. Structured HTTP logs record method, matched route template, status, and duration but do not log request bodies, passwords, cookies, query strings, or path values.
- Compose pins PostgreSQL 18.6 linux/amd64 digest, publishes only loopback ports, drops capabilities, uses a read-only API container, and disables privilege escalation.

## Risks and explicit exclusions

This reference is a classroom-only sample, not a production security guarantee. It is not safe to expose publicly or to make multi-tenant isolation claims from one test suite. It lacks TLS termination/certificate lifecycle, CSRF token/origin policy for hostile same-site subdomains, account recovery/email verification/MFA, pending invitations for users who have not registered, session rotation/revocation UI, durable backups, secret management, database least-privilege bootstrap, migration coordination across concurrent API replicas, alerts, and independent penetration review. Its rate limit is per process and IP; NAT users share a budget, and a trusted reverse proxy needs an explicit proxy trust policy before forwarding addresses may be honored. PostgreSQL runs with a local Compose password, which is unsuitable outside the disposable environment.

The outbox receipt demonstrates durable local processing but does not guarantee exactly-once effects outside PostgreSQL. A future broker integration needs a delivery/ack, redelivery, publisher-confirm and destination idempotency design, plus resource/time bounds. Redis is not a durable queue substitute here. Do not run arbitrary submitted Compose or student code with these credentials. The application worker belongs to the trusted service; user Go code remains in the network-isolated project runner.

The test creates a random schema and drops it at completion. Only point `TASKFLOW_TEST_DATABASE_URL` to a dedicated disposable database and test role. The Compose stack is loopback-only and the container image digest recorded here is verified only for linux/amd64.
