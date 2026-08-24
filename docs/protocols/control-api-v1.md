# YWD Control API v1

Base path: `/api/v1/`

The Control API is the authoritative request/response API used by the WebUI, future Android client, and other approved clients. The WebUI must not receive private internal shortcuts that bypass the public client model.

## Foundation endpoints

Currently implemented and read-only:

- `GET /api/v1/health`
- `GET /api/v1/system`
- `GET /api/v1/status`
- `GET /api/v1/capabilities`

Mutating radio controls are intentionally absent until authentication, authorization, PTT leases, and TX timeout behavior exist.

## Planned rules

- JSON request/response for normal control operations.
- Server-side authorization on every protected operation.
- Opaque browser sessions and separately revocable device credentials.
- Secrets may be replaced but are never returned to a client after storage.
- Capability discovery is preferred over hard-coding server version checks in Android/WebUI.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

An OpenAPI document will become the machine-readable contract before external clients are considered stable.
