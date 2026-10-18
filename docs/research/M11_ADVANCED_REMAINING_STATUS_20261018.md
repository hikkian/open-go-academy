# M11: сверка остаточных рисков Middle+ и Senior

**Обновлено:** 2026-10-18. Это ограниченная read-only сверка после предыдущих remediation; не полный педагогический review и не приёмка артефактов проекта.

## Что уже закрыто

- Middle+21 объясняет разницу между измеряемыми SLI/SLO и реальным admission control; текст не приписывает цели SLO управление запросами (`internal/content/data/middleplus-21.md`, раздел «Зачем нужна эта граница»).
- Middle+24 описывает воспроизводимый эксперимент и решение о выпуске, подчёркивает, что ScaleLab не принимает production-запросы (`internal/content/data/middleplus-24.md:22`). Предыдущая запись о шаблонной фразе в обоих уроках устарела.
- Senior13–23 helper specs больше не содержат лишь один private vector: текущие JSON имеют по 3–8 private cases плюс по одному public case. Проверены, например, все исходы `CapacityHeadroom` в `senior-16-medium.json` и все состояния `MigrationPhase` в `senior-17-easy.json`. Предыдущая запись об одном private case устарела. Полный текущий all-88 production grading gate исполняется; его точный результат ещё ожидается в `docs/validation/m11-current-tree-acceptance-20261018.txt`.
- Middle+13–23 overflow/mutation remediation и Senior13–24 scoped acceptance evidence остаются описаны в `docs/validation/m11-middleplus21-overflow-fix.txt`, `m11-middleplus13-17-remediation.txt`, `m11-middleplus18-20-22-23-remediation.txt`, `m11-senior13-24-quality.txt`. Эти logs — доказательства своих scoped snapshots; итоговый текущий gate отдельный.

## Последний vector gap закрыт (targeted gate pending full gate)

`middleplus-21-easy.json` теперь показывает public `SLOBand(100,0)→within` и `(100,1)→errors`; private cases закрывают `total=0`, negative total, negative failures и `failed>total`. Markdown example/public output синхронизированы. Scoped `middleplus-21` acceptance PASS записан в `docs/validation/m11-middleplus21-easy-current-20261018.txt`. Новый full fingerprint-bound all-88 gate v2 уже выполняется на hash из `AGENT_HANDOFF.md`.

## Границы автоматической оценки

Senior13–24 helpers остаются детерминированной практикой по decision tables. Они не принимают ADR, design doc, migration plan, benchmark или postmortem как реальные артефакты. Это честное ограничение; такие материалы должны оцениваться через P5 manual-review workflow. Сейчас Senior award связывает exact P5 submission/evaluation/artifact, policy snapshot и ручной quorum (2+ distinct allowed keys). Подпись подтверждает владение установленным ключом, но не реальную личность или независимость проверяющих. См. `docs/research/M11_SKILL_MAPPING_20261018.md`.

Содержательная вычитка всех 88 глав остаётся открыта: автоматические examples/references не подтверждают глубину объяснений, реалистичность workload, качество проектных артефактов или учебный исход сами по себе.
