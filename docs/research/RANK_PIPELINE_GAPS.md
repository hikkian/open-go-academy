# Аудит конвейера рангов: текущие гарантии и разрывы

**Срез:** 2026-10-16. Это архитектурная проверка существующего кода и плана; код платформы не менялся. Цель — чтобы каждый ранг подтверждался цепочкой курса, проекта и всех объективных частей аттестации, а Senior дополнительно требовал содержательного P5 и реального human review.

## Что уже обеспечено

- Сейчас `assessment.All()` публикует ровно экзамены Zero, Junior и Junior+. Middle, Middle+ и Senior отсутствуют в каталоге, award не выдаётся — это правильное fail-closed поведение.
- Junior требует предыдущий award Zero, все перечисленные prerequisite exercises и автоматический результат P1; Junior+ требует award Junior, все prereqs вплоть до Junior+ и автоматические P1/P2 evidence. Они требуют 100% всех объективных quiz/task checks. Это criteria Go Academy, не подтверждение человеческой оценки качества проекта или профессионального стажа.
- `GET /api/exams` показывает недостающие rank/exercise prerequisites; `POST .../attempts` снова проверяет их на сервере. `StartExam` сохраняет ссылки на конкретные exercise attempt IDs и rank exam/version/attempt IDs в evidence snapshot.
- `ClaimExamAttempt` и `FinishExamAttempt` сверяют snapshot с текущими строками. Финальная транзакция связывает completed/pass, award и monotonic profile rank; server-owned scorer игнорирует клиентские score/rank. Credential issuance требует тот же точный passed attempt и persisted award.
- Упражнения M11, `p3:automatic` и текущее `manual_review_required` сами по себе не присваивают ранг. P3 API явно говорит, что архитектура, качество и эксплуатационные компромиссы остаются ручным review.
- Локальная подпись credential подтверждает целостность относительно ключа этой установки. Она не удостоверяет личность, авторство, должность, стаж, отсутствие списывания или аккредитацию; self-signed local credential не является доверенным human-reviewed Senior credential.

## Конкретные разрывы и будущие этапы

| Ранг | Уже есть | До award ещё требуется |
|---|---|---|
| Zero | 3 начальных exercise proofs и полный Zero exam | Спецификация разрешает пропустить остальную Zero-теорию знакомому студенту; это осознанное исключение. Если намерение пользователя изменится на «все уроки Zero», policy/version и prerequisites нужно расширить явно. |
| Junior | Все Zero+Junior exercise proofs, Zero award, P1 automatic, все quiz и coding checks | Уже не требует отдельного завершённого P1 manual review. Это корректно только если ранг явно назван автоматическим учебным критерием. |
| Junior+ | Все предыдущие proofs/award, P1+P2 automatic, полный objective exam | P2 human quality review не является prerequisites. Не считать текущий проектный checker доказательством архитектурного качества. |
| Middle | P3 TaskFlow каталожен условно и имеет isolated automatic acceptance; отдельный `p3:automatic` attempt | Нет опубликованного Middle exam/policy; нет server-side связи всех 40 Middle competencies/required course proofs + prior Junior+ award + P3 accepted evidence + все objective parts. Текущая P3 attempt запись не входит в `StartExam`/rank award. Ручное review P3 сейчас только предупреждение, не persisted review decision. |
| Middle+ | План P4 ScaleLab и curriculum | P4 project/evidence/checker, immutable exam policy, все Middle+ proofs + Middle award + P3/P4 chain, все objective parts. Никаких rank awards сейчас нет. |
| Senior | План P5 Architecture Capstone и заявленные критерии human review | P5 artifact package/checklist, versioned objective exam, prior Middle+ award и все required course evidence; главное — persisted human review с reviewer, scope, rubric version, decision, timestamps и точными artifact/version hashes. Все review criteria должны быть удовлетворены до Senior award. Никакой LLM verdict, learner self-review, helper exercise или подпись локальной установки не заменяет этого evidence. |

Текущие «все prerequisite lessons» реализованы как все IDs из `Exam.Prerequisites`, но набор существует только для трёх текущих экзаменов и задаётся exam catalog. Для будущего ранга нужно зафиксировать точную версионированную матрицу обязательных lesson/exercise competency IDs; не выводить её из XP, UI, `Profile.Rank` или неполного списка текущих упражнений.

`FinishExamAttempt` выдаёт automatic awards только для Zero/Junior/Junior+. Middle/Middle+/Senior намеренно блокируются `ErrRankEvidenceRequired`; добавление advanced exam в каталог само по себе rank не выдаёт. До открытия guard каждая advanced exam attempt должна иметь зарегистрированную immutable evidence policy с course/prior-rank/project requirements и отдельными objective checks.

## Безопасные точки расширения

1. **Immutable policy/catalog:** расширить версионированное exam definition явными `required_rank`, course competency/exercise IDs, project evidence kind/version и объективными parts; включить каждое поле/требование в hash. Отдельная rank eligibility policy должна задавать, какой вид доказательства обязателен, а не полагаться только на `Exam.Rank`.
2. **Project evidence:** отдельный store/API слой над сохранёнными проектными submissions. Сохранять attempt ID, project/checker/rubric version, pass result и artifact hash. Клиентский `passed`, произвольный `p3:automatic` идентификатор или ручной текст в HTTP-ответе не является доказательством.
3. **Human review:** отдельная server-owned review record/state machine для P5 (`submitted → pending → approved/rejected`), с reviewer identity/authority в пределах выбранной модели доверия, versioned rubric, artifact hashes, решение и audit trail. Self-review запрещён как qualifying decision; AI может подготовить обратную связь, но не выставлять approval.
4. **Transactional award:** eligibility повторно проверять из БД в транзакции award: prior award chain, все конкретные verified exercise rows, project evidence hashes/versions, все objective exam parts и (Senior) неизменённый approved review. В той же транзакции записывать финальный attempt status, award, profile rank и audit; credential выпускать лишь по этой точной award-ссылке.
5. **UI/catalog:** показывать каждый prerequisite как verified/missing/pending с доказательством и причиной блокировки. Не скрывать unavailable project/review gate и не показывать будущий ранг как earned.

Для P1/P2 текущая формулировка может сохранить автоматическую политику Junior/Junior+, если UI и credentials прямо указывают, что manual quality review не выполнен. Для Middle/Middle+ решение о необходимости human review следует закрепить immutable rubric; Senior human review — обязательный gate по текущему плану и намерению пользователя.

## Обязательные unit/API/negative тесты при реализации

- Каждый advanced rank закрыт по умолчанию: отсутствующий/неверный exam version, неизвестный rank, изменённый criteria hash, неизвестная competency или недоступный project profile не создают award.
- Для каждой ступени отдельно удалить по одной course proof, prior award, project submission, checker/rubric version, objective exam part и (Senior) human review; start/submit/award должны fail closed с объяснимым missing item.
- Передать поддельные client rank/score/pass/project ID/review decision/reviewer; убедиться, что все значения игнорируются или отклоняются.
- Сломанный/частичный objective exam, один failed Go task или низкий score никогда не создаёт award/credential.
- Project proof от другого профиля, старый submission, изменённый artifact после review, неправильный P3/P4/P5 ID, автоматический PASS без ожидаемого checker version — отклоняются.
- Senior: self-review, AI-решение, неподтверждённый reviewer, изменённый после review P5 artifact, pending/rejected review и несовпадающий rubric version не выдают award. Approved review должен ссылаться на точный P5 package/hash и сохраняться отдельно от auto score.
- Transaction rollback/fault injection: при ошибке записи любого prerequisite, completed attempt, award, profile rank, review audit или credential ни половинчатый award, ни credential не остаются.
- Replay/CAS/concurrency/restart: повторный submit, два одновременных завершения, отзыв/изменение доказательства между start→claim→finish, restart/reopen и pruning не обходят eligibility и не теряют award-linked evidence.
- Credential для каждого ранга ссылается на точный award/exam/policy version; приватность/revoke работают. Локальный verifier обязательно показывает `local-self-issued`/self-signed notice и не отображает identity-verified/accredited.

**Вывод на дату исходного среза:** Junior/Junior+ — автоматические учебные цепочки по перечисленным упражнениям и P1/P2 checks, а не «один экзамен». Middle/Middle+/Senior не выдаются. Маленькие helper tests — учебная практика, не Senior evidence.

## Addendum — signed offline review groundwork — 2026-10-16

After this read-only snapshot, an Ed25519 offline review workflow was added: API packets bind exact project submission/evaluation/artifact/rubric, the operator registry verifies configured public keys, and append-only decisions can be stored and read from the workspace. A later store-only M11 slice adds optional manual review requirements to immutable `ProjectEvidenceRequirement`; it pins exact allowed key fingerprints and trust hash, snapshots approved exact-artifact reviews in append-only FK-backed `exam_project_review_evidence`, chooses the newest candidate with current quorum, and revalidates latest decisions on Claim/Finish. Three independent P2 findings (unbound trust hash, reject-after-approval, and unreviewed-newer-submission shadowing) were fixed and re-reviewed with no remaining findings in the store scope. See `docs/validation/manual-review-rank-store-progress-20261017.txt`.

**Current remaining gaps:** API does not yet construct policies from the active operator registry or register policies for the full course; policy trust fingerprints are not yet bound to startup configuration. The `automaticExamRank` guard still blocks all Middle/Middle+/Senior awards. All 88 M11 lessons remain draft pending pedagogical review. Fingerprints prove key bytes only, not identity, independence, or separation from the learner. Senior review must remain local/operator-attested unless an external identity/authority model is introduced. A future API/browser test must exercise manual review → course/prior/project eligibility → objective exam → transactional award and credential, plus rejection/replay/revocation/restart failures.

## Update — API integration and evidence-bound awards — 2026-10-17

The preceding store-only addendum is a historical snapshot. Current API setup now registers immutable policies for Middle/P3, Middle+/P4 and Senior/P5 only when the operator reviewer registry, exact manual-rubric hashes and project-runner profile match. Policy requirements include objective exam course prerequisites, prior rank and exact passing project evidence/review quorum. Changes in reviewer keys, rubric or evaluator profile produce a new immutable policy version; stale active attempts fail closed. The store advanced-award guard is conditionally open only for a valid pinned policy and transactionally revalidated evidence.

Evidence: [`advanced-rank-pipeline-progress-20261017.txt`](../validation/advanced-rank-pipeline-progress-20261017.txt). HTTP tests cover the three rank transitions, rejection supersession, quorum, failed exam without award, exact successful award and replay denial. They seed prior rank/course proofs and stub the project checker; they do not validate reviewer identity or OCI isolation. UI surfaces missing review and explains that a key fingerprint is not identity/independence. M11 pedagogy and rank-course mapping remain open; 88 lesson bundles stay drafts. An operator-supplied local registry cannot prove reviewer identity, independence, or separation from the learner. Do not describe these local credentials as publicly trusted ranks.
