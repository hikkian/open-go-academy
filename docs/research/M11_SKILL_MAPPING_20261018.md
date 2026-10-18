# M11: skill/evidence traceability audit

**Проверено:** 2026-10-18. **Объём:** 13 канонических навыков, метаданные 88 черновиков, серверная запись учебных и проектных свидетельств, advanced rank policy. Это read-only анализ; он не заменяет полный M11 draft gate и педагогическую вычитку.

## Результат

`internal/competency/skills.go` задаёт 13 canonical IDs. Загрузчик каталога проверяет эти ID в каждом упражнении, не допускает неизвестные/повторные/пустые соответствия (`internal/content/skills.go`). Все 12 навыков, кроме `communication`, встречаются в исполняемых exercise mappings соответствующих рангов. Completed exercise без положительного серверного evidence не повышает уровень; XP и client assertions не используются (`internal/learning/catalog.go`). P1–P5 дают отдельные серверные automatic project evidence для тематических навыков. Их helper-уровень остаётся level 2 — свидетельство практики, а не production proficiency.

| Rank | Канонические тематические ID с локальными exercise mappings | Граница автоматического evidence |
|---|---|---|
| Zero | `program`, `output`, `functions` | Вводные задания и функции |
| Junior | `go-core`, `testing` | Практика Go core и тестов |
| Junior+ | `http`, `sql` | Модели API/SQL; не доказывают production-интеграцию сами по себе |
| Middle | `concurrency`, `reliability` (также темы `http`, `sql`, `architecture`) | Детерминированные задания и project evidence; не утверждают работоспособность внешнего брокера/Redis/БД |
| Middle+ | `performance`, `distributed` | Practice evidence; фактические workload/методология проверяются отдельно |
| Senior | `architecture`; также встречаются `distributed`, `performance`, `reliability` | P5 automatic evidence покрывает только архитектуру, не решение человека |

Ранги не вычисляются из skill level. Advanced award требует predecessor award, course requirements, project artifact, objective exam parts и закреплённый human-review quorum (`internal/httpapi/rank_policy_registration.go`, `internal/store/rank_policy.go`). `WithAwards` читает только сохранённые rank awards. Экзамен/проектный результат и skill matrix остаются отдельными сигналами.

## Communication: evidence и его границы

`communication` не имеет course exercise mapping — локальные helpers его не доказывают. Теперь уровень 4 появляется только при already-awarded Senior attempt, если его сохранённая policy/evidence snapshot связывает точный P5 artifact и exact manual-review quorum (минимум 2 различных разрешённых fingerprint). Store helper сверяет immutable review links, подписи/digests и project/rubric binding; `/api/learning` и passport используют этот результат. P5 automatic result и ярлык Senior без сохранённого P5 human-review evidence уровня не повышают. Новые текущие отзывы после награды не переписывают историческую snapshot; pending Start/Claim/Finish по-прежнему отклоняют superseded approval.

Свидетельство подтверждает решение держателей разрешённых ключей в данной локальной установке; оно не подтверждает настоящую личность, независимость проверяющих или внешнюю аккредитацию. В браузер отдаётся только доменно-хешированный непрозрачный ID, без reviewer ID/fingerprint. Отображаемый текст: «P5: итоговое ручное ревью Senior». Tutor получает лишь coarse skill level (cap 2), без ID доказательства или reviewer details.

## Проверки и ограничения

- `TestAllDraftCompetencies` и loader validation проверяют канонические mapping всех 88 черновиков; актуальный результат полного 88 gate фиксируется отдельно в `docs/validation/m11-current-tree-acceptance-20261018.txt`.
- `TestCatalogSkillsRequireBothCompletionAndEvidence`, `TestP3ProjectEvidenceUpdatesMiddleSkillsButNeverAwardsRank`, `TestP4P5AutomaticProjectEvidenceUpdatesSkillsButNeverAwardsRank`, `TestCommunicationRequiresOpaqueSeniorReviewEvidence`, `TestVerifiedSeniorCommunicationEvidenceForAcceptedMultiKeyAward`, `TestAdvancedRankPipelineRequiresProjectReviewAndAwardsMiddleMiddlePlusSenior` проверяют server evidence, двух/трёх reviewer keys, no-auto-P5, award→skill boundary, coarse-only Tutor context и сохранность accepted evidence после нового review.
- Текущий targeted reviewer code review не оставил actionable findings; affected package tests и полный `make test` PASS (Go unit/integration, P1 reference, vet, race, frontend typecheck/build). Повторный 88 gate на source fingerprint выполняется отдельно.
- Read-only traceability scan выполнен 2026-10-18; он не измеряет pedagogical quality/level и не заменяет human review курса.
- Не приписывать project/helper-level 2 как завершённую Senior-компетенцию; уровень 4 для communication приходит из exact awarded human-review snapshot, а не из одного project/helper результата.
