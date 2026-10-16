# Threat model — шаблон

Для каждой строки укажите asset, trust boundary, actor, precondition, impact, enforceable control, test, residual risk и response owner. Не ставьте risk = 0 только потому, что добавили authentication.

| Threat / abuse case | Asset and boundary | Actor/precondition | Impact | Control and deterministic test | Residual risk / response |
|---|---|---|---|---|---|
| Forged tenant ID | | | | | |
| Replay / idempotency conflict | | | | | |
| Stale worker after lease expiry | | | | | |
| Poison event / resource exhaustion | | | | | |
| Credential compromise / PII leak | | | | | |
| Operator redrive / restore error | | | | | |

Identify trust in reviewer evidence separately: an AI grader or learner cannot stand in for independent authenticated human review.
