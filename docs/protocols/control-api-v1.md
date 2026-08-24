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

## Setup validation

Phase 2 begins with server-side validation before persistence or authentication is added.

### Validate station identity

```text
POST /api/v1/setup/identity/validate
Content-Type: application/json
```

Request:

```json
{
  "callsign": "N0CALL",
  "dmr_id": 1234567,
  "essid": 1
}
```

Successful transport with valid fields:

```json
{
  "valid": true,
  "normalized": {
    "callsign": "N0CALL",
    "dmr_id": 1234567,
    "essid": 1
  },
  "errors": []
}
```

A syntactically valid request with invalid setup fields still returns HTTP `200` because field validation is a normal result. The response contains `valid: false` and one or more field errors containing `field`, `code`, and a plain-language `message`.

Malformed JSON, unknown JSON fields, multiple top-level JSON values, or an oversized request return HTTP `400`. Other methods return HTTP `405` with `Allow: POST`.

Current validation rules:

- Callsign is trimmed, converted to uppercase, must be 3 to 12 ASCII letters/numbers, and must contain at least one letter and one digit.
- Base DMR ID must be from 1 through 9999999.
- ESSID must be from 0 through 99.

This endpoint is deliberately **non-mutating**. It does not save configuration, create a user/session, connect to BrandMeister, or control/transmit radio traffic. It is temporarily available before authentication because it only validates caller-supplied non-secret values and changes no daemon state.

See [Setup and Security Phase](../developers/setup-security-phase.md) for the ordering of persistence, claim/auth, roles, and setup commits.

## Planned rules

- JSON request/response for normal control operations.
- Server-side authorization on every protected operation.
- Opaque browser sessions and separately revocable device credentials.
- Secrets may be replaced but are never returned to a client after storage.
- Capability discovery is preferred over hard-coding server version checks in Android/WebUI.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

An OpenAPI document will become the machine-readable contract before external clients are considered stable.
