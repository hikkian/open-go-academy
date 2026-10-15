# P2 — LinkHub API (Junior+)

LinkHub is a learning project for a small REST API that stores short links in PostgreSQL and redirects visitors through enabled slugs. The repository contains a starter without the business solution and a separately verified reference implementation. The reference is a trusted example, not a service for executing student code.

## Learning outcomes

The project practices HTTP handlers with `net/http`, JSON validation, middleware, PostgreSQL through `pgx`, SQL migrations, transactions, ownership checks, cookie sessions, CSRF, database indexes, logging, OpenAPI, and container-based local development. Authentication is an educational implementation. A local run is not production-ready identity, security certification, or a trusted credential issuer.

## Checkpoints

1. **Schema and repository:** create users, sessions, links, tags and link-tag associations; add indexes, constraints and reversible migrations. Show that the migration can be applied to an empty database.
2. **Authentication and HTTP boundary:** registration, login, logout, current-user endpoint, bounded JSON, password hashing, session revocation, CSRF and Origin checks. Never accept an owner ID from a request.
3. **Links and tags:** owner-scoped CRUD, transactional tag association, bounded stable pagination, public redirect for enabled links only, and OpenAPI documentation.
4. **Release evidence:** automated tests, `go vet`, `go build`, local Compose instructions, threat notes, and an explanation of one design trade-off.

## Requirements and acceptance

See [`starter/README.md`](starter/README.md) for the full student contract, [`rubric.md`](rubric.md) for scoring, [`openapi.yaml`](openapi.yaml) for the API contract, and [`threat-model.md`](threat-model.md) for boundaries. The following observable behaviors are mandatory:

- User A cannot list, read, change, or delete User B's resources. A user cannot associate a tag owned by someone else.
- Session secrets are random, cookies are HttpOnly and SameSite=Strict, stored session and CSRF values are hashes, and logout revokes the server-side session.
- Mutations with cookie authentication require a matching CSRF token and same configured Origin. Cross-site fetches are rejected.
- Redirect targets accept only HTTP(S) URLs without credentials or control characters. The API never fetches a target itself.
- SQL values are parameterized; migrations enforce unique slugs, normalized per-user tag names, foreign keys, and useful owner indexes.
- Errors do not expose database diagnostics, connection strings, passwords, or tokens.

## Run the reference locally

Requirements: Go 1.27.2 and Docker Compose v2. Copy `.env.example` to `.env`; generate a local password using `openssl rand -hex 24` and replace `replace-with-local-random-hex-password`. `.env` is ignored by this project. Compose publishes both services only on loopback.

```sh
cp .env.example .env
# Edit POSTGRES_PASSWORD in .env with a locally generated hex value.
docker compose up --build
curl http://127.0.0.1:8097/healthz
```

To roll back the initial schema deliberately, stop the API and run `docker compose run --rm api migrate-down`. This drops the LinkHub tables in the configured database; use it only with the disposable local project database.

Reference checks, from `reference/`:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

For a manual browser/API flow, register at `POST /api/v1/auth/register`, retain both cookies, read `csrf_token` from the response, then send it in `X-CSRF-Token` for mutations. Do not place the session token in local storage. API field names and examples are in `openapi.yaml`. `PATCH` sends the full editable link representation; omitted tag names clear the link's tag associations.

## Student workflow

Start in `starter/`, read its project contract, and add implementation plus tests in small commits. The separate `reference/` folder contains a complete example after attempting the project. Do not copy its entire source as a substitute for understanding the ownership, transaction, and CSRF rules. The project can be copied into a Git repository as-is; no external account or paid service is required.
