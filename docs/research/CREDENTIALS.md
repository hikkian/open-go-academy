# Проверяемые credentials и границы доверия

Проверено 2026-10-15. Требования: MASTER_SPEC G3/G4. Статус: исследование; подпись, schema validation, status endpoints и сертификация продукта НЕ реализованы.

## Стандарты и выбранное направление

Актуальная canonical [Open Badges 3.0 specification](https://standards.1edtech.org/open-badges/specifications/standards/v3p0) использует VC 2.0 и context `https://purl.imsglobal.org/spec/ob/v3p0/context-3.0.3.json` вместе с `https://www.w3.org/ns/credentials/v2`. Старые VC 1.1 credentials имеют отдельные verification support models; нельзя смешивать их `issuanceDate`/`expirationDate` с VC2 `validFrom`/`validUntil`. Новая выдача должна следовать закреплённому профилю и schema, а не случайному старому примеру. В badge отражаются achievement/criteria/version/evidence и issuer. Полную поддержку API и baking изображений пока не заявлять.

[W3C VC Data Model 2.0, Recommendation 15 May 2025](https://www.w3.org/TR/2025/REC-vc-data-model-2.0-20250515/) разделяет issuer, holder, verifier. Валидная криптография доказывает происхождение утверждения и целостность; verifier самостоятельно решает, доверять ли issuer и его оценке. Это не проверка личности, отсутствия списывания, аккредитации или опыта работы.

Подпись должна реализовать конкретный стандартный профиль: [VC Data Integrity 1.0](https://www.w3.org/TR/vc-data-integrity/) с выбранной cryptosuite, например [EdDSA cryptosuites v1.0](https://www.w3.org/TR/vc-di-eddsa/), либо [VC JOSE/COSE](https://www.w3.org/TR/vc-jose-cose/). Самодельный «Ed25519 над JSON.stringify» не является автоматически Data Integrity. Для JSON-LD нужны нормативные canonicalization/transformation/proof verification; для JOSE нужны точные envelope/header/claim rules. Выбор библиотеки и проверка её interoperability остаются gate M9, конкретная implementation ещё не выбрана.

## Предлагаемая политика проекта

- Local self-issued: отдельные issuer ID и key, явная метка «локальное самостоятельное обучение». Корректная подпись не превращает установку в доверенного публичного эмитента. Экспорт без верифицированного proof отмечать «подпись не реализована».
- Hosted verified: оператор контролирует issuer identity/keys и серверную exam evidence, а verifier использует явно заданный trust registry/policy. Присвоение только после criteria/version/capstone; XP и LLM-ответ недостаточны.
- Ключи вне repository, browser, runner, outputs и обычных logs; least-privilege signer, key ID/rotation, durable public verification methods, audit trail. Отзыв при compromise и сохранение интерпретации старых подписей определяются до первой выдачи.
- PDF/PNG — отображение record; QR ведёт на `/verify/:credentialId` и сам по себе не подпись. Verify UI различает integrity, trusted issuer, criteria/evidence, expiry, revoked и status unavailable. Network error нельзя показывать как valid.

## Status и privacy

[Bitstring Status List v1.0](https://www.w3.org/TR/vc-bitstring-status-list/) предлагает пакетные статусы revocation/suspension, позволяющие проверять группу credentials без запроса к issuer о конкретном человеке. Сам status list также должен быть проверен криптографически; freshness/cache policy и обработка unavailable обязательны. Нельзя создавать отдельный публичный list для каждого студента: это упрощает tracking. Совместимость выбранного status профиля с Open Badges verifier потребуется проверить.

Privacy by default — проектная политика: публичное имя/passport/achievement и indexability только opt-in; предпочтительны псевдоним и случайный credential ID, минимальные evidence, без email и исходников. Уникальный URL не равен приватности; приватный verify record требует доступа или consent и отдаёт минимум metadata. История просмотров/исходники не публикуются. Удаление записи на сервере не уничтожает уже скачанные подписанные credentials: lifecycle, revocation и удаление PII — отдельные операции с прозрачными последствиями. Не заявлять соответствие законодательству без отдельной проверки.

## Gates M9

1. Version-pinned schemas/contexts и fixtures; independent known-good verifier подтверждает реальные signatures. Reject modified fields/proof, unknown/unauthorized key, unsupported suite, expired/revoked credential и invalid status signature.
2. Неизвестный issuer отдельно от tampered credential; status unavailable отдельно от revoked. Ротация/compromise, fresh/stale cache и проверка старого key покрыты tests.
3. Issuance API не допускает IDOR, client-assigned rank, недостаточные критерии, replay и concurrent duplicate issue; evidence/version фиксируются atomic transaction.
4. Consent по умолчанию false; публичный endpoint/search/export не раскрывает приватные имя/email/evidence. SSRF через issuer/context/status URL запрещён: allowlist, pinned local contexts, time/size budgets и запрет private-network redirects.
5. Документированный формат PDF/PNG/QR реально генерируется и проверен. Без выполнения официального [1EdTech certification процесса](https://certification.imsglobal.org/certification/verifiable-credentials/org.1edtech.ob.v3p0.testspec) нельзя заявлять «1EdTech certified»; schema/crypto tests сами по себе такой статус не дают.

Все gates сейчас NOT RUN. Следующий шаг — выбрать один proof/status профиль и библиотеки, закрепить normative snapshots/hash, затем сделать issuance/verify fixture с отрицательными tests до публичной выдачи.
