# Security Policy

YWD-DMR controls network voice transmission and stores network credentials, so security is a product requirement rather than an optional deployment add-on.

## Foundation status

The current development scaffold has only read-only status endpoints. Authentication, mutating configuration, audio streaming, and PTT are not implemented yet. The default development listener is loopback-only (`127.0.0.1`). Do not expose development builds directly to the public internet.

## Design requirements

- No default administrator password.
- First-run setup uses a short-lived one-time claim mechanism.
- Browser sessions use secure server-side/opaque session identifiers rather than secrets stored in JavaScript local storage.
- Native clients receive separately revocable device credentials.
- Admin, Operator, and Observer permissions are enforced server-side.
- PTT requires a server-owned renewable lease plus a server-side transmit timeout.
- Secrets are never returned to a client after storage.
- WebSocket endpoints use authentication, Origin validation where appropriate, bounded messages/queues, and rate limits.
- Third-party vocoders run out-of-process and should not require root.
- Support bundles redact credentials automatically.
- Updates verify official release artifacts before activation and automatically roll back failed releases.

Until a formal private vulnerability-reporting path is published, avoid posting active credentials or private support bundles in public issues.
