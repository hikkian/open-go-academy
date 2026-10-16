# P5 rubric — architecture capstone

Критерии оценивают конкретную версию артефактов, а не должность, стаж или профессиональную пригодность. Всего 100 баллов; автоматическая часть и человеческая часть хранятся раздельно.

## Машинно наблюдаемое — 40 баллов

| Область | Баллы | Наблюдаемая проверка |
|---|---:|---|
| Prototype compiles and deterministic tests | 12 | Pinned Go build/test/vet; concurrent duplicate, lease expiry and stale fencing cases; no race in model |
| Requirements and traceability structure | 6 | Every listed unknown has question/assumption/owner/validation; acceptance criteria link to requirement IDs |
| Architecture artifact package | 6 | Diagram/source file present; APIs/events/data invariants and trust boundaries are addressable |
| Operations evidence | 8 | Explicit SLI/SLO units, capacity equations, rollout/rollback, restore and incident actions are present |
| Reproducibility and integrity | 8 | Commands and environment stated; source/document snapshot hash generated; machine report bound to same hash |

Machine checks cannot infer the correctness of an architecture from headings, diagram syntax or a passing in-memory test. A TODO-free artifact can still be wrong.

## Экспертная оценка — 60 баллов

| Область | Баллы | Вопросы для review |
|---|---:|---|
| Requirements reasoning and scope | 10 | Distinguishes facts from assumptions; asks useful questions; avoids unsupported SLO claims |
| Architecture and alternatives | 15 | Compares at least two viable options; describes consistency, failure and migration trade-offs; decisions trace to requirements |
| Reliability and capacity | 12 | Model matches failure modes; capacity assumptions are dimensionally coherent; RPO/RTO/SLO and operational response can be tested |
| Security, privacy and tenancy | 10 | Trust boundaries and residual risks are credible; controls are enforceable and testable |
| Defense and learning | 13 | Author can explain and revise choices under a new constraint; evidence is their own and review discussion is recorded |

## Блокирующие gates для Senior rank

All automated mandatory invariants must pass, but that is only eligibility evidence. In addition, a **separate independent, authenticated human reviewer** must inspect the exact immutable artifact hash, conduct a live defense, fill this versioned rubric and record reviewer identity, timestamp, decision, conflicts/independence attestation and comments. The reviewer must not be the learner. Neither AI output, AI-generated score, another automated checker, learner self-review, nor a missing reviewer may approve or substitute for this gate. Until the platform can verify and persist this signed review record, P5 evidence must remain `pending_human_review` and Senior rank must fail closed.

Passing score is at least 75/100 and all blocking gates. A high score cannot waive a failed mandatory safety invariant or absent independent review. Reference artifacts do not receive a learner's rank.
