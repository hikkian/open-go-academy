# Frontend: проверка стека и выбор

**Срез исследования: 2026-10-15.** Цель — выбрать простой клиент для учебной платформы с русскими уроками, маршрутизацией, адаптивным lesson workspace и редактором Go. Проверены первичные документы проектов и доступные npm registry `latest` endpoints. Релизы после указанной даты не учитываются; `latest` меняется со временем и должен фиксироваться lockfile.

## Решение

Для первого рабочего вертикального сценария рекомендую **React + TypeScript + Vite + CodeMirror 6 + `react-markdown` + Playwright**. Это обычный браузерный SPA: платформа в ТЗ имеет отдельный Go API, а описанный dashboard/урок/редактор — интерактивное приложение. Next.js добавил бы серверный runtime и правила server/client components, которые сейчас не требуются; вернуться к нему стоит при подтверждённой потребности в публичном SEO-контенте или SSR. На дату среза Next.js 16.4.1 доступен, минимум Node 20.9+, TypeScript 5+, но это не преимущество для текущего авторизованного учебного workspace.

Рекомендуемый стартовый набор зависимостей с фиксацией точных версий при реализации: `react`/`react-dom` 19.3.x, `typescript` 7.0.x после проверки совместимости `@types/react` и всех инструментов, Vite 8.1.x + соответствующий `@vitejs/plugin-react` 6.x, CodeMirror 6 (`codemirror`, `@codemirror/*`) с Go language package, `react-markdown` 10.x, `@playwright/test` 1.64.x. Не фиксирую патч для пакетов, у которых npm endpoint не удалось независимо открыть. Установка и lockfile ещё не проверялись.

**Node 24.18.0 совместимость: по заявленным диапазонам — да.** Vite 8 требует Node 20.19+ или 22.12+; Next.js 16 требует 20.9+; Playwright 1.64.1 публикует `engines.node >=20`; Vitest 5 требует Node >=22.12. Node 24.18.0 удовлетворяет этим минимумам. React, TypeScript и браузерные редакторы выполняются преимущественно в браузере; их Node engine metadata сама по себе не заменяет проверку всей toolchain. Точная сборка проекта на Node 24.18.0 **не запускалась**.

## Сравнение решений и trade-offs

| Область | Выбор | Официально подтверждённые версии/даты | Альтернатива и цена |
|---|---|---|---|
| UI | React 19.3.x | React 19.3 объявлен 2026-09-09; стабильные View Transitions и Fragment Refs. npm `react/latest` на срезе отвечает 19.3.1. | Next.js 16.4.1 даёт SSR, routing/caching conventions, но добавляет серверную архитектуру; React без framework оставляет routing/data fetching приложению. |
| Типы | TypeScript 7.0.x, с проверкой библиотек | Официальный release-notes индекс уже указывает 7.0; npm `typescript/latest` отвечает 7.0.2 (Node >=16.20). TypeScript 6.0 — переходный выпуск с deprecations, предшествующий 7.0. | 5.9.x — более консервативный baseline, но старее. Для раннего проекта выбор 7.0 разумен только после проверки ecosystem type compatibility и `tsc --noEmit`; иначе временно pinned 5.9.x и обновление отдельным изменением. |
| Dev/build | Vite 8.1.x + `@vitejs/plugin-react` 6.x | Vite 8 stable — 2026-03-12, Rolldown заменил пару esbuild/Rollup; 8.1 — 2026-06-23. Node >=20.19 или >=22.12. Официально заявлена совместимость plugin API и upgrade path, но отмечен прирост установки Vite примерно на 15 MB относительно Vite 7. | Vite 7 зрелее, Vitest 3.2+; его Node >=20.19/22.12. Vite 8 — актуальная поддерживаемая линия, но новый bundler заслуживает реальной проверки плагинов и билда. Next.js нужен при SSR/SEO, сейчас это лишняя серверная поверхность. |
| Редактор | CodeMirror 6 | Модульная система пакетов; подключаемые gutter/undo/highlight и язык. Go package `@codemirror/lang-go` имеет `go()` с Lezer parser, folding, indentation и локальным completion. | Старый GitHub repo архивирован 2026-04-15 и указывает новый upstream `code.haverbeke.berlin/codemirror/lang-go`; опубликованный пакет/версия остались в npm-экосистеме, но governance/source переехали. Перед pin проверить npm metadata, обновления и лицензию. Monaco — более полная IDE-поверхность, привычна по VS Code и хороша для нескольких файлов, но крупнее, требует интеграции worker/assets и сама не даёт Go `gopls`/backend compiler. |
| Уроки Markdown | `react-markdown` 10.x, raw HTML выключен по умолчанию | Проект заявляет React Markdown renderer, 10.1.0 — последняя видимая upstream release (2026-03-07); использует unified/remark. README отмечает CommonMark; GFM — отдельный `remark-gfm`. | MDX даёт React components внутри контента, но превращает материалы в исполняемый/собираемый авторский код и повышает сложность контентной безопасности/сборки. Обычный Markdown лучше соответствует версионируемым урокам в репозитории. Удалённый пользовательский Markdown требует URL policy/sanitization; не подключать raw HTML без отдельной модели угроз. |
| Unit/component | Vitest 5.x — опционально при добавлении тестов | Vitest 5.0 — 2026-09-03; требуется Vite >=6.4 и Node >=22.12. Официально поддерживаются ветки 5.0 и 4.1; Browser Mode стал stable ещё в 4.0 (но требует provider package). | Для рекомендуемого набора Vite 8 Vitest 5 совместим по явному минимуму; всё равно проверить точную пару в lockfile. Не использовать одновременно Vitest Browser Mode и Playwright E2E для одних и тех же сценариев без явной причины. |
| E2E / браузеры | `@playwright/test` 1.64.x | npm latest endpoint: 1.64.1, Node >=20. Документация содержит браузерные проекты и E2E. | Cypress — альтернативный E2E framework с отдельной моделью запуска; Playwright даёт Chromium/Firefox/WebKit проекты в одном наборе и подходит для полного сценария «урок → редактирование → запуск → сохранение». |
| Accessibility | Playwright + `@axe-core/playwright`, плюс ручная клавиатурная проверка | Playwright docs прямо указывают, что axe ловит только часть типовых проблем; нужны ручные проверки. ТЗ требует семантику, порядок фокуса, контраст, reduced motion и keyboard-only flow. | Автоматический скан не подтверждает соответствие WCAG и не проверяет качество клавиатурного UX CodeMirror сам по себе. Нужны проверки label/role, видимого фокуса, zoom/контраста и редактора клавиатурой/скринридером. |

## Совместимость и ограничения

- Vite 8 поставляется как ESM-only и использует Rolldown/Oxc; минимальный Node engine соблюдён Node 24.18.0. Нечётные/непроверенные диапазоны Node официальная документация Vite исключает из CI, но Node 24 — чётная LTS-линия.
- TypeScript выполняет type-check отдельно: Vite transpile не заменяет `tsc --noEmit`. TS 7.0 — текущая официальная ветка на дату исследования, однако конкретную совместимость `@types/react`, плагинов и редакторных пакетов с TS7 здесь не удалось собрать и проверить. Без такой проверки фиксировать можно TS 5.9.x как более безопасный baseline либо принять TS7 после чистого install/typecheck.
- Проверка live npm metadata прошла для `react` 19.3.1, `typescript` 7.0.2, `next` 16.4.1 и `@playwright/test` 1.64.1. npm endpoint в используемом web-инструменте отказал для Vite, Vitest, Monaco, CodeMirror и `react-markdown`; для них версия/дата указана только там, где доступен upstream changelog/release. Не утверждаю, что проверил установку/сборку, а эти registry metadata не являются lockfile.
- Для редактора минимально необходимы Go подсветка, line numbers, tabs, multi-file state, форматирование через backend/runner contract, undo/redo, resize, autosave и отображение stdout/compiler diagnostics. Не обещать gopls autocomplete: ни CodeMirror, ни Monaco не предоставляют рабочий Go language server сами по себе.

## Источники (первичные)

Все страницы проверены 2026-10-15; последние патчи могут выходить позже.

1. React 19.3, 2026-09-09: https://react.dev/blog/2026/09/09/react-19-3 ; versions: https://react.dev/versions ; npm: https://registry.npmjs.org/react/latest
2. TypeScript release notes index: https://www.typescriptlang.org/docs/handbook/release-notes/overview.html ; TS 6.0 notes: https://www.typescriptlang.org/docs/handbook/release-notes/typescript-6-0.html ; npm: https://registry.npmjs.org/typescript/latest
3. Vite 8.0, 2026-03-12: https://vite.dev/blog/announcing-vite8.html ; Vite 8.1, 2026-06-23: https://main.vite.dev/blog/announcing-vite8-1 ; support policy: https://vite.dev/releases
4. Next.js 16.4 release/blog index: https://nextjs.org/blog ; npm metadata for 16.4.1: https://registry.npmjs.org/next/latest ; Node/TS requirements and breaking changes in 16 announcement: https://nextjs.org/blog/next-16
5. Node.js 24.18.0 (LTS), 2026-06-23: https://nodejs.org/en/blog/release/v24.18.0
6. CodeMirror reference manual: https://codemirror.net/docs/ref/ ; Go package README: https://github.com/codemirror/lang-go ; moved upstream: https://code.haverbeke.berlin/codemirror/lang-go
7. Monaco changelog: https://github.com/microsoft/monaco-editor/blob/main/CHANGELOG.md
8. `react-markdown` releases: https://github.com/remarkjs/react-markdown/releases ; docs/source: https://github.com/remarkjs/react-markdown ; GFM integration docs are referenced by upstream README.
9. Vitest 5, 2026-09-03: https://vitest.dev/blog/vitest-5 ; release policy: https://vitest.dev/releases
10. Playwright releases: https://playwright.dev/docs/release-notes ; npm: https://registry.npmjs.org/@playwright/test/latest ; accessibility guidance: https://playwright.dev/docs/accessibility-testing

## Что ещё не подтверждено

Не устанавливал зависимости, не запускал `npm view`, install, typecheck, lint, build, Playwright или accessibility scan; локального frontend manifest/lockfile в доступном списке файлов нет. До фиксации ADR нужно проверить точные npm версии Vite/Vitest/editor/Markdown пакетов, совместимость TS7 с типами зависимостей и React 19 peer ranges, затем сделать чистый install и build на Node 24.18.0. Для Go подсветки следует зафиксировать конкретный upstream commit/tag и npm artifact, потому что репозиторий пакета сменил хостинг.

## Уточнение registry (интегратор, 2026-10-15)
Прямой Python urllib registry fetch прошёл для всех выбранных пакетов. Снимок evidence/npm-versions.json: Vite 8.3.4, plugin-react 6.1.2, codemirror 6.0.2, lang-go 6.0.1, react-markdown 10.1.0. Это уточняет начальный ориентир 8.1.x; точные pins см. VERSION_MATRIX.
