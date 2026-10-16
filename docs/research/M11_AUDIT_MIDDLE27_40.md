# M11 pedagogical audit: Middle-27…40

**Status:** read-only review of the current draft snapshot. No candidate implementation, runner, or acceptance gate was executed. Reviewed all 14 chapter JSON/Markdown pairs, 42 exercise contracts, corresponding grading specs (public/private vectors), and the 42 checked-in reference solutions/examples. This review checks curriculum/content consistency and plausible edge-case coverage; it is not a full Go/runtime or production-security audit.

## Findings

### P1 — SLO reference arithmetic can report the wrong result on large counters

**Where:** middle-33.md lines 60–76 and 80–98; middle-33.json Medium/Hard contracts; middle-33-medium.json, middle-33-hard.json; internal/content/solutions/middle-33-medium/main.go and middle-33-hard/main.go.

Both examples and references compare failed*100 with total*maxErrorPercent using int. Go integer overflow wraps; the grader's ordinary-size vectors never exercise it. A 64-bit counter example is total=9223372036854775807, failed=4703919738795935661, maxErrorPercent=25: the true error ratio is about 51%, so it must fail, but wrapped products are 9223372036854775700 <= 9223372036854775783, so the current predicate returns true. That is a false SLO pass, directly contradicting the stated “доля сбоев не превышает порог” contract.

**Minimum fix:** use overflow-safe comparison (for example, quotient/remainder arithmetic or a carefully bounded comparison) in both references and standalone examples. Add exact-boundary, just-over-boundary, zero-percent, 100-percent, and maximum-int vectors for both exercises. Keep integer rounding semantics explicit.

### P2 — Middleware/observability example contradicts its own cardinality guidance

**Where:** middle-32.md lines 29–34, 60–76, and 102–106; middle-32.json Medium contract; internal/grading/specs/middle-32-medium.json currently accepts route with a single positive vector.

The theory correctly says to use a **route template** and exclude raw URLs, IDs, and high-cardinality values. The exercise/reference allow the key route without defining that its value must be the registered template rather than a raw path such as /users/839274/tasks/991. A learner can satisfy the function grader while shipping the dangerous raw value; the one negative vector (authorization) does not distinguish those cases. Trace attributes can also expose identifiers or secrets even when their key is on an allowlist, so key-only validation is not a privacy guarantee.

**Minimum fix:** define the accepted value as a framework/router-produced route template, explicitly reject raw URL/path values and user-controlled attributes, and add realistic vectors for template vs. parameterized path. State clearly that an allowed key does not sanitize its value. Keep trace privacy separate from metric cardinality.

### P2 — The 14 chapters share an unrelated, misleading content scaffold

**Where:** JSON glossary/outcomes and Markdown opening sections in middle-27…40.{json,md}. For example, middle-28.md lines 8–18 lists authentication, IDOR, middleware, readiness, SLI/SLO, outbox and circuit breakers in a DI chapter; lines 24–34 then repeat material on authorization, middleware, fault injection, and postmortems. The same glossary and long cross-topic paragraphs recur across the range, including the metrics, SLO, outbox and TaskFlow chapters.

This is not just repeated terminology: the shared material crowds out the promised chapter-specific teaching. middle-28 promises application construction/DI, yet its three assessed contracts are (a) string length of a database URL, (b) first missing boolean dependency, and (c) conjunction of three booleans. No constructor, interface design, startup error, ownership/lifecycle, test seam, or executable composition is practiced. Similar off-topic paragraphs recur in chapters whose outcomes concern unrelated subjects. This weakens the course’s Russian beginner-friendly clarity and makes the outline look more comprehensive than the practice is.

**Minimum fix:** retain only terms needed by that chapter; explain the target concepts with a coherent TaskFlow example and Go code; remove repeated generic paragraphs or turn them into concise shared references. For middle-28, make at least one exercise build/validate a small dependency graph with constructor errors and a testable fake, while preserving explicit limits on what its unit test proves.

### P2 — Easy/Medium/Hard difficulty and the “Hard” label do not match Middle-level outcomes

**Where:** all 42 exercise definitions in middle-27…40.json; corresponding specs and internal/content/solutions/middle-27…40-*/main.go. Every task is a scalar-to-string/boolean classifier over a few arguments. Every Hard contract is still a single pure predicate/table (e.g. AuthorizeTask, CircuitAction, SafeExperiment, ReleaseEvidence, ReleaseDecision); none implements or tests a handler, repository, transaction, worker, real cancellation, concurrency behavior, or observable failure. Most are small enough to solve by transcribing the decision sentence. The matching 2,000-word chapters and lab cards ask for broader integration/review evidence, but the graded task does not collect it.

Examples: middle-38-hard reduces outbox idempotency to (key, alreadyApplied) -> string; middle-39-hard reduces unit/race/failure/load evidence to AND-ing four booleans; middle-40-hard reduces release acceptance to three booleans. Those functions cannot demonstrate durable idempotency, an actual race test, bounded load, rollback readiness, or reviewer approval. The generic exercise disclaimer correctly says a helper is not system proof, but the progression still presents these as Middle Easy/Medium/Hard and repeatedly names Hard an “инвариант безопасности.”

**Minimum fix:** keep small helpers only as warm-ups, then make Medium require a meaningful table-driven test or composed behavior and Hard require a bounded, executable project artifact with explicit evidence (isolated DB/handler path, worker retry/idempotency, cancellation/shutdown, or reviewed acceptance record). If a task remains only a helper, label it as a helper check and do not claim it assesses the named system-level outcome. A separate rubric/evidence object must be graded or explicitly marked as manual review rather than implied by the helper.

### P2 — Most private vectors are single examples and leave obvious decision branches unguarded

**Where:** internal/grading/specs/middle-27-*.json through middle-40-*.json. Many contain only one public and one private vector. Concrete gaps include:

- middle-31-easy: tests ready and started-but-not-ready, but not started=false (the first branch).
- middle-31-medium: tests request_id and authorization, but not route, status, duration_ms, empty/case-variant/unknown keys.
- middle-32-medium: tests only route and authorization; does not test method/status_code or a raw path value.
- middle-30-medium: no MaxInt64 addition overflow vector.
- middle-33-medium and middle-33-hard: no large-counter/overflow or exact threshold-boundary vectors (also exposes P1).
- middle-36-easy: the spec checks ordinary and missing project only; it does not exercise missing title or the 120/121 boundary.
- middle-37-easy: only editor-write and viewer-delete; owner, viewer-read, editor-delete and unknown action/role are not represented.

The source prose repeatedly tells the learner to test “границу и отрицательный случай,” but the evaluator itself often checks only a happy path and one rejection. These vectors therefore do not support the claim that plausible branch-swapping or boundary mutants are rejected.

**Minimum fix:** create a compact table-driven vector set for every contract: each branch, each precedence collision, equality at the boundary and just outside it, empty/unknown inputs where relevant. Add a mutation-oriented review pass that asks whether each stated rule can be inverted or omitted while still passing.

### P2 — Drain-deadline contract silently permits signed overflow

**Where:** middle-30.json Medium, internal/grading/specs/middle-30-medium.json, and internal/content/solutions/middle-30-medium/main.go.

The contract returns -1 only for negative inputs and otherwise says to return their sum. The reference performs nowMillis + graceMillis without overflow handling. Two valid nonnegative values near MaxInt64 can wrap to a negative deadline, invalidating the shutdown deadline. The sole private vector tests negative grace, not arithmetic boundary behavior. This is a relevant systems edge case for a Middle lesson.

**Minimum fix:** define overflow behavior explicitly (reject/sentinel or saturate), implement it without overflow, and add MaxInt64 boundary vectors. Explain whether wall-clock timestamps are actually represented as Unix milliseconds or whether a time.Time/time.Duration deadline is intended; the integer helper is only arithmetic, not a shutdown mechanism.

### P3 — User-facing title limit has undefined Unicode length semantics

**Where:** middle-36.json Easy, internal/grading/specs/middle-36-easy.json, and internal/content/solutions/middle-36-easy/main.go (uses len(title)). The contract says “длина >120” but does not say bytes, Unicode code points, or grapheme clusters. Since the course UI and examples are Russian, a 120-character Cyrillic title occupies more than 120 UTF-8 bytes and the reference rejects it. The supplied vectors use only ASCII and omit exactly-120/121 cases.

**Minimum fix:** define the API/storage limit in bytes or user-perceived characters; match implementation, validation and UI wording to that choice; add multibyte Russian and exact-boundary vectors. If bytes are intentional, say “байт UTF-8” explicitly.

## Cross-cutting limitations and disposition

- The standalone Go examples were inspected as text; they were not executed. No candidate solution, isolated runner, live service, or gate was run, as required by the task.
- P3 lab cards ask for integration and operational evidence, but the reviewed content does not define a machine-checkable artifact or an explicit assessment rubric that records that evidence. Where this is intended to be human-reviewed, state the reviewer, criteria, and rank impact; otherwise the helper exercises cannot be treated as proof of the chapter outcomes.
- No attempt was made to certify security of a real service. The chapters often disclaim that helper vectors are not production proof; those disclaimers are appropriate, but do not repair the curricular mismatch or arithmetic defects above.

**Disposition:** Middle-27…40 need revision before publication/acceptance as a complete Middle competency assessment. Fix P1 arithmetic first, then repair the cross-chapter scaffold, observability contract and per-lesson vector coverage; replace or clearly relabel the current helper-only Hard tier and attach actual project/assessment evidence.
