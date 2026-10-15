# Независимое ревью M9: локальные credentials

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`. База — рабочее дерево M9 поверх `9e1b794`. Reviewer меняет только этот отчёт, реализацию не редактировал. Использованы ранее прочитанные audit-context-building/differential-review instructions, без подагентов. Публичная hosted аттестация и идентификация человека вне scope.

## Результат текущего прохода

High/Critical обхода выдачи или проверки подписи в прочитанном коде не найдено. Обнаружены три Medium дефекта поведения/приватности: URL выпуска через localhost, невозможность скрыть отозванный credential через UI и несовпадение DTO страницы публичной проверки. **Все три исправлены и CLOSED после повторной проверки 2026-10-15; открытых High/Medium в проверенной области нет.** Это ограниченный аудит, а не доказательство отсутствия ошибок.

## Контекст и границы доверия

Прочитаны `internal/credentials/**`, `internal/store/credentials.go` и tests/migration0006, `internal/httpapi/credentials.go` и новый HTTP test, `cmd/api/main.go`, API auth middleware, `web/src/Passport.tsx`, `internal/credentialrender/render.go` и tests. Спецификация G1–G4 и threat model прочитаны в предшествующем ревью этого же проекта.

POST issue принимает только attempt_id/name/consent; API получает completed exam attempt, текущие immutable exam criteria и сам подписывает payload. Store дополнительно проверяет существующий rank_award, passed100 и соответствие registered exam snapshot. Private key принадлежит локальной установке; verifier использует закреплённый ключ, envelope не выбирает trust root. Вывод PDF/PNG происходит только после проверки подписи в HTTP caller. Publish и passport visibility независимы и выключены по умолчанию. Отозванная опубликованная запись остаётся доступной для проверки статуса, пока владелец не снимает публикацию.

Blast radius: IssueCredential имеет один production caller (POST issue); Sign — тот же caller; Verify используется при private/public/check/export. PublicCredential вызывается только публичной проверкой. Новые файлы M9 не удаляют ранее существовавшие security guards. Mutations по-прежнему проходят global session+CSRF; только GET `/api/verify/{id}` и `/api/passport/public` освобождены от session.

## M9-R1 — Medium, CLOSED: выпуск не работал через разрешённый localhost Host

**Пути:** `internal/httpapi/credentials.go`, POST issue; `internal/credentials/credentials.go`, validVerifyURL; `internal/httpapi/api.go`, New.

Исходный issue формировал VerifyURL из r.Host, но crypto валидатор разрешает только числовые loopback IP. API при этом принимает localhost:port. На отдельной БД reviewer выполнил один и тот же валидный issue через httptest: localhost:8380→422,127.0.0.1:8380→201. Это мешало выпуску из разрешённого браузерного origin, не позволяя обойти подпись.

**Исправление:** API.New выбирает первый числовой loopback address из trusted hosts и хранит VerifyOrigin; Sign использует его независимо от входящего Host. Production ValidateAddress требует numeric loopback, main передаёт addr первым. Статически перепроверено; cmd/API tests после изменения PASS. Дополнительный временный httptest probe после fix реально выполнил localhost issue:201, verify_url с http://127.0.0.1:8380/verify/...; exit0. CLOSED.

## M9-R2 — Medium, CLOSED: отозванный опубликованный документ нельзя скрыть через UI

**Путь:** `web/src/Passport.tsx`, credential privacy checkbox/save button.

Publish→revoke сохраняет запись публичной специально, чтобы проверяющий увидел revoked. Но UI одновременно отключал published checkbox и save button при любом revoked_at. Поэтому владелец не мог убрать публичное имя/псевдоним после отзыва через продуктовый UI. Backend SetCredentialPublished(false) намеренно разрешён и должен оставаться доступным независимо от отзыва.

**Воспроизведение:** выпустить документ, опубликовать с согласием, отозвать. Checkbox публикации и «Сохранить видимость» disabled; публичная ссылка продолжает показывать subject. Отдельный HTTP probe reviewer подтвердил: API unpublish revoked→200, следующий unauth verify→404. Следовательно, ограничение именно frontend.

**Минимальная рекомендация:** разрешить published=false для revoked; запретить только повторную публикацию=true. Browser regression publish→revoke→unpublish→404.

**Повторная проверка:** checkbox отключён только для уже приватного revoked документа; у опубликованного revoked документа снятие checked разрешено. Save допускает revoked+original published+draft unpublished без нового consent. Backend продолжает разрешать unpublish и запрещать publish revoked; HTTP flow подтверждает404 после скрытия. Код повторно прочитан, frontend typecheck PASS. CLOSED; фактический browser click-flow reviewer не запускал.

## M9-R3 — Medium, CLOSED: публичная проверка теряет содержимое credential из-за DTO mismatch

**Пути:** `internal/httpapi/credentials.go`, verifyCredential; `web/src/Passport.tsx`, VerifyResponse/CredentialVerification.

API возвращает `payload` и `notice`; UI читает `credential` и `trust_notice`. Из-за этого посетитель видит статус подписи, но не subject/rank/date/criteria и полный notice этой установки. Подпись остаётся корректной, однако продукт не показывает, какие именно данные проверены.

**Evidence:** выполненный HTTP probe: public verify→200/statusactive, payload присутствует, credential отсутствует. В frontend `const credential=result?.credential` делает dl недоступным для реального API-ответа.

**Минимальная рекомендация:** согласовать DTO; показать SubjectName, Rank, CriteriaVersion/Hash, IssuedAt из подписанного payload и notice. Browser regression должен проверять реальные поля ответа HTTP, а не выдуманный UI fixture.

**Повторная проверка:** VerifyResponse теперь описывает payload/notice/key_id/public_key/issuer_trust. CredentialVerification показывает ID/subject/rank/exam/criteria/date из payload и notice. Успешный статус требует signature_valid:true, совпадение expected ID и полный локальный DTO; неполный ответ не превращается в valid. Revoked показывает явный статус и скрывает subject на уровне UI (сама опубликованная signed payload остаётся доступной API по исходной модели). Frontend typecheck PASS. CLOSED; visual/browser regression reviewer не запускал.

## Исправленный дефект test fixture, Low

Первый focused suite дал3FAIL render tests: credentials.Sign отвергал фиксированный IssuedAt2026-10-15T10:30:00Z, который был в будущем относительно UTC времени запуска. Fixture заменён на time.Now().UTC().Format(...). Полный повторный suite PASS; это дефект теста, а не renderer/crypto bypass.

## Проверенные свойства

- Ed25519 с domain separation; подпись покрывает json.Marshal фиксированной структуры Payload. Это собственный фиксированный формат oga-local-v1; JCS/Open Badges/W3C VC compatibility не заявляется.
- Envelope не содержит public key. KeyID должен совпадать с SHA-256 caller-pinned public key; IssuerID связан с тем же fingerprint. Signature base64url проверяется на canonical encoding. Все важные payload поля и competencies входят в подпись.
- Crypto test меняет каждое поле/подпись/keyID и проверяет wrong pinned key. Unknown JSON поля отвергаются HTTP decoder; клиент не может передать rank/issuer/key вместо серверных полей. Duplicate JSON keys сами по себе не дают неподписанных полномочий: проверяется структурированный итоговый payload.
- Store issue JOIN требует rank_award+completed/passed/score100 и совпадение exam/version/rank/hash. UNIQUE attempt_id блокирует повторный выпуск; публикация всегда0 на issue. Audit записывается в той же транзакции, reason отзыва не сохраняется.
- Публичный verify не отдаёт source/answers/checks/token/profile state; только подписанные данные, статус и публичный ключ. Default private→404 без имени; публикация требует отдельного consent. Passport public default404, содержит только опубликованные credentials. Profile name в публичном паспорте появляется лишь после отдельного opt-in паспорта.
- Revoke idempotent; отозванный документ сохраняет valid cryptographic signature, но status=revoked. Unpublish доступен API даже после revoke; private public-route снова404.
- Import/check не выпускает документ и не меняет БД. Сначала проверяет pinned signature, затем сверяет с сохранённым envelope и статусом. Тамpered Rank→signature_valid:false/statusinvalid в независимом probe.
- Key file atomic temp+sync+hardlink публикация,0600, symlink/nonregular/insecure permissions отвергаются; public half сверяется с seed. Concurrent24 initializer test возвращает один полный ключ. Private key не попадает в DTO/экспорт/renderer.
- Startup загружает `.data/issuer.key` либо explicit path после DB lock. При наличии credentials и пропавшем key — отказ; fingerprint другого ключа — отказ. Автоматическая ротация не происходит. Key nil в тестовом API даёт503 issue; экспорт требует успешной проверки.
- HTTP экспорт проверяет signed envelope и связь его полей с DB record перед renderer. Renderer не является trust boundary для выдачи; его самостоятельный validate не заменяет crypto.Verify. QR — только локальная ссылка, не внешняя аттестация.

## Фактические проверки

```sh
.tools/go/bin/go test -count=1 -timeout=45s ./internal/credentials ./internal/store ./internal/httpapi ./internal/credentialrender ./cmd/api
```

Первый запуск: crypto/store/httpapi PASS, renderer3FAIL из-за future fixture, cmd/api build без tests. Повторный после исправления: credentials0.006s, store0.212s, httpapi0.038s, credentialrender0.135s PASS; cmd/api успешно собран, своих tests нет.

Отдельный временный Go httptest probe на изолированной БД (exit0, удалён) подтвердил localhost422/numeric201 до R1fix; private unauth verify404; publish без consent422/с consent200; public active; tampered rank invalid; revoke/publicrevoked; API unpublish revoked200 и затем404. Данные реальной `.data/academy.db` не менялись. Прочитан новый `TestCredentialIssueForgeryPrivacyRevokeAndImport` с native HTTP/store flow.

Reviewer не запускал M9 browser E2E, фактический restart с отсутствующим/подменённым key, PDF visual render и QR decode, полный race-suite. Эти результаты должны фиксироваться отдельно автором интеграции. Макет PDF/PNG и layout для длинных имён не сертифицированы этим security review; font license/provenance и визуальную QA ведёт renderer agent. Локальный владелец ОС/исходников/БД остаётся доверенным оператором и может выпустить собственные документы; локальная подпись не превращает установку в доверенного внешнего эмитента.

После frontend fixes reviewer дополнительно выполнил `cd web && npm run typecheck`: PASS. Временный повторный localhost probe удалён, штатная БД не затрагивалась. Новые substantive изменения signature/issuance/public DTO/ключа после этого прохода потребуют дополнительного ревью.
