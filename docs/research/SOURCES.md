# Источники исследования

Дата проверки 2026-10-15. URL, результаты и различия tested/inferred находятся в тематических файлах:

- [Go](GO_CURRENT_STATE.md): официальные download API, release notes, pkg.go.dev.
- [Backend](ECOSYSTEM.md): module proxy metadata и первичные документы проектов.
- [Frontend](FRONTEND.md): официальные docs/releases/npm registry; [raw versions](evidence/npm-versions.json).
- [Codex/AI](AI_CODEX.md): официальный schema, модели, subagents, pricing; снимки evidence/.
- [Sandbox](SANDBOX_SECURITY.md), [threat model](../architecture/THREAT_MODEL.md): runtime/kernel границы, фактические local probes.
- [Credentials](CREDENTIALS.md): 1EdTech и W3C; реализация compliance не заявляется.
- [Version matrix](VERSION_MATRIX.md) и [ADR](../architecture/ADR-0001-tech-stack.md): решения и pin policy.

Environment: Linux amd64; Go installed 1.26.9-X:nodwarf5, Node 24.18.0, npm 11.16.0, Codex 0.162.0-alpha.17.2. Папка исходно без .git; создан локальный Git, remote отсутствует, публикации не было. Docker rootful, Podman rootless, bubblewrap доступны. Runtime cgroup probe реально прошёл, но это не аудит отсутствия escape.
