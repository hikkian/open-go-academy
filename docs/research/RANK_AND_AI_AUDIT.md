# Rank evidence and AI Tutor audit

Date: 2026-10-16. Read-only audit against `docs/MASTER_SPEC.md` sections E/G and the current rank, exam, project, credential and tutor implementation. No provider was contacted for this audit.

## Rank policy today

The verified rank sequence is server-controlled and sequential for the three ranks that currently have an exam:

| Award | Course/project eligibility | Final exam | Result |
|---|---|---|---|
| Zero | Three initial lesson exercises (`hello`, `repair`, `two-lines`) | 12 objective questions plus coding and debugging tasks; every check must pass | Local educational Zero award |
| Junior | All Zero and Junior exercise evidence, automatic P1 evidence, and an existing Zero award | 20 objective questions plus three Go tasks; every check must pass | Local educational Junior award |
| Junior+ | All Zero/Junior/Junior+ exercise evidence, P1 and P2 automatic project evidence, and an existing Junior award | 20 objective questions plus three API/access-control tasks; every check must pass | Local educational Junior+ award |

`GET /api/exams` computes missing prerequisites; `POST /api/exams/{id}/attempts` rechecks them on the server. `Store.StartExam` captures prerequisite and prior-rank evidence. Submission ignores any client-supplied score/rank; the server evaluates every answer and coding check. `FinishExamAttempt` revalidates the evidence and atomically stores a passed attempt plus rank award. XP does not award a rank. Credential issuance requires that exact passed attempt and rank award.

This is meaningfully more than one exam. A Junior+ award requires the lower-rank awards, course evidence and both project automatic checks. Project evidence is specifically `p1:automatic` / `p2:automatic`; each project response says that README/Git/design quality needs separate review. That review is currently **not** a rank-award prerequisite, so ranks claim only the deterministic Go Academy criteria that were checked. They do not attest identity, job readiness, or human-reviewed project quality.

Middle, Middle+ and Senior currently exist in the roadmap and competency catalog but have no exam versions, award eligibility or credentials. Consequently no one can earn those three ranks yet; this is the correct fail-closed state.

Relevant paths: `internal/assessment/assessment.go`, `internal/assessment/newcatalog.go`, `internal/httpapi/exams.go`, `internal/store/exams.go`, `internal/store/credentials.go`, `internal/learning/learning.go`, and `internal/projects/catalog.go`. Existing tests cover forged rank, missing prior award, missing project/course evidence, failed exam, replay, atomic award, and credential prerequisites.

## AI Tutor today

The product has a real optional provider-backed tutor through an OpenAI-compatible HTTP endpoint. It is configured by the server administrator and can use a loopback/local model or an explicitly allowed HTTPS provider. The platform has **not** trained or fine-tuned a model. It supplies mode-specific instructions in each request; it does not maintain a tutor conversation or learner memory.

Eleven modes exist: explain, simpler, deeper, example, hint, debug, review, quiz, interview, architecture and study plan. Without a configured provider, the lesson/course/project/exam paths remain available. The UI asks for per-request consent and a second opt-in before sending source code. The gateway limits endpoint resolution, time, request/response size, retries, and output tokens; the API reserves a conservative daily token budget. The model has no authority to run code, alter XP/rank, pass exams or issue credentials. Current tests use a local fake HTTP provider, not a live model.

The current context is limited to the student's question, a bounded public lesson overview, and code only with consent. It does **not** yet send compiler diagnostics, run/test feedback, verified skills or progress. Those are useful tutoring signals but are not prerequisites for a safe tutor. They are being added as separate explicit opt-ins, with server-derived progress and a strict size bound on user-provided diagnostics. The provider's location is currently inferred from URL scheme; the integration work is changing this to an explicit `local / cloud / unknown` admin label. Settings are currently environment-configured, not a learner-editable provider/key screen. Token counts are shown; dollar cost/provider health are not claimed.

Relevant paths: `internal/tutor/gateway.go`, `internal/httpapi/tutor.go`, `cmd/api/main.go`, `web/src/Tutor.tsx` and `internal/httpapi/tutor_test.go`.

## Remaining work for the intended six-rank path

- Define immutable Middle, Middle+ and Senior exam/rubric versions. Each must require its own course competencies and project artifacts, plus the previous verified rank; publication stays disabled until the associated curriculum/projects work.
- Link project evidence to each rank explicitly: P3 for Middle, P4/ScaleLab for Middle+, P5/Architecture Capstone for Senior. Machine-check behavior separately from design/review artifacts; require human approval only for criteria that genuinely need it, and keep local self-review distinct from a trusted external credential.
- Make the rank screen explain required course evidence, project checks, exam components, review state, and the precise reason an award is locked. No XP or AI response may satisfy a missing item.
- Complete the consented Tutor context feature and its API/UI privacy tests. Clearly state the configured model is provider-hosted or local as declared by server configuration, and is prompted for tutoring rather than trained on the learner.
- Recheck every bypass path at exam start, completion, duplicate/replay, project submission and credential issuance. Keep course unlock/skipping behavior aligned with the spec: learners may skip familiar lessons only when the corresponding verified competencies are demonstrated.
