# P3 TaskFlow objective rubric

This rubric checks reproducible project behavior, not the learner's identity, personality, employment, job title, years of experience, or professional competence outside these artifacts. A missing human review is reported as unreviewed; the automated score is not a job credential.

| Area | Points | Observable evidence |
|---|---:|---|
| Data model and migrations | 15 | Versioned up/down migration, FK/check/unique/index invariants, fresh-database setup, documented cardinalities |
| Identity and tenant RBAC | 20 | Hashed random session, server-owned actor, owner/editor/viewer matrix, denied cross-tenant read/write paths |
| HTTP and SQL boundary | 15 | Strict bounded JSON, parameterized values, safe errors, stable OpenAPI, list/body/time limits |
| Task correctness | 15 | CRUD slice, optimistic version conflict, comment ownership, idempotency-key replay and collision behavior |
| Transactional outbox | 15 | Atomic task+event, concurrent claim exclusion, unique effect receipt, crash/lease recovery |
| Worker operations | 10 | Bounded workers/batches/retries, terminal failure, shutdown drain and cancellation checks |
| Verification and operations | 10 | Test/vet/race/build, real disposable-Postgres integration, fixed-size load/failure evidence, pinned local Compose, threat notes |

**Automated acceptance gate:** each invariant must pass against a clean disposable database; cross-tenant data must not appear; wrong role and stale version must be denied; same authorized idempotency key must not duplicate a task/event; repeated worker delivery must not duplicate the receipt; an invalid event must retry then reach `dead`; student code is not assumed safe merely because the trusted reference passed. The exact checker provides the authoritative machine-readable tests. This rubric does not turn static review observations into claims of production readiness.

**Evidence record:** record Go version, exact commands, test output, local database image digest/platform, fixed workload size/timeout, and any failed/skipped test. Performance results are valid only for their stated machine and workload; no universal throughput threshold is implied.

## Ручная проверка Middle: обязательные результаты M27–40

Авторитетная подписываемая rubric выдаётся API в review packet; для P3 используется `p3-taskflow-v1-manual-review-v2`. Этот файл поясняет её критерии и сам по себе не является решением проверяющего.

- **M27:** матрица actor/tenant/role и отрицательные IDOR-сценарии с реальными HTTP/SQL результатами.
- **M28:** типизированный граф handler → service → repository, composition root, владение ресурсами и cleanup; fake-тесты должны проверять экземпляры, вызовы и аргументы.
- **M29:** middleware, deadline/cancellation, bounded request body и безопасные ошибки/логи; воспроизводимый отменённый запрос и отказ зависимости.
- **M30–32:** readiness/liveness, bounded shutdown/drain и фактически полученные logs/metrics/traces с источниками и ограниченной cardinality.
- **M33:** SLI с числителем/знаменателем/окном, пропусками сигналов, SLO и error budget; фиксированный workload, наблюдённые ошибки и latency, параметры машины.
- **M34–35:** retry/timeout budget, выбор circuit breaker или обоснованной альтернативы; fault injection, восстановление, идемпотентность и освобождение ресурсов.
- **M36–40:** acceptance record с версией и hash артефакта, schema/API, test/vet/race/build, нагрузкой, failure/recovery, rollback-планом, совместимостью данных и оставшимися рисками.

`approve` означает, что проверяющий изучил весь exact artifact и каждый обязательный критерий; отсутствующее или невоспроизводимое evidence требует `changes_requested`/`reject`. Ключ и подпись связывают решение с настроенным проверяющим, но сами не устанавливают его личность или независимость. Учебные helper-функции остаются подготовкой и не заменяют эти проектные доказательства.
