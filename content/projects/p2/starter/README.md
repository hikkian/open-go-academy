# P2 LinkHub — student starter

Build a REST API for personal short links. This folder is a compiling starting point, not the implementation. The complete contract is in `../README.md`, `../openapi.yaml`, and `../threat-model.md`; scoring criteria are in `../rubric.md`.

## Required endpoints

- `POST /api/v1/auth/register`, `/login`, `/logout`; `GET /api/v1/me`
- Owner-only `GET/POST /api/v1/links`, `GET/PATCH/DELETE /api/v1/links/{id}`
- Owner-only `GET/POST /api/v1/tags`, `PATCH/DELETE /api/v1/tags/{id}`
- Public `GET /{slug}` redirects only when the saved link is enabled; `GET /healthz` reports process health

Implement the listed behavior and error semantics in the OpenAPI contract. A client may omit `slug` at link creation; then generate a cryptographically random slug and let the unique database constraint decide collisions, retrying only a small fixed number of times. Use PostgreSQL and parameterized `pgx` queries. Do not execute submitted Go source, make outbound HTTP requests for redirect targets, or trust an owner ID supplied by a client.

## Work checkpoints

1. Design tables, reversible migrations, constraints, and indexes. Explain why the slug index is global while tag names are unique within one owner.
2. Implement bounded JSON and validations, then authentication with salted PBKDF2-SHA256 (or a reviewed, pinned alternative), opaque server-side sessions, CSRF, and Origin checks.
3. Implement owner-scoped link/tag operations. A link's write and tag association must commit or roll back as one transaction; the transaction must reject another user's tag.
4. Add deterministic HTTP tests using a fake store, then opt-in PostgreSQL tests that create an isolated disposable database. Cover foreign-owner ID access, CSRF, redirect validation, collisions, and rollback.
5. Run formatting, tests, vet and build; document how a local user starts PostgreSQL and the API with Compose.

The initial source file contains only the HTTP server setup and health endpoint. Replace its explicit `TODO` response as you implement each endpoint. The `reference/` directory is a separate comparison solution and should not be used as a source of hidden tests.

```sh
go test ./...
go vet ./...
go build ./...
```
