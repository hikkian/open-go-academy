# Независимое ревью каталога экзаменов

Дата: 2026-10-16. Ревьюер: настроенный `gpt-6.1-sol`. Scope: immutable multi-exam catalog, API draft/submit/credentials, transactional exercise/rank prerequisites, generic private grading и Spec.Fingerprint. Reviewer меняет только этот отчёт; production code не редактировал. Подагенты, hostile programs и эксплуатационные probes не запускались. Применены ранее прочитанные audit-context-building/differential-review инструкции.

## Итог

**Найденные Medium замечания AC-R1–R3 исправлены и CLOSED; открытых High/Medium в проверенной области нет.** Пинованный focused suite четырёх пакетов PASS. Native chain новых экзаменов и browser E2E reviewer не запускал; соответствующий opt-in test без флага пропущен. Результат не заменяет независимый полный аудит либо проверку всех входов функций.

## Контекст и контракт

Прочитаны `internal/assessment/{assessment,newcatalog}.go` и tests, `internal/httpapi/{exams,credentials}.go`, `exam_catalog_test.go`, прежние exam tests, `internal/store/exams.go` и tests, generic grading/Fingerprint и релевантные новые function specs/references. Код одновременно дорабатывался авторами; статусы ниже относятся к последнему повторному проходу.

Согласованная родителем policy v1: Junior/Junior+ — автоматические учебные criteria Go Academy: все указанные course exercise proofs, предыдущий persisted rank award, P1/P2 automatic project evidence,20 objective вопросов и3 coding/debugging задачи. Это не подтверждение manual project quality или профессионального стажа. Project manual_review_required остаётся отдельным неподтверждённым milestone; новые Criteria/Description явно оговаривают пределы, а credential берёт эти серверные criteria. Эта policy принята как документированное уточнение области, а не выдана за human review.

Zero `zero-core@1.0.0` остаётся прежним immutable материалом. Golden hash: `bd2cc6db2c954ef3410db69039bcc12cb15085098194686e9883d1e7b70f2d53`. Новые `junior-core@1.0.0` и `juniorplus-api@1.0.0` получают собственные grade fingerprints; их материалы/guards менялись до первого production registration, по сообщению интегратора. После регистрации любое изменение требует новой версии, иначе RegisterExam fail-closed.

## AC-R1 — Medium, CLOSED: legacy private stderr выходил через checks

**Путь:** `internal/httpapi/exams.go`, legacy Zero branch.

Исходная ветка подставляла actual.Stderr в check.Detail при private case failure. Этот текст мог включать private input и попадал в HTTP response и persisted ChecksJSON. В новой generic ветке raw output уже не раскрывался.

**Исправление/evidence:** legacy detail теперь только generic failure, raw stderr не копируется. TestLegacyExamPrivateOutputNeverLeavesGrading использует обычный fake result с PRIVATE_CASE_MARKER в stdout/stderr, проверяет отсутствие marker в response/ChecksJSON и отсутствие pass. Fixture прочитан и прошёл в final HTTP suite. Zero Hash не затронут, поскольку HTTP diagnostics не входят в immutable assessment payload. CLOSED.

## AC-R2 — Medium, CLOSED: неизменность входа/встроенный append не защищались

**Пути:** transitions task и `junior-exam-count-transitions.json`; `internal/grading/grading.go`, predeclared.

Task обещал «вход нельзя менять», но initial driver проверял только count. После добавления before clone через builtin append обнаружен второй статический пробел: ручной список predeclared не содержал append/cap, поэтому harness не должен был полагаться на их непереопределённость. Reviewer не создавал/не запускал эксплуатационную программу.

**Исправление/evidence:** driver выдаёт `[answer, unchanged]`, сохраняет самостоятельный before slice и сравнивает через slices.Equal; expected требуютtrue. Обычный bounded TestTransitionsRejectsMutationWithCorrectAnswer проверяет отклонение функции с верным count, меняющей элемент; это доверенный отрицательный учебный fixture. Predeclared lookup теперь использует types.Universe для всех builtins/types/constants, AST fixtures с append/cap отвергаются без исполнения. Source policy fingerprint явно обновлён до function-source-policy-v2. Final assessment/grading suites PASS. CLOSED.

## AC-R3 — Medium, CLOSED: unknown-role vectors не различали owner shortcut

**Путь:** `juniorplus-exam-access-policy.json` и authored CanAccessResource contract.

Ужесточённый контракт запрещает unknown role даже владельцу. Первые unknown-role cases имели разные owner/actor ID, поэтому не проверяли именно эту ветку. Это узкое отсутствие отрицательного примера для явно заявленного условия.

**Исправление/evidence:** добавлены ownerID=actorID=4 с role=root и пустой role, обаfalse. Reference сначала требует положительные owner/actor ID, затем ровно admin либо user+owner equality. Авторский reference и все production private checks прошли. CLOSED. Это не доказательство полноты для произвольных policy expressions.

## Проверенные границы API и authority

- Catalog All/Find возвращает три версии; GET/start выбирает exam ID, draft/submit — stored attempt.ExamID+Version. Неизвестная версия не разрешается; перед draft/submit проверяется CriteriaHash. Client answers/task IDs проверяются относительно найденного экзамена, foreign task/answer keys отвергаются.
- Начало принимает пустой object, не client prerequisites/rank/score. API передаёт trusted prerequisites и RequiredRank. Nonce32random bytes хранится как SHA-256; claim проверяет token/status/revision в транзакции и переводит active→grading. Все разновидности экзамена используют один state machine; replay не запускает второй grade. Draft CAS не изменяет grading/completed snapshot.
- StartExam snapshots exercise→attempt_id и required rank→exam/version/attempt_id из persisted rank_awards того же local profile. Profile.Rank/XP не заменяют rank award. Claim повторно проверяет оба набора evidence; Finish перед выдачей pass также rechecks snapshot и registered rank/hash. Tests удаляют prior award до claim и после claim: обе границы fail-closed, финальная транзакция не создаёт новый award.
- Finish требует score100↔passed, statusgrading, matching registered immutable criteria. Completion, rank award и monotonic profile rank update атомарны. Single-open-attempt index и retention award-linked evidence сохранены. JSON проверок — результат trusted API evaluation, не входящий client verdict.
- Generic task input/driver направляется только isolated runner. Expected остаются на API; raw runtime stdout/stderr/files не возвращаются и не попадают в Check.Detail. Сохраняются submitted answers/source и generic check results. AST diagnostics содержат ученический source, а не private inputs. Не было нового пути user execution на API host.
- Credential issue теперь Find по точным attempt exam/version, проверяет completed/passed100/hash и подписывает server exam.Criteria. Store IssueCredential дополнительно JOIN-проверяет persisted rank_award/attempt/registered criteria. Клиент не задаёт issuer/key/rank/criteria; прежние pinned-key export/verify guards сохранены.

## Fingerprint и новые security-контракты

Spec.Fingerprint непуст только после BindReference; canonical JSON включает source-policy-v2, ID/signature/import allowlist, trusted driver, public/private inputs и expected, transitive bound type definitions. Exam.Hash новых versions добавляет RequiredRank и отсортированные task→fingerprint pairs к полному base payload с criteria/questions/private answer keys/prerequisites. Результат устойчив к map iteration; public DTO не публикует private spec data. Semantic changes source policy должны сопровождаться bump её version; автоматическая совместимость будущих toolchains не заявляется.

Authz contract проверен статически: IDs>0, role ровно user/admin, owner shortcut только user. Redirect contract теперь задаёт PATH-only, valid UTF-8, одно percent decoding, запрет query/fragment/residual percent/backslash/control/leading double slash и dot segments; reference и public/private cases соответствуют этому узкому контракту. Это чистые учебные функции, не обещание универсального web URL sanitizer. Retry status — ровно429/500..599. Golden Zero hash PASS после этих additions.

## Фактически выполнено

```sh
./scripts/go.sh test -count=1 -timeout=45s ./internal/httpapi ./internal/store ./internal/assessment ./internal/grading
```

Последний повтор после fixes: HTTP0.402s, store0.255s, assessment1.394s, grading0.201s — PASS без test cache. Предыдущие промежуточные проходы также PASS, но не заменяют final repeat. В suite запускаются только trusted references и bounded обычный отрицательный function fixture; произвольный ученический или враждебный код на host не запускался reviewer.

Прочитан TestNewExamRankPrerequisiteAndCrossExamDraft: все exercise proofs и искусственно высокий XP не обходят предыдущий rank gate; cross-exam draft422, unknown version404. Прочитаны store rank snapshot/recheck tests и Golden Zero/Fingerprint mutation tests. Native TestNativeJuniorAndJuniorPlusAssessmentChain — opt-in; без ACADEMY_EXAM_CATALOG_NATIVE=1 пропущен. Он явно seed-ит previous Zero и prerequisites как test-only fixtures; их нельзя считать фактическим прохождением курса. Новые Junior/Junior+ grading и credentials в этом native test reviewer не выполнял.

Остаточные evidence limits на момент этого прохода: generic private marker/new-exam wrong nonce/completed draft/changed hash и all-three-catalog assertions родитель расширяет отдельными обычными HTTP tests; на прочитанном снимке часть этих комбинаций опирается на прежние state-machine/store tests и статическое чтение новой dispatch-ветки. Parent выполняет native6-task chain и browser E2E отдельно. Это не открытые выявленные code defects, но границы сделанных reviewer проверок. Нативные/браузерные результаты не присвоены себе.
