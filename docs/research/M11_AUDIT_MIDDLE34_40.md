# M11 audit: Middle-34…40

**Scope:** read-only review of the seven Middle chapter Markdown/JSON files, all 21 grading specs, inline starter code, and the 21 checked-in reference solutions. The aggregate gate source was inspected to understand what it verifies; no gate or Go code was run. This is a content and assessment review, not a runtime/security certification.

## Findings

### P1 — The assessed “Hard” contracts do not assess the stated Middle outcomes

**Files:** `internal/content/data/middle-34…40.{md,json}`, `internal/grading/specs/middle-34…40-hard.json`, `internal/content/solutions/middle-34…40-hard/main.go`.

Each Hard task is a pure scalar classifier/boolean conjunction. Examples: `CircuitAction` only maps four input fields to a label; `SafeExperiment` ANDs three booleans; `IdempotencyResult` trusts a caller-supplied `alreadyApplied` flag; `ReleaseEvidence` ANDs four booleans; `ReleaseDecision` trusts three booleans. None exercises retry scheduling/cancellation, a breaker or bulkhead under concurrency, an injected failure against a disposable dependency, a durable outbox/worker, race/shutdown behavior, or a reviewable release artifact. The prompts themselves say these helpers do not perform those checks, but the difficulty title still calls each one an “инвариант безопасности.”

This is especially material in Middle-38…40: `alreadyApplied=true` does not demonstrate durable deduplication, the four booleans in `ReleaseEvidence` do not run the named tests, and the `reviewerApproved` function argument is not an independently recorded approval. The prose/lab cards describe the missing evidence, but the production grader consumes none of it.

**Recommendation:** keep such functions as clearly labelled warm-ups. Make the Middle-level assessment require a bounded runnable artifact and evidence (e.g. deterministic fake-clock retry policy, stateful worker/outbox exercise, a real shutdown test, or versioned acceptance record). If review is manual, capture rubric, reviewer decision and rank impact as explicit evidence. Do not describe a helper result as a security invariant.

### P2 — All grader specs have only one public and one private vector

**Files:** `internal/grading/specs/middle-34-{easy,medium,hard}.json` through `middle-40-{easy,medium,hard}.json`.

All 21 contracts have exactly one public and one private input. Several have more decision branches than those two examples can meaningfully exercise. For instance, Middle-34 Easy leaves invalid ranges and permanent errors untested; Medium leaves negative inputs, zero caps, and exact saturation untested; Hard leaves deadline precedence, full bulkhead, half-open, and invalid states untested. Middle-35 leaves the other failure stages and negative impact untested. Middle-36 leaves title missing/length boundaries and most transition edges untested. Middle-37 `RoleCan` tests editor/write and viewer/delete only, leaving owner, viewer/read, editor/delete, and unknown role/action uncovered. Middle-38 leaves outbox branches and worker ack/dead-letter/attempt boundaries uncovered. Middle-39 leaves invalid/zero/draining boundaries and every one-false `ReleaseEvidence` case uncovered. Middle-40 leaves count overrun, incomplete, risk precedence, and each false release prerequisite uncovered.

The prose repeatedly directs the student to test boundaries and negative cases, but the hidden vectors do not verify most of those branches. A branch omission or precedence mutation can therefore pass the grader.

**Recommendation:** give every contract a compact table-driven set covering each branch, precedence collision, equality and one-step-outside boundaries, and empty/unknown inputs where applicable. Add mutation-oriented review against the contract.

### P2 — Generic shared lesson scaffold remains in all seven chapters

**Files:** `internal/content/data/middle-34.md`…`middle-40.md` lines 7–28; matching `.json` `glossary` and `learning_outcomes`.

Every chapter repeats an almost identical glossary of authentication, authorization, tenant, IDOR, DI, middleware, readiness, SLI/SLO, outbox and bulkhead/circuit-breaker. The opening lesson text also repeats broad TaskFlow/DI/middleware/shutdown material in chapters whose outcomes are retries, incident review, schema, CRUD, outbox, load testing, or acceptance. For example, Middle-34 starts its domain theory with DI/middleware/shutdown; Middle-35 repeats it before discussing fault injection; Middle-39 again spends its opening on unrelated DI/middleware content. The glossary includes terms irrelevant to each chapter (e.g. outbox in the retry chapter, and authentication/IDOR in load/shutdown). This crowds the chapter-specific teaching and makes the same content look newly taught seven times.

**Recommendation:** retain shared architecture principles as a short linked reference; give each chapter a focused glossary and use its word budget for a coherent, progressively developed TaskFlow example aligned to its outcomes.

### P2 — Middle-36 title-length contract is ambiguous for Russian/UTF-8 text

**Files:** `internal/content/data/middle-36.json` Easy exercise, `internal/grading/specs/middle-36-easy.json`, and `internal/content/solutions/middle-36-easy/main.go` (`TaskInput`).

The contract says title length `>120`; the reference uses Go `len(title)`, which counts UTF-8 bytes. That rejects a Russian title well before 120 Unicode code points. Neither the learner prompt nor vectors say whether the persisted/API limit means UTF-8 bytes, code points, or grapheme clusters. The single positive/negative vector set has no multibyte or 120/121 boundary case.

**Recommendation:** state the storage/API unit explicitly (bytes or user-perceived characters), make implementation/UI wording agree, and test multibyte Russian at the exact boundary and just beyond it.

### P3 — Middle-35 prerequisite metadata duplicates the previous chapter

**File:** `internal/content/data/middle-35.json`, fields `prerequisites` and `order`.

The chapter order is 75, after Middle-34 at 74, but prerequisites contain both `middle-34` and the redundant phrase “previous chapter `middle-34`” in the Markdown. This is not harmful, but it weakens dependency metadata as a signal distinct from simple sequence. (Middle-39 and Middle-40 similarly name the immediately prior chapter as a prerequisite; retain this only if intentional.)

**Recommendation:** distinguish true conceptual prerequisites from linear order, or document that every prior chapter is intentionally a required dependency.

## Disposition of the broader Middle-27…40 audit

The earlier audit's cross-cutting findings still apply to this subset: generic shared scaffold, helper-only Hard tasks, and sparse grading vectors. Its specific Middle-30/33 overflow and Middle-27…32 assessment findings are outside this subset; their remediation reports say those scoped gates passed, so they should not be attributed to Middle-34…40. The Unicode-length concern previously identified for Middle-36 still applies and has not been resolved in the reviewed files. No evidence in the supplied scoped remediation reports indicates that chapters 34–40 received content or vector remediation.

## Gate coverage and limits

`internal/content/m11_acceptance_test.go` validates candidate catalog loading, lesson word-count bounds, runnable documentation examples, three exercises, starter compilation, and each private reference against its production grader via the isolated runner. That is useful integrity coverage, but it does not evaluate whether lesson prose matches the chapter outcome, whether vector sets kill likely mutants, or whether the Hard task collects the project/review evidence described by the chapter. The acceptance source was read only; this audit did not run it.

## Remediation status update — 2026-10-16

The scoped audit findings for Middle-34…40 were addressed and the pinned/offline production acceptance passed in 260.841 s. All 21 graders now have broader branch/boundary vectors, lesson theory/glossaries are chapter-specific, Middle-36 defines the 120-byte UTF-8 limit, and Hard helpers are explicitly labelled as practice rather than rank evidence. See `docs/validation/m11-middle34-40-remediation.txt`. The P1 artifact/system-evidence gap remains open by design and must be closed in the project/rank assessment pipeline; this scoped remediation does not substantiate a Middle rank.
