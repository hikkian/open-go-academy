# История изменений

Проект пока не выпускал публичных версий. Этот файл фиксирует крупные реализованные части в репозитории; он не является release announcement или заявлением о завершении M13. Даты и границы сверены с `AGENT_HANDOFF.md` на 2026-10-18.

## Unreleased — 2026-10-18

### Добавлено

- Полные Middle/Middle+/Senior треки, P3–P5, immutable exam/rank workflows и human-review ограничения приняты в M11; all-88 gate v7, full make test и desktop/mobile E2E проходят.
- Выбранные владельцем лицензии: MIT для программного кода и CC BY 4.0 для оригинальных учебных материалов. Атрибуция указывает на Open Go Academy как проект, без заявления об индивидуальном авторе. Детали: `LICENSE`, `COURSE_LICENSE.md`, `NOTICE`.
- Финальный локальный Linux/rootless M12 audit покрывает API/runner границы, ресурсные лимиты, namespace/route probes, desktop/mobile E2E и accessibility. Мобильный overflow Middle lesson исправлен и закреплён регрессионной проверкой.
- M13 clean-source technical acceptance прошла на предыдущем source checkpoint; повторная чистая сборка и сканирование после добавления license files остаются следующими шагами.

### Ограничения

- M12 acceptance относится к локальной Linux/rootless Podman конфигурации; public multi-tenant Go execution остаётся отключённым. Прямой host PID-1/full mount, внешний DNS и literal `cgroup.events populated=0` не были подтверждены.
- M13 ещё не завершён: после licensing update нужен fresh-source bootstrap/test/build/E2E/Gitleaks; third-party legal/NOTICE text review и native non-Linux validation остаются открытыми. Публичной публикации не было.

## Unreleased — 2026-10-16

### Добавлено

- Локальная русскоязычная платформа с 40 опубликованными главами Zero, Junior и Junior+, редактором Go, проверкой и сохранением прогресса.
- P1 CLI Toolkit, P2 LinkHub и интеграция evidence workflow P3; проектные материалы P4 ScaleLab и P5 Architecture Capstone находятся в ходе интеграции и не означают готовность M11.
- Необязательный Tutor с local/cloud OpenAI-compatible настройкой, отдельным согласием на передачу контекста и ограничением бюджета.
- Локальные учебные credentials и приватный паспорт с оговорённой моделью доверия.
- Исследования версий, модель угроз, validation evidence и панель прогресса.

### Ограничения

- M11 остаётся незавершённым: 88 старших глав — draft; автоматическое принятие не заменяет педагогическое ревью, реальные advanced exam policies и независимую человеческую оценку P5.
- Публичное multi-tenant выполнение Go-кода отключено и не подтверждено.
- M12 audit и M13 release acceptance не завершены. Публикации не было.
- Снимок на 2026-10-16 не содержал project license. В последующем checkpoint владелец выбрал MIT для кода и CC BY 4.0 для курса; M13 clean-source gate нужно повторить после добавления этих файлов.
