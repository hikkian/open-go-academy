# P3 TaskFlow: browser acceptance and remaining scope

Updated 2026-10-16 after implementing the runner capability contract and workspace policy fixes. This report distinguishes completed browser evidence from the still-open product and rank gates.

## Completed browser acceptance

- `tests/p3-workspace.spec.ts` verifies that a combined runner advertises P3 over the isolated Unix socket, the API lists one P3 card, and P3 details report execution available. P2-only capability behavior is separately covered by `TestClientReadsFailClosedProjectCapabilities` and `TestTaskFlowAvailabilityRequiresHealthyAdvertisedCapability`.
- Desktop P3 browser acceptance passed 2/2: locked/read-only P3 module files, bounded workspace validation, no unsupported arbitrary run-arguments control, isolated Run, user Test, Submit with exactly 45/45 checks, rendered manual-review criteria, no rank change, browser reload persistence, and snapshot restore.
- Mobile P3 catalog/workspace check passed; axe reported zero violations and no horizontal overflow. Runtime-heavy duplication is intentionally skipped on mobile.
- P2 browser regression passed 2/2 on the combined project runner. The socket-level runtime evidence also covers P2 against both combined and old P2-only profiles.
- Visual artifacts and exact commands are in `docs/validation/p3-browser-acceptance.txt`.

## Fixes found during acceptance

1. The API required `SupportsProject("p3")`, but the Unix-socket Client did not expose runner capabilities. Added `GET /capabilities` to the project runner, strict bounded client decoding, and API propagation. Capability metadata is fail-closed; health is checked separately before the API consumes it.
2. The shared workspace treated every non-P2 project as a standard-library CLI project. It now applies the locked P2/P3 module and workspace policy to P3, disables editing/deletion of `go.mod`/`go.sum`, displays pgx/PostgreSQL and correct project-runner guidance, and hides unsupported arbitrary run arguments for the fixed TaskFlow liveness contract.
3. The project result UI now renders API-provided manual-review labels. P1/P2 responses also identify their actual review areas instead of using a generic README/Git message.

## Remaining scope and limitations

- The browser workflow uses a deterministic reference fixture. It proves integration with the current browser/API/isolated runner; it does not independently evaluate a learner's architecture or code quality.
- Browser persistence was verified across reload and snapshot restore, but not across an API-process restart while preserving the temporary database. Runner runtime tests separately cover isolated job cleanup and runner recovery.
- The browser suite does not yet exercise a stale P2-only profile through a full browser catalog session; unit and runner tests verify that P3 remains unavailable in that profile.
- Negative browser cases for missing `TestX`, TestMain-only tests, and a student implementation that fails P3 acceptance remain candidates for stronger coverage.
- P3 submission is automatic project evidence only. It does not award Middle, Middle+, or any credential. The separate human-review record and server-side advanced-rank evidence pipeline remain unimplemented. P4 and P5 are still open.
- This local runner evidence does not certify a public multi-tenant execution service.
