# P2 threat boundary

## Assets and actors

The API protects user email/password hashes, opaque login sessions, each owner's links and tags, and the integrity of link destinations. An actor may be unauthenticated, a normal authenticated user, or a person who controls a submitted link destination. This project runs on the learner's own machine.

## Main controls

- Passwords are salted and derived with PBKDF2-SHA256. Session and CSRF credentials are random and stored as SHA-256 hashes. Session cookies are HttpOnly and SameSite=Strict; state-changing browser requests also need a CSRF header and configured Origin.
- Every resource query includes the session-derived owner ID, and tag association occurs transactionally while checking that the tag has that owner.
- Link destinations accept HTTP(S) only, reject credentials and control characters, and are emitted only from persisted validated rows. LinkHub never makes an outbound request to the destination.
- PostgreSQL values use parameters. The API applies request-body and pagination limits, generic responses, server timeouts, and bounded connection count.
- Compose binds the API and PostgreSQL ports to loopback. The local `.env` file carries a generated developer password.

## Explicit limits

This is reference coursework, not a production identity system. Its in-memory request abuse controls, if added by a learner, do not protect a distributed public service. It has no email verification, password reset, MFA, distributed rate limiting, audited key rotation, managed secrets, or deployment hardening. The PostgreSQL service and API share a local developer network. Do not expose them publicly or reuse these settings as a production deployment. LinkHub does not compile or execute user code; academy code execution is a separate isolated worker boundary.
