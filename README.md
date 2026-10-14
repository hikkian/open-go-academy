# Open Go Academy

Локальная браузерная платформа обучения Go на русском. Реализованы milestones M0–M6. Работает сквозной сценарий: профиль → самостоятельный урок Zero → редактор Go → изолированный запуск → проверка → сохранённый прогресс. Это начало платформы; полный курс Zero → Senior, проекты P2–P5, экзамены, AI gateway и сертификаты ещё не готовы. P1 Go CLI Toolkit поддерживает редактирование файлов, snapshots, реальный запуск/тесты/автооценку и ZIP; качество README и история Git требуют отдельного ревью. Точный статус: [план](docs/EXECUTION_PLAN.md), [handoff](AGENT_HANDOFF.md).

## Локальный запуск Linux

Требуются Node 24 LTS, npm, Go ≥1.21 (скрипт выбирает точный 1.27.2), make, **rootless Podman с cgroup v2/systemd, seccomp**. Подтверждённая среда: Fedora 44, Podman 5.8.7. Первоначальная установка скачивает зависимости и Go-образ; обучение не использует AI API и не требует ключа.

```sh
make bootstrap
make build
```

В двух терминалах из корня проекта:

```sh
make runner
```

```sh
make dev
```

Откройте http://127.0.0.1:8380 . Создайте профиль и откройте урок. Runner должен сообщить `local runner ready`. Если preflight не подтвердил ресурсы или runtime отсутствует, исполнение недоступно; API продолжает хранить профиль, уроки и черновики. Unsafe fallback отсутствует. Vite для разработки UI: `cd web && npm run dev` (API нужен на 8380).

Прогресс и профиль находятся в `.data/academy.db`; Unix socket и приватный session.secret — в `.data/`. Остановите процессы Ctrl+C. Для backup остановите API и скопируйте `.data/academy.db` вместе с `session.secret` в закрытый каталог. Не удаляйте БД при обновлении: migrations применяются автоматически. Локальная установка рассчитана на одного владельца ОС, cookie не является внешней проверкой личности.

## Автоматические проверки

```sh
make test
make sandbox-test
python3 scripts/crash-check.py
# С работающим make runner:
make e2e
```

`npm run e2e` без ACADEMY_E2E_RUNNER выполняет reader/editor smoke и явно пропускает запуск кода. Sandbox tests намеренно выполняют ограниченные memory/PID/output/disk/cancel probes внутри rootless контейнеров. Playwright включает desktop/mobile, light/dark, axe, screenshots и trace при ошибке. Артефакты в ignored `artifacts/`; evidence в [VALIDATION_REPORT](docs/VALIDATION_REPORT.md).

## Ограничения и сопровождение

API допускает только loopback. Runner — отдельный процесс с Unix socket 0600, image по digest, uid65532, read-only root, network none, capabilities drop, no-new-privileges, seccomp, cgroups и ограниченный tmpfs. Контейнер делит ядро с host; это локальный учебный режим, не проверенная защита публичного multi-tenant сервиса. [Модель угроз](docs/architecture/THREAT_MODEL.md).

Поддержка Windows/macOS и hosted PostgreSQL пока не проверена. Лицензия кода и учебных материалов ещё не выбрана владельцем; пакеты помечены UNLICENSED. Публикации и платной инфраструктуры нет. Исследование версий: [ADR](docs/architecture/ADR-0001-tech-stack.md) и [матрица](docs/research/VERSION_MATRIX.md). Обновления требуют регрессионной проверки, lockfiles сохраняются.
