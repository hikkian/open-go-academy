# ADR-003: server-derived tenant identity and bounded isolation

- Status: accepted for first release; security review required
- Date: 2026-10-16
- Requirements: FR-2, NFR-2
- Revisit when: customer-specific keys, residency or hostile-tenant isolation require stronger boundaries.

## Context

Events are sensitive and a caller-controlled tenant field would permit cross-organization access. Isolation must survive new endpoints and worker queries.

## Alternatives

| Option | Benefits | Cost/failure mode | Suitable when |
|---|---|---|---|
| Shared tables with tenant predicates and compound FKs | Efficient and easy to operate | One omitted predicate can leak; requires systematic tests/review | Many tenants with trusted service boundary |
| Schema/database per tenant | Stronger blast-radius and residency controls | Migration/connection/backup fleet complexity | Few high-isolation customers |
| Separate deployment per tenant | Strong operational boundary | High cost and fleet drift | Contract or threat model demands dedicated environment |

## Decision and consequences

Use server-validated rotating partner credential → tenant mapping; never trust tenant ID from payload. Use tenant-scoped keys/FKs and repository methods require an authenticated tenant context. Worker claim is tenant-aware for quota fairness; privileged support reads use audited separate role. Shared tables retain residual risk from application query mistakes.

## Verification

Automate cross-tenant read/write tests for every API operation and worker path, audit SQL predicates, test credential rotation/revocation, and run an independent threat review. A single integration test is not sufficient proof of complete isolation.
