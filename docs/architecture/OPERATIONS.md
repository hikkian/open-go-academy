# Локальная эксплуатация и восстановление

Документ описывает подтверждённую локальную эксплуатацию single-user. Он не является runbook для публичного хостинга.

## Старт и контроль готовности

Сначала соберите приложение (`make build`), затем запустите отдельные процессы в отдельных терминалах: `make runner`, затем `make dev`. API по умолчанию слушает `127.0.0.1:8380`; базовый runner сообщает `local runner ready`. Не меняйте bind address на публичный интерфейс. Отсутствие runtime/cgroup/seccomp или неверный pinned image должны оставлять Run/Submit закрытыми.

Проектный worker требуется отдельно. Для используемого проекта подготовьте immutable runtime профили по командам из README/текущего handoff и запустите `make project-runner`. Нельзя считать успешным запуск только потому, что API доступен: проверьте project capability endpoint/UI и фактическую Run/Test/Submit операцию в изолированном worker.

## Данные, backup, upgrade

Локальное состояние расположено под `.data/`. Чтобы сделать согласованный backup, остановите API, скопируйте `.data/academy.db` и относящиеся к ней session/issuer/provider secrets в закрытое хранилище. Не помещайте backup в репозиторий. Для восстановления используйте ту же БД и issuer key; потеря signing key не исправляется генерацией нового ключа для прежних credentials. Точный перечень файлов зависит от включённых функций.

Обновления схемы применяются миграциями при запуске API. Перед обновлением сохраните backup, затем проверьте health/основной пользовательский сценарий и регрессию. Не удаляйте базу ради прохождения миграции. Состояние и восстановление отдельных migrations — в `internal/store/migrations` и тестах `internal/store`.

## Офлайн-ревью проекта

Для сохранения ручного решения нужен отдельный reviewer с ключом, закреплённым владельцем установки. Ключ доказывает владение ключом из allowlist, но не устанавливает личность reviewer и не доказывает его независимость от автора. Один локальный пользователь не должен засчитывать свою подпись как независимое ревью. Эта функция пока не выдаёт Middle/Middle+/Senior и не снимает серверный fail-closed guard.

Создай новую пару ключ/registry; команда не перезаписывает существующие файлы. Храни raw private key вне публичных каталогов и backup репозитория:

```sh
sh scripts/go.sh run ./cmd/project-review-keygen \
  --reviewer-id external-reviewer-1 \
  --private-out /secure/path/reviewer.key \
  --registry-out .data/project-reviewers.json
```

Команда создаёт registry формата `{"reviewers":[{"reviewer_id":"…","public_key":"<base64>"}]}`. Для нескольких reviewers оператор вручную объединяет публичные записи без повторения ID. Registry должен быть обычным файлом без symlink и иметь mode `0600`. Запусти API с `ACADEMY_PROJECT_REVIEWERS_FILE=.data/project-reviewers.json`; при пустой настройке upload закрыт.

В Project workspace скачай packet для успешной пары submission/evaluation. Reviewer изучает приложенные файлы и rubric, заполняет `attestation_template` (reviewer ID, решение, комментарий и время), затем подписывает файл офлайн:

```sh
sh scripts/go.sh run ./cmd/project-review-sign \
  --key /secure/path/reviewer.key \
  --in unsigned-review.json \
  --out signed-review.json
```

Загрузи `signed-review.json` в workspace этой же версии проекта. Сервер проверит точные submission/evaluation/artifact/rubric hashes и операторский allowlist; решения append-only и видны из workspace после перезагрузки. Не выдавай утилиту за прокторинг или внешний реестр доверенных лиц. Ручное review сохраняется отдельно от machine evaluation и само по себе не влияет на XP или ранг.

## Диагностика

1. Проверьте точный commit, версии через `sh scripts/go.sh version` и `node --version`.
2. Проверьте API bind и runner readiness; не публикуйте socket/секреты/полные приватные логи.
3. При недоступности выполнения проверьте Podman rootless, cgroup v2/systemd, seccomp и наличие закреплённого образа. Не переключайте worker на host execution.
4. Запустите `make test`, а для actual sandbox probes — `make sandbox-test` на disposable машине.
5. Перед очисткой контейнеров убедитесь, что они принадлежат этой установке; recovery scripts могут воздействовать на контейнеры с собственными project labels.

## Остановка и инциденты

Останавливайте процессы через Ctrl+C и дайте cleanup завершиться. При подозрении на sandbox escape остановите runner, сохраните необходимые журналы с редактированием секретов и не продолжайте выполнять недоверенный код на этом host. Публичный incident response/rotation runbook ещё не подготовлен; публичный deployment не поддерживается.
