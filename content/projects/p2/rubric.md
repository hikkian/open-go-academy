# P2 LinkHub rubric — 100 points

## Automated checks — 70 points

| Area | Points | Evidence |
|---|---:|---|
| HTTP contract and validation | 12 | Methods/status codes, JSON bounds, unknown fields, invalid slugs/URLs/tags, pagination bounds |
| Authentication and browser protection | 14 | salted password KDF, generic login failure, random hashed sessions, expiry/logout, cookie flags, CSRF and Origin checks |
| Ownership and authorization | 14 | every link/tag operation scopes by authenticated owner; wrong-owner IDs return the same not-found behavior; owner input is never trusted |
| PostgreSQL integrity | 12 | parameterized SQL, reversible migrations, unique constraints/FKs/indexes, atomic link/tag updates and cross-owner association rejection |
| Redirect and abuse boundaries | 8 | enabled slugs only, validated fixed destination, no target fetch, bounded requests and generic errors |
| Deterministic test and build quality | 10 | fake-store HTTP tests, explicit opt-in real PostgreSQL suite, race/vet/build evidence |

## Manual review — 30 points

| Area | Points | Evidence |
|---|---:|---|
| Readability and Go structure | 8 | clear names, focused handlers/store, error handling, formatting and useful comments |
| API and operational docs | 7 | accurate OpenAPI, local Compose setup, migration and test instructions, no secret values |
| Design explanation | 8 | student can explain ownership predicates, transaction boundary, session/CSRF design and redirect threat |
| Git workflow | 7 | small meaningful commits, tests in relevant changes, reviewable history and honest completion notes |

Passing target: 75/100 and every mandatory security acceptance case. Any cross-user access, plaintext passwords, client-controlled ownership, or SSRF behavior is a gate failure regardless of points. This is an educational project score, not employment qualification or a professional security certification.
