# Архитектурный обзор

Состояние на 2026-10-16. Это схема текущей local-first реализации, а не целевой hosted deployment. Ссылки на конкретные модули помогают находить фактическое поведение; подробности выбора — в [ADR-0001](ADR-0001-tech-stack.md), граница проектного исполнения — в [ADR-0002](ADR-0002-project-execution.md).

## Компоненты

| Компонент | Реализация | Ответственность и границы |
|---|---|---|
| Web UI | `web/src/`, React/Vite/TypeScript, CodeMirror | Уроки, редактор, прогресс, проекты, экзамены и Tutor. Браузер не авторитетен для рангов/evidence. |
| API | `cmd/api`, `internal/httpapi` | Loopback HTTP, сессии/origin checks, каталог, grading orchestration, проекты, credentials и Tutor configuration. |
| Persistence | `internal/store`, `internal/store/migrations` | SQLite через `database/sql`; атомарные события, попытки, evidence и локальные настройки. Один локальный профиль/владелец установки. |
| Course/grading | `internal/content`, `internal/grading`, `internal/assessment`, `internal/learning` | Встроенный контент, deterministic checks, навыки и eligibility. Draft-контент не считается опубликованным. |
| Basic runner | `cmd/runner`, `internal/runner` | Отдельный Unix-socket worker для учебного кода; Podman OCI, pinned image, bounded resources и cleanup. |
| Project runner | `cmd/project-runner`, `internal/projectrunner`, `internal/projects` | Отдельные project/checker profiles и неизменяемые inputs; browser не задаёт image, path или policy. Реальный support зависит от профиля/runtime readiness. |
| Tutor | `internal/tutor`, API Tutor routes | Необязательный gateway. Только выбранный контекст и явное согласие; model output не является оценкой/решением о ранге. |
| Credentials | `internal/credentials`, `internal/credentialrender` | Локальная подписанная образовательная запись с приватностью/отзывом; самоустановка не является внешним доверенным issuer. |

## Поток урока и запуска

```text
Browser → loopback API → validate session/origin/quota → runner Unix socket
       → ephemeral rootless OCI job → bounded result → deterministic grader → SQLite
```

API не запускает student source через host shell. Runner failure закрывает действие вместо unsafe fallback. Для проектного Submit доказательство содержит immutable исходный артефакт и результат pinned checker; автоматический PASS не закрывает субъективный rubric и не выдаёт advanced rank.

## Данные и доверие

`.data/academy.db`, session secret, issuer key и provider configuration локальны и исключены из Git. Источники данных пользователя, содержимое урока, runner output и LLM response имеют разные уровни доверия. Серверная транзакция и immutable evaluation evidence — единственный источник для rank/credential rules. Для P5 требуется независимый человеческий review; модель, submitter и автоматический checker не могут его подменить.

## Ограничения

- Поддержанный и проверенный режим — Linux single-user local install.
- OCI namespaces и rootless Podman всё равно разделяют host kernel; это не security boundary для публичных hostile workloads.
- Hosted multi-user deployment, внешняя identity, tenant isolation, hosted PostgreSQL, Windows/macOS и trusted public issuer не считаются реализованными.
- Полные M12/M13 audit/release gates и advanced-rank policies ещё не закрыты. См. [traceability](../REQUIREMENT_TRACEABILITY.md) и [handoff](../../AGENT_HANDOFF.md).
