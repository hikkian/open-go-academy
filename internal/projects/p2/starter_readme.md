# LinkHub API — student starter

Build a REST API for personal short links. This workspace starts with a compiling server shell, not the business implementation. The project contract is in [`docs/project.md`](docs/project.md), with API details in [`docs/openapi.yaml`](docs/openapi.yaml), threat boundaries in [`docs/threat-model.md`](docs/threat-model.md), and scoring in [`docs/rubric.md`](docs/rubric.md).

Implement the listed behavior in small steps. Use PostgreSQL and parameterized pgx queries. Do not execute submitted Go source, make outbound HTTP requests for redirect targets, or trust an owner ID supplied by a client.

## Work checkpoints

1. Design tables, reversible migrations, constraints, and indexes. Apply the migration to an empty database.
2. Implement bounded JSON and validations, then salted password hashing, opaque server-side sessions, CSRF, and Origin checks.
3. Implement owner-scoped link and tag operations. Save links and tag associations atomically; reject another user's tag.
4. Add deterministic HTTP tests using a fake store, then opt-in PostgreSQL tests with an isolated disposable database.
5. Run `go test ./...`, `go vet ./...`, and `go build ./...`; document local Compose setup and a design trade-off.

The initial source contains only `/healthz` and an explicit not-implemented fallback. Replace the fallback as you implement each endpoint. The Compose file and Dockerfile are export assets for a learner's local environment. The academy does not execute student Compose files or Dockerfiles.
