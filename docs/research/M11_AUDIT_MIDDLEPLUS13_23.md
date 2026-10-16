# M11 audit: Middle+13–23

**Scope:** read-only review of the 11 draft chapters, their grading specs and reference solutions. No candidate submissions, runner, or acceptance gate were executed. Findings below come from source inspection and hand-checked counterexamples. I did not edit curriculum source, specs, or solutions.

## Findings

### P1 — SLO arithmetic overflows for valid `int` inputs

**Affected:** `middleplus-21-medium` and `middleplus-21-hard`; also the matching examples/reference solutions.

The comparison `failed*100 <= total*maxErrorPercent` overflows signed `int` for large valid counts. The prompt permits any `total>0`, `0<=failed<=total`, and percentages from 0 to 100; it does not cap counts. On the intended 64-bit target, `SLOMeets(MaxInt, MaxInt, 0)` wraps the left product to `-100` and returns `true`, although a 100% error rate cannot meet a 0% budget. The same expression appears in the lesson examples, so learners are taught the faulty calculation as well as graded against it.

Evidence: [middleplus-21.md](../../internal/content/data/middleplus-21.md#L61) and [middleplus-21.md](../../internal/content/data/middleplus-21.md#L77), [medium spec](../../internal/grading/specs/middleplus-21-medium.json), [hard spec](../../internal/grading/specs/middleplus-21-hard.json), [medium reference](../../internal/content/solutions/middleplus-21-medium/main.go#L3), [hard reference](../../internal/content/solutions/middleplus-21-hard/main.go#L3). Existing specs test only 100/1/2 and one negative-total case, so they cannot catch this.

**Minimal fix:** define overflow-safe semantics in the prompt and calculate the maximum allowed failures without multiplying the full counts (for percentage `p`, use `floor(total/100)*p + floor((total%100)*p/100)` after validating `p`). Add MaxInt vectors including `failed=total, p=0`, exact budget boundary, and one above it. Keep the examples, reference solutions, and expected results in sync.

### P2 — The shared “practical result” paragraph is copy-pasted and nonsensical in every chapter

**Affected:** all `middleplus-13` through `middleplus-23` markdown files, line 22. The paragraph says each topic (“capacity”, “backpressure”, “queue depth”, “retry”, “outbox”, “overload”, “fault injection”, “slo”, “rollout”, and “postmortem”) “safely accepts work, stops accepting it, or changes state,” followed by a generic sentence that “the word scaling” does not define that condition. This is unrelated to several subjects and grammatically wrong for `workers`; for SLO/postmortem it also assigns behavior to an abstraction that does not accept requests. Example: [middleplus-21.md](../../internal/content/data/middleplus-21.md#L22), where SLO is described as accepting work; compare the actual SLI/SLO goal at line 4. The same template is present at line 22 in the other ten files.

**Minimal fix:** replace the paragraph per lesson with a short explanation of that chapter’s actual decision/problem (or remove it). Search the whole 13–23 group for the shared “Само слово «масштабирование»” sentence after repair.

### P2 — Middle+ assessment does not assess the advertised engineering outcomes

**Affected:** all 11 chapters; especially fairness in 20, operational outbox semantics in 17, canary decisions in 22, and postmortem practice in 23.

The lesson bodies discuss capacity measurement, backpressure, retries, transactional outbox, overload, fault injection, worker fairness, SLI/SLO windows, rollout safeguards, and incident analysis. Yet each chapter’s three graded tasks are separate stateless helper functions over small scalar inputs. They do not ask the learner to produce or explain a reproducible measurement, transaction/recovery state machine, scheduling/fairness result, telemetry window, safe rollout decision under missing signals, or incident timeline/action item. Examples:

- `middleplus-20` advertises worker-pool fairness, but `WorkerMode`, `WorkerCapacity`, and `WorkerSafe` only classify counts; no tenant/order/starvation input exists.
- `middleplus-17` describes a transactional outbox, while `OutboxAction`/`OutboxState`/`OutboxSafe` accept only booleans and attempt count; no atomicity, duplicate delivery, idempotent effect, or recovery transition is exercised.
- `middleplus-23` teaches timeline and system-level postmortem, but the tasks only classify impact/ongoing/mitigation flags; no events, timeline, evidence, or action item is modeled.
- Similar gaps apply to capacity (13), queue age and measurement (15), retry budgets/deadlines (16), overload policy (18), fault injection/recovery (19), SLI/SLO observation windows (21), and rollout telemetry/rollback (22).

The prose often explicitly warns that a helper is not an integration or production guarantee, which is good calibration, but the current grading still cannot substantiate the stated learning outcomes or a 180-minute Middle+ chapter. This matters if completion/rank is meant to demonstrate mastery rather than merely function-syntax fluency.

**Minimal fix:** retain small functions for syntax practice, but add at least one chapter-specific assessed artifact per topic (for example, a deterministic scenario/table-driven state machine plus a short evidence-based explanation). Make the contract evaluate the central objective named in that lesson, and calibrate estimated time to the actual work.

### P2 — Most private test sets are too small to reject obvious boundary/branch mutants

**Affected:** 31 of 33 exercises have exactly one public and one private vector; the two exceptions are `middleplus-13-medium` and `middleplus-13-hard`, whose capacity vectors cover reserve boundaries and MaxInt. In many of the other exercises the private vector is simply a negative value, while the public vector is an ordinary successful case. This is not enough to distinguish the intended policy branches.

Concrete uncovered branches include:

- `middleplus-15-medium/hard`: full queue (`depth==capacity`) and oldest-age threshold (`oldestSeconds==60`) are absent; the sole private input is negative depth.
- `middleplus-17-medium/hard`: published/acknowledged and retry-limit/dead-letter states are absent; no negative-attempt case.
- `middleplus-18-medium/hard`: over-capacity noncritical shedding is absent; current public case is over-capacity and critical, which takes the prioritize path.
- `middleplus-19-medium/hard`: before-commit and after-commit branches are absent; the only public case is publish/retryable, private is unknown.
- `middleplus-21-medium/hard`: no valid above-budget or exact-boundary vector; only a 1%/2% success and invalid negative total.
- `middleplus-22-medium/hard`: no insufficient-sample or rollback vector; only an under-threshold sample of 120 and invalid negative error percentage.
- `middleplus-23-medium/hard`: no positive-impact ongoing incident case; a mutant that ignores `ongoing` can pass the current hard vectors.

Evidence is in the corresponding JSON under `internal/grading/specs/middleplus-13..23-*.json`; each contains the input and expected output. The same sparse vectors are reflected in many E/M/H families, so a candidate can pass without demonstrating the branch/principle emphasized by the prose.

**Minimal fix:** add compact, orthogonal vectors for each contract: ordinary success, exact threshold, just across threshold, invalid values, and every distinct policy branch. For each chapter, include at least one vector that distinguishes a plausible wrong implementation from the reference; do not merely repeat the same one valid/one invalid pattern for Easy, Medium, and Hard.

## Per-chapter coverage snapshot

| Chapter | Main subject in text | What current graded functions actually cover | Audit note |
|---|---|---|---|
| 13 | Capacity, reserve, measurement limits | Classification; quotient/remainder reserve calculation; capacity predicate | Strongest coverage in this group; arithmetic boundaries are explicit. |
| 14 | Backpressure and controlled rejection | Queue depth/limit classification; wait/shed label | No wait/deadline/caller behavior; medium does at least test the full boundary. |
| 15 | Queue depth, age, latency | Small count/age threshold predicates | Threshold/full branches are absent from private vectors. |
| 16 | Retry, backoff, retry budget | Retry label; capped delay; retry eligibility | Backoff calculation is present, but its edge/overflow/large-attempt contract is weakly tested. |
| 17 | Transactional outbox and publication | Boolean publication state and retry count | No transaction or duplicate-effect property is verified. |
| 18 | Overload and load shedding | Accept/prioritize/shed predicate | Critical/noncritical overload split is not covered by vectors. |
| 19 | Fault injection and recovery | Named-stage mapping and retry eligibility | No actual failure sequence or recovery invariant is executed. |
| 20 | Worker pool, concurrency limit, fairness | Worker/queue count arithmetic | Fairness/starvation is not represented in inputs or assertions. |
| 21 | SLI/SLO and error budget | Error percentage threshold | Valid-input arithmetic bug (P1); windows and missing telemetry not modeled. |
| 22 | Canary rollout and rollback | Error percentage and fixed sample-count threshold | No rollback/insufficient-sample vector; no window or missing-signal policy. |
| 23 | Postmortem timeline and systemic measures | Impact/ongoing/mitigation predicate | Timeline/evidence/action quality is not assessed. |

## Chapters with no direct content in the audited scope

All 11 chapters are present, each has JSON, markdown, Easy/Medium/Hard grading specs, and matching reference solutions. No chapter is wholly missing. The gaps above concern depth and coverage of the advertised skill rather than missing files.
