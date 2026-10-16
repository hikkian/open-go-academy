# Учебный postmortem (вымышленный сценарий)

> Это упражнение, а не реальный инцидент или исторический факт.

## Влияние

Во время учебного пика downstream отвечал медленно. При реализации без fencing старый worker мог бы записать локальный success после того, как lease истёк и второй worker повторил вызов. Учебный ущерб: возможен двойной внешний side effect и неясное состояние части событий; реальных пользователей и данных нет.

## Временная шкала (относительное учебное время)

- T+00: latency downstream растёт, backlog oldest-age пересекает черновой alert threshold.
- T+05: on-call ограничивает новую нагрузку и сохраняет диагностические метрики без payload.
- T+12: принудительно останавливается один worker после отправки downstream request, до локальной фиксации.
- T+25: lease истекает, второй worker получает больший fence token; receiver возвращает ранее сохранённый результат по idempotency key.
- T+31: прежний worker возобновляется; compare-and-set по старому токену отклонён.
- T+45: backlog обрабатывается ограниченной скоростью; incident lead проверяет receipt и quarantine counts.

## Обнаружение и восстановление

Обнаружение — очередь oldest age и downstream timeout, коррелированные по event ID. Сначала защитили систему backpressure, затем восстановили обработку и проверили, что каждый event либо имеет receipt, либо находится в явном retry/quarantine state. Полезное действие: сверить внешний результат с локальным журналом, не запускать массовый blind replay.

## Причины и действия

Системная причина сценария — исходный контракт описывал «ровно один раз», хотя БД commit и сторонний HTTP side effect не образуют одну транзакцию. Содействующие условия: неизвестная receiver idempotency, недостаточно явно определённая lease expiry и метрики только по throughput.

| Действие | Владелец | Срок/критерий |
|---|---|---|
| Уточнить receiver idempotency/reconciliation contract | Product + integration | До утверждения SLO |
| Fault-injection тест kill-after-send и stale completion | Engineering | CI deterministic + disposable DB suite |
| Alert oldest pending age и quarantine rate | Operations | Game day page reaches on-call |
| Подтвердить retention, RPO/RTO и tenant audit | Security + owner | Restore and deletion exercise |

Разбор не обвиняет worker или on-call: система должна безопасно переживать задержки и повторы.
