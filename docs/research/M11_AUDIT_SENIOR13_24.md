# M11 audit: Senior-13…24 / P5 Architecture Capstone

**Дата:** 2026-10-16  
**Область:** `internal/content/data/senior-13..24.{json,md}`, соответствующие `internal/grading/specs/senior-{13..24}-{easy,medium,hard}.json` и `internal/content/solutions/senior-{13..24}-{easy,medium,hard}/main.go`. Read-only review: candidate code и runner не запускались, исходники/учебные данные не менялись. Это педагогический и статический контрактный аудит, не проверка качества всей Senior-программы.

## Вывод

Материалы хорошо обозначают границы доказательств: P5 называется учебным системным дизайном, helpers не доказывают пригодность production-системы, а human review ограничен конкретной версией и scope и не удостоверяет профессию. Однако текущие Easy/Medium/Hard упражнения в основном проверяют маленькие булевы/строковые классификаторы. Ни один из 36 заданных артефактов (ADR, design doc, миграционный план, postmortem/retrospective, review package) не принимается или не оценивается grader-ом. Поэтому прохождение этих заданий не является доказательством навыка архитектурного проектирования и не должно быть достаточным основанием для Senior rank.

## Находки

### P1 — задания и grader не проверяют центральный P5 результат

**Файлы:** все `internal/content/data/senior-13..24.json` и `.md`; контракты в `internal/grading/specs/senior-13..24-{easy,medium,hard}.json`.

Каждое упражнение требует «отдельную заметку/артефакт», часто с конкретными полями, но критерии повторяют одни и те же четыре пункта: точная функция, decision table, артефакт с источником/владельцем/границами, ограниченный вывод. И JSON prompts, и Markdown явно уточняют, что артефакт grader-ом не проверяется. Grading specs передают только примитивные поля в helper и содержат по одному public и одному private vector. Например, `senior-13-hard` принимает три bool и проверяет их conjunction; он не получает requirements, risks или rollback plan. `senior-23-hard` принимает `openRisks` и целое `reviewerSigned`; он не получает подпись, reviewer identity, версию, scope или сам review.

**Воспроизведение по контракту:** студент может пройти все 36 упражнений, реализовав 36 классификаторов и не представив ни одного ADR/design doc, диаграммы, migration walkthrough, воспроизводимого evidence bundle или проведённой защиты. Это следует из интерфейсов grading specs и явного текста «отдельный артефакт не проверяется grader-ом»; выполнение тестов не нужно для вывода.

**Минимальное исправление:** сохранить helpers только как локальную практику, но отделить их от rank evidence. Добавить один сквозной P5 capstone с версионированным заданием и обязательным собранным пакетом артефактов: problem/context, alternatives + ADR, контракты/данные, failure + migration/rollback, наблюдаемость/privacy, воспроизводимые эксперименты, риски/неизвестные и retrospective. Автоматически проверять структуру/наличие/согласованность там, где это детерминировано; содержательное качество оставить явно отдельному human review с конкретными rubric, reviewer consent, scope/version и outcome. Не выдавать Senior award до прохождения всех требуемых компонентов и review gate; подпись reviewer сама по себе не должна означать профессиональную квалификацию.

### P2 — непропорционально узкая проверка логики и слабая защита от неверных реализаций

**Файлы:** все 36 соответствующих grading specs; особенно ветвистые контракты в `senior-14`, `senior-16`, `senior-17`, `senior-18`, `senior-20`, `senior-21`, `senior-22`, `senior-23`, `senior-24`.

У каждого контракта ровно два вектора — public и private. Для функций с несколькими ветвями это оставляет большую часть decision table непроверенной: `CapacityHeadroom` имеет invalid/over-limit/low-headroom/headroom; `MigrationPhase` — четыре состояния; `ReleaseGate` — четыре приоритетные ветви; `ActionPriority` — invalid/now/schedule/track; `DefenseStatus` — invalid/not-reviewed/reviewed-with-risks/reviewed-no-risks. Большинство specs проверяют только положительный основной пример и один переход; равенство порогам, конфликтные flags, отрицательные значения/границы и приоритет ветвей часто отсутствуют. Ветка всё ещё может быть неверной при PASS указанных vectors.

**Минимальное исправление:** хранить в spec таблицу случаев по каждой ветви и приоритетному сочетанию; для числовых контрактов включить граничные и overflow cases; для каждого контракта добавить mutation-oriented контроль (например, инверсия сравнения, перестановка ветвей, пропуск одного флага) и убедиться, что suite его ловит. Эти тесты повысят уверенность только в helper-контрактах, не в P5 артефактах.

### P2 — риск ошибочно прочесть учебные эвристики как архитектурные пороги

**Файлы:** `internal/content/data/senior-16.json` (`CapacityHeadroom`), `senior-20.json` (`CanExpand`), `senior-21.json` (`CapacityBand`).

Фиксированные пороги `80%` и `sampleCount >= 100` являются условием искусственных классификаторов, а не статистически выведенным SLO, capacity margin или универсальным canary minimum. `senior-20` прямо предупреждает, что 100 — синтетический порог и не достаточное основание для расширения; `senior-16/21` также оговаривают, что helpers не прогнозируют production и применимы к одинаковым единицам/workload. Эти caveats хороши, но короткие vectors и ярлык `low_headroom`/`near_tested_limit` могут закрепить эвристику без расчёта чувствительности.

**Минимальное исправление:** оставить числа только как явно маркированные toy inputs, добавить случаи ровно на границе и рядом с ней, и в артефакте потребовать обосновать настоящий порог через измерительный сценарий, окно, denominator, uncertainty и владельца решения. Не переносить helper output в release/scale gate.

### P2 — `DecisionReview` смешивает наличие нового evidence с его проверенностью

**Файлы:** `internal/content/data/senior-24.json` (`senior-24-medium`); повтор контракта в `senior-24.md`; `internal/grading/specs/senior-24-medium.json`.

Prompt формулирует ветку как «trigger с новым evidence не проверенным → gather_evidence», но signature содержит только `triggerObserved`, `evidenceUpdated`, `ownerAssigned`. Boolean `evidenceUpdated` не может сказать, существует ли новый evidence, проверено ли оно, или только обновлён сам документ. При `true` функция идёт к `assign_owner`/`revisit_decision`; при `false` — `gather_evidence`. Имена входа и ветви не полностью согласованы с тем, что предлагается классифицировать. Текущие public/private vectors покрывают только `(true,true,true)` и `(true,false,true)`.

**Минимальное исправление:** уточнить наблюдаемый state-machine contract. Например, передавать отдельные `newEvidenceAvailable` и `evidenceVerified` (и назвать, что именно проверяется), либо переименовать состояния/функцию так, чтобы один флаг однозначно обозначал переход. Добавить вектор на каждую комбинацию с явным приоритетом и согласовать JSON, Markdown, spec и reference.

## Человеческая проверка и доверительная граница

Уроки 13 и 23 корректно говорят, что reviewer оценивает конкретную версию в ограниченном scope, подпись не является аккредитацией/стажем, а согласие нужно получить явно. Тексты 13 и 23 также требуют альтернативы, остаточные риски, ограничения, воспроизводимое свидетельство и несогласие. Это сильная концептуальная рамка.

Но сейчас это только учебные инструкции и непроверяемые заметки. `reviewerSigned int` в `senior-23-hard` — симуляционный вход, не криптографическое доказательство и не реальный review event; само условие отдельно запрещает трактовать статус как credential. В будущей rank-системе нужно хранить фактический artifact/version/scope, review outcome и согласие отдельно от кода helper, а автоматический экзамен или LLM не должен имитировать человеческое approval. Любой локальный сертификат должен явно ограничиваться критериями Academy.

## Remediation addendum — Senior-24 Medium DecisionReview — 2026-10-16

The ambiguous `evidenceUpdated` input was replaced with separate `newEvidenceAvailable` and `evidenceVerified` inputs in `internal/content/data/senior-24.json`, the Markdown lesson, `internal/grading/specs/senior-24-medium.json`, and its trusted reference. The ordered contract now defines all 16 combinations of `triggerObserved`, `newEvidenceAvailable`, `evidenceVerified`, and `ownerAssigned`; the Markdown gives a Russian condition/output for each tuple. Eight public and eight private vectors cover each tuple exactly once, including `verify_evidence` as a separate branch.

Verification: `docs/validation/m11-senior24-decisionreview-20261016.txt` records the actual production grading path through `Spec.Request`, pinned rootless runner and `Spec.Check`: 16/16 vectors PASS. The subsequent `docs/validation/m11-senior24-vector-remediation-20261016.txt` expands Easy `ActionPriority` to 16 boundary/priority tuples (16/16 PASS) and Hard `RetrospectiveComplete` to the full eight bool states (8/8 PASS). `go test ./internal/content ./internal/grading -count=1` and `go vet ./internal/content ./internal/grading` PASS after both changes. These close the specific Senior-24 helper/vector findings only. Sparse-vector gaps remain in other Senior lesson helpers; all helpers remain separate from P5 artifact review. The subsequent exact-current-tree M11 aggregate passed: 530 subtests, 2,985.278s; see `docs/validation/m11-current-tree-acceptance-20261016.txt`. None of these helpers or vectors checks P5 architecture artifacts or reviewer independence.

## Ограничения аудита

- Кандидатский Go-код, sandbox и grading runner намеренно не запускались.
- Coverage оценена по public/private vectors в specs и контрактах заданий; это не mutation-test run и не количественная оценка покрытия кода.
- Аудит ограничен Senior-13…24 и не устанавливает готовность предыдущих Senior глав, P5 project integration, полного rank policy или внешней credential trust.
- Наличие подробной теории и checklist в Markdown подтверждено просмотром этих файлов; качество реального студенческого артефакта не проверялось, поскольку текущий контракт его не принимает.
