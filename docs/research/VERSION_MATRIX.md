# Version matrix

verified_at: 2026-10-15. Выбранные версии фиксируются lockfiles; latest не используется при сборке.

| Component | Selected | Source | Why |
|---|---|---|---|
| Go | 1.27.2 | https://go.dev/dl/?mode=json | актуальный stable patch |
| Node | 24.18.0 (installed) | https://nodejs.org/en/blog/release/v24.18.0 | LTS, подходит engines |
| SQLite driver | v1.60.1 | https://proxy.golang.org/modernc.org/sqlite/@latest | pure Go local persistence |
| react | 19.3.1 | https://registry.npmjs.org/react/latest | совместимый pinned candidate |
| react-dom | 19.3.1 | https://registry.npmjs.org/react-dom/latest | совместимый pinned candidate |
| typescript | 7.0.2 | https://registry.npmjs.org/typescript/latest | совместимый pinned candidate |
| vite | 8.3.4 | https://registry.npmjs.org/vite/latest | совместимый pinned candidate |
| @vitejs/plugin-react | 6.1.2 | https://registry.npmjs.org/@vitejs/plugin-react/latest | совместимый pinned candidate |
| @types/react | 19.3.0 | https://registry.npmjs.org/@types/react/latest | совместимый pinned candidate |
| @types/react-dom | 19.3.0 | https://registry.npmjs.org/@types/react-dom/latest | совместимый pinned candidate |
| codemirror | 6.0.2 | https://registry.npmjs.org/codemirror/latest | совместимый pinned candidate |
| @codemirror/lang-go | 6.0.1 | https://registry.npmjs.org/@codemirror/lang-go/latest | совместимый pinned candidate |
| react-markdown | 10.1.0 | https://registry.npmjs.org/react-markdown/latest | совместимый pinned candidate |
| @playwright/test | 1.64.1 | https://registry.npmjs.org/@playwright/test/latest | совместимый pinned candidate |
| @axe-core/playwright | 4.13.0 | https://registry.npmjs.org/@axe-core/playwright/latest | совместимый pinned candidate |
| vitest | 5.0.4 | https://registry.npmjs.org/vitest/latest | совместимый pinned candidate |
| @types/node | 26.6.5 | https://registry.npmjs.org/@types/node/latest | совместимый pinned candidate |

Актуальность проверена сетью; совместимость сборки проверяется M1. Vite 8.3.4 уточняет исследовательский ориентир 8.1.x. CodeMirror пакеты имеют независимые semver; version 6.0.2 для codemirror, lang-go 6.0.1.

## Инструменты проверки и уточнения M1–M6

- @codemirror/language6.13.1, @lezer/highlight1.2.5 — direct dependencies, реально установлены/собраны.
- golang.org/x/vuln v1.9.0 — https://proxy.golang.org/golang.org/x/vuln/@latest (2026-10-11), govulncheck выполнен; vulnerabilities не найдены.
- GitHub Actions official latest releases fetched2026-10-15: checkout v7.0.1, setup-go v7.0.0, setup-node v7.1.0, upload-artifact v7.0.2; git ls-remote tag commit pinned в workflow, remote CI не выполнялся.
- Rootless Podman5.8.7, bubblewrap0.12.0, Docker29.9.0 установлены; только Podman используется. Image digest runner константа в internal/runner; Go внутри фактически1.27.2.

M9 additions verified2026-10-15: `github.com/gpdf-dev/gpdf v1.0.13` (upstream tag336a66e7f7ffe4a1557570e53f2a6384815ac02e), `golang.org/x/image v0.47.0` (officialproxy/pkgdocs), transitive x/text0.43.0/x/sys0.49.0 pinnedbygo.sum. Dev-only independent QR decode `jsqr1.4.0` and `pngjs7.0.0`, currentnpmregistry and primaryGitHub README checked. Research/Unicodefontprovenance: CREDENTIAL_IMPLEMENTATION.md. Actualgo+frontendtests,renderandQRdecodepassed; govulncheck/npm audits0 atthischeckpoint.
