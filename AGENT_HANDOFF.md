# Open Go Academy — точка продолжения

Обновлено **2026-10-18 04:03 Asia/Atyrau**. Репозиторий локальный, ветка `master`, remote нет. Последний завершённый checkpoint до текущих изменений: `060d6fa` (ранее `c480648`, `b4d7304`). **M0–M10 DONE; M11, M12, M13 IN PROGRESS.** Не объявлять проект законченным.

## Состояние продукта

- 40 опубликованных глав Zero/Junior/Junior+, 120 заданий; 88 Middle/Middle+/Senior — черновики, **не публиковать автоматически**.
- Рабочие урок/editor/Go runner/grading/drafts/progress, проекты P1–P5, шесть objective exam catalogs, local credentials, optional AI Tutor.
- Advanced policies связывают course/prior-rank, точный P3/P4/P5 machine artifact и manual rubric/reviewer quorum (1/1/2 ключа). Без operator registry и совпадающего runner profile выдача закрыта. Политика/отзывы закреплены на Start и повторно проверяются Claim/Finish; award атомарный. `c480648` и `docs/validation/advanced-rank-pipeline-progress-20261017.txt`.
- Подпись reviewer доказывает владение разрешённым ключом, не личность/независимость. Credentials локальные self-issued, не внешняя аттестация. Public execution не разрешён.
- Tutor: локальный loopback OpenAI-compatible endpoint или облачный HTTPS со своим API key; курс и проверки доступны без AI. Retrieval только опубликованного курса; реальных платных вызовов не делали.
- Панель прогресса: `docs/PROJECT_PROGRESS.html`, исходник `docs/project-progress.json`. 11/14 gates (79%) — невзвешенное число этапов, не доля работы. 88/88 теперь явно означает подготовленные черновики, а не завершённую приёмку.

## Текущие изменения 2026-10-18

- M16–18 Hard: реальные fake/callback протоколы effect→ack, replay без повторного эффекта, commit→publish→mark, unknown commit→reconcile. Grader наблюдает вызовы/ID; 9 содержательных мутантов отклоняются. Последний scoped gate PASS 100.139s; reviewer findings закрыты.
- M20 Hard: все 8 сочетаний mandatory/confirm/returned; противоречие returned без mandatory — invalid. M22 Hard учитывает прошедшее пополнение, capacity и безопасный ceil без int64 overflow. 5000 `testing/quick` случаев с независимым `math/big` oracle, invalid dimensions, MaxInt64. Production regression отклоняет 5 мутантов; scoped M20/M22 PASS 66.887s, race/vet PASS; strong reviewer перепроверил.
- M26: validation до Begin, Apply/Commit/Rollback, неизвестный commit outcome; Medium driver независимо проверяет trace, Hard driver добавляет trace сам. References и 3 ошибочных решения проверяются production Request/Check. Scoped gate и регрессии PASS; reviewer findings закрыты.
- Все 88 M11 metadata получили канонические topical practice skills. Новый `internal/competency` — общий список 13 ID; loader отклоняет неизвестные/дубли/пустой mapping function. Helpers не дают communication; P5 automatic больше не даёт communication. Learning policy `learning-evidence-0.3`, ранги не менялись. Целевые content/learning tests и independent review PASS.
- **Новые находки после старта общего gate:** M28 DI не проверял обещанный constructor; Senior13–23 sparse vectors; M+21/24 шаблонные неверные абзацы. Их исправляют ограниченные workers (см. ниже). Старое blanket замечание «нет project evidence M27–40» неверно относительно c480648; реальный пробел — явная traceability rubric.
- P3 manual rubric теперь **v2**: обязательные M27–40 критерии DI graph, access, telemetry, SLI/SLO/error budget, fault/recovery, release/rollback evidence. API GET project отдаёт manual_review_rubric до отправки проекта; automatic project criteria/version не менялись. Старые v1 review не подходят новой policy. Root targeted API tests PASS; strong review структуры PASS. UI worker добавляет видимые критерии.
- Новые `scripts/m11_source_fingerprint.py`, `check_m11_acceptance.py`: полный PASS принимается только при scope all88 и одинаковом source fingerprint до/после; mtime не используется. 5 Python tests PASS. Старые отчёты без fingerprint остаются историческими.
- `docs/COURSE_AUTHORING.md`, GitHub issue/PR templates добавлены; Python gate tests включены в Makefile/CI, progress axe check в CI. Лицензия владельцем не выбрана.

## Проверки и процессы текущей сессии

- `GOFLAGS=-p=1 GOMAXPROCS=2 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/oga-m26-20261018 make test` — **PASS** для интегрированного среза до последних M28/Senior/P3 UI изменений: Go unit, P1 refs, vet, race, TS/build. Raw: `artifacts/m11-integrated-tech-20261018.log`. Повторить затронутые/общие checks после окончания workers.
- Полный88 gate был намеренно остановлен через SIGINT после новых review findings, exit130; **не PASS и не регрессия кода**. Отчёт `docs/validation/m11-superseded-acceptance-20261018.txt`, лог `artifacts/m11-full-acceptance-20261018.log`. После окончательных правок запустить заново:

```sh
python3 scripts/check_m11_acceptance.py --report docs/validation/m11-current-tree-acceptance-20261018.txt --log artifacts/m11-full-final-20261018.log --cache-dir /tmp/oga-m26-20261018
```

- Full browser **50 tests запущены, результат ещё проверить**: exec session `94758`, лог `artifacts/m12-full-browser-20261018.log`. API disposable `/tmp/oga-e2e-20261018/api/academy.db`, port8081. Flags RUNNER/PROJECT_RUNNER/P3_RUNNER=1, отдельные sockets. Function runner PTY session `70597`; project runner `45536`. После browser suite остановить только эти процессы Ctrl-C, пользовательские сервисы не трогать. P3/P4/P5 native cases уже проходили в текущем логе.
- `python3 scripts/local_load_check.py --report artifacts/m12-local-load-20261018.json` — **PASS**: 120/120 reads200, 119 draft PUT200 +21 quota429/Retry-After, целый accepted bundle, restart persistence, baseline profile, Zero/verified=false/no awards/XP0, SQLite integrity/FK PASS. Собственный temp API/keys/DB удалены. Это ограниченный local load test, не SLO или public certification.
- `python3 -m unittest discover -s scripts -p 'test_m11_source_fingerprint.py'` — 5PASS. `npm run test:progress` — desktop/light + mobile/dark axe clean/no overflow; screenshot visually inspected, теперь `artifacts/project-progress-*-latest.png`.
- Исторические полноценные runner/fresh-source проверки: `docs/validation/m13-clean-source-acceptance-20261016.txt`, `p3-*`, `p4-p5-*`, `m11-current-tree-acceptance-20261016.txt` (530 subtests, 2985.278s). Последний88 aggregate устарел после правок M24/M25 и сегодняшних исправлений.

## Незавершённые workers / точный следующий шаг

1. Дождаться `repair_middle28_composition`, `close_senior13_23_vectors`, `p3_manual_rubric_ui`. Они владеют непересекающимися source paths. Не потерять их изменения. Проверить сообщения/статус, targeted tests, независимый review оставшихся контрактов.
2. UI worker **не запускать build/Playwright до завершения текущего50-test suite** (общий port8081 и web/dist). Потом typecheck/build и новые desktop/mobile rubric tests.
3. После всех исходных правок: полный `make test`, новый88 gate с fingerprint; обновить report path в progress JSON, render/test dashboard, handoff/plan, локальный Git checkpoint. Не отмечать M11 DONE лишь за aggregate.
4. M11: окончательная педагогическая приёмка, skill/course/project/rank traceability и публикационный gate всех88. Сверки: `docs/research/M11_*STATUS_20261018.md`, `M11_SKILL_MAPPING_20261018.md`; source-review не равен запуску или полной сертификации.
5. M12: завершить полный security/threat, load/failure, responsive/theme/a11y audit и закрыть findings. M13: final clean-source install/recovery, SPDX/NOTICE/secret review, traceability; **владелец должен выбрать LICENSE перед release readiness**, публикации не разрешены.

## Лимиты, продолжение и безопасность

Последний check **2026-10-18 04:02 Asia/Atyrau**: ordinaryUsageAllowed=true, primary99%, weekly16%; reset credit не использован. Primary reset 2026-10-18 08:38:23, weekly 2026-10-25 03:38:23. Единственный heartbeat `open-go-academy-m13` обновлён через tool: **2026-10-18 08:39:23 Asia/Atyrau**, COUNT=1, ACTIVE; не создавать дубликаты/ежедневный repeat. При исчерпании сохранять checkpoint и переносить тот же heartbeat на ограничивающий reset+1мин. Новые узкие agents использовать `fork_turns=none` для экономии; builder/researcher gpt-6-luna, strong reviewer gpt-6.1-sol, max3children.

Работать в репозитории. `.data/academy.db`, `session.secret`, `issuer.key` — пользовательские данные/ключи: не удалять, не логировать, не коммитить. Сервис8080 и основной8380 не трогать. Не публиковать, не тратить деньги, не включать публичный multi-tenant runner. Все QA базы/ключи — disposable. Чат не архивировать.

## Обновление перед checkpoint (04:06 Asia/Atyrau)

Все source workers завершили редактирование. M28 typed constructor/service/fake hard и mutant regression PASS, scoped40.409s. Senior13–23 все33specs расширены, bool conjunction truth tables,4mutants/all33refs PASS; общий scoped gate ещё выполняется (агент close_senior13_23_vectors), последний transient failure был от несинхронного M28 до его завершения. M+21/24 предметные абзацы исправлены. P3 pre-submit manual rubric UI готов, typecheck PASS, build/new browser test ещё не запускались до освобождения8081. Новые файлы tests/project-manual-rubric.spec.ts и project_review_outcomes_test.go.

**Новый финальный88 gate действительно запущен:** exec session30117, `artifacts/m11-full-final-20261018.log`, report `docs/validation/m11-current-tree-acceptance-20261018.txt` появится после завершения. Source fingerprint `sha256:4108ef753f3214cf7c4df2ea0b288a9ef4b377ca642be48121d14a1b78f0f4fe`. Не менять content/grading/runner/competency или fingerprint/check scripts во время этого прогона. После него сверить PASS/FAIL/STALE и source hash; report/path уже настроен в progress JSON.

Full50E2E session94758 пока работает: desktop P1 PASS4.1m, P3/P4/P5 runtime PASS, далее mobile. После окончания выполнить новый UI rubric spec (2projects), web typecheck/build; остановить PTY70597/45536. Если лимит прекратит turn до этого, heartbeat08:39 должен прочитать логи/процессы, а не считать проверки завершёнными. Все API/runner процессы принадлежат только disposable QA сессии.

M12 load итог скопирован в tracked `docs/validation/m12-local-load-20261018.json`; все проверкиPASS, baseline/no awards/XP и cleanup подтверждены. Общий make test PASS до последних M28/Senior/P3UI изменений; новый final make test ещё нужен. Checkpoint сохраняет реализацию, не завершённость M11/M12/M13.
