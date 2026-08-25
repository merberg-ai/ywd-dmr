# YWD Control API v1

Base path: `/api/v1/`

The Control API is the authoritative request/response API used by the WebUI, future Android client, and other approved clients. The WebUI must not receive private internal shortcuts that bypass the public client model.

## Foundation endpoints

Currently implemented and read-only:

- `GET /api/v1/health`
- `GET /api/v1/system`
- `GET /api/v1/status`
- `GET /api/v1/capabilities`

Mutating radio controls are intentionally absent until PTT leases, TX timeout behavior, and the DMR runtime exist.

## Browser mutation rule

State-changing browser requests pass through a global same-origin check before the API endpoint runs:

```text
POST
PUT
PATCH
DELETE
```

For requests carrying browser origin metadata:

- `Origin`, when present, must match the request scheme and Host;
- `Sec-Fetch-Site`, when present, must be `same-origin` or `none`;
- cross-site and same-site/different-origin mutations return HTTP `403` with `{"error":"same-origin request required"}`.

Read-only GET requests are not blocked by this filter. Direct non-browser API clients such as curl normally send neither header and remain supported.

This behavior is Raspberry Pi 5 validated. YWD-DMR does not currently trust forwarded proxy headers for this decision; production HTTPS reverse-proxy behavior needs an explicit trusted-proxy contract.

## Role authorization contract

Protected hierarchy:

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live opaque session, enforce their minimum role, return HTTP `401` for missing/invalid authentication and HTTP `403` for insufficient role, and receive the authenticated principal through request context.

The first claimed account is Admin. Operator/Observer account management is not exposed yet.

## Setup status

```text
GET /api/v1/setup/status
```

This read-only daemon-owned summary never returns stored identity, credentials, tokens, passwords, or other protected settings.

A fresh installation reports:

```json
{
  "claimed": false,
  "stage": "unclaimed",
  "next_step": "claim",
  "configuration": {
    "state": "missing",
    "identity_configured": false,
    "recovered": false
  }
}
```

Configuration state may be `missing`, `loaded`, `recovered`, or `error`. Once claimed and identity is durable, setup moves to `identity_complete` / `network`.

## One-time installation claim

```text
POST /api/v1/setup/claim
```

This is the one deliberate unauthenticated setup mutation. It requires the high-entropy bootstrap code retrieved locally with:

```bash
sudo ywd-dmr claim-code
```

Successful claim returns HTTP `201` with non-secret Admin/session metadata and sets the opaque `ywd_dmr_session` only as an HttpOnly `SameSite=Strict` cookie. The token is never returned in JSON. On HTTPS the cookie is also `Secure`.

Claim failure behavior includes malformed JSON `400`, wrong code `403`, and already-claimed `409`. The complete claim lifecycle is Pi 5 validated.

## Administrator login/logout/session

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username and wrong password both return generic HTTP `401` authentication failure and both execute the password KDF. Five failures from one direct client IP inside five minutes cause a 60-second memory-only block; blocked login returns HTTP `429` plus `Retry-After`.

Successful login sets a fresh opaque cookie. Logout invalidates the in-memory session and expires the cookie. Daemon restart intentionally clears sessions/throttle but preserves durable Admin state. The complete behavior is Pi 5 validated.

## Validate station identity

```text
POST /api/v1/setup/identity/validate
Content-Type: application/json
```

This endpoint is public and deliberately non-mutating.

Request:

```json
{
  "callsign": "N0CALL",
  "dmr_id": 1234567,
  "essid": 1
}
```

Valid response:

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

Syntactically valid requests with invalid fields still return HTTP `200` with `valid: false`. Malformed/unknown JSON returns HTTP `400`.

## Commit station identity

```text
POST /api/v1/setup/identity/commit
Content-Type: application/json
```

This is Admin-only and subject to browser same-origin protection.

Request uses the same station-identity shape as validation. The daemon normalizes and validates the untrusted candidate, commits it through the known-good store, then advances runtime setup state only after durable commit succeeds.

First successful commit returns:

```json
{
  "committed": true,
  "revision": 1,
  "identity": {
    "callsign": "N0CALL",
    "dmr_id": 1234567,
    "essid": 1
  }
}
```

Later successful commits increment the revision and rotate the previous snapshot. Unauthorized, cross-origin, malformed, invalid, or storage-failed requests must not replace current known-good state.

The installed Pi 5 validation proved revision `1 -> 2`, `0600 ywd-dmr:ywd-dmr` current/previous snapshots, invalid-candidate preservation, restart persistence, and recovery of API-created previous revision `1` after deliberate corruption of current revision `2`.

## Validate DMR network candidate

Implemented on `dev`:

```text
POST /api/v1/setup/network/validate
Content-Type: application/json
```

This endpoint is **Admin-only** even though it does not persist anything, because the request contains the BrandMeister hotspot password. Browser same-origin protection also applies.

Request:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your BrandMeister hotspot password"
}
```

`master_port: 0` means use the current BrandMeister/Homebrew default `62031`.

A valid response contains only non-secret normalized data:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "master.example.net",
    "master_port": 62031,
    "password_set": true
  },
  "errors": []
}
```

The password is never echoed. Validation does not contact BrandMeister, write known-good configuration, increment a revision, or advance setup stage.

A syntactically valid request with invalid fields returns HTTP `200` with `valid: false` and field errors. Malformed/unknown JSON returns HTTP `400`. Missing/invalid Admin authentication returns HTTP `401`; cross-origin browser requests return HTTP `403` before validation runs.

Current validation rules:

- only backend `brandmeister` is accepted;
- master address must be a hostname or IP address, not a URL/path/embedded port;
- port must normalize into `1..65535`;
- password must be non-empty/non-whitespace, at most 256 characters, and contain no control characters.

## Network test/commit contract

The real network test and commit endpoints are **not exposed yet**.

Network configuration must use:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

A local validation result is never proof that credentials work.

`internal/dmrnet` defines structured backend test reasons including:

```text
ok
login
auth
config
timeout
network
unavailable
```

The future BrandMeister tester will map the established Homebrew login/auth/config handshake into those reasons, and durable network commit will remain closed until a real test succeeds.

See [DMR Network Backend and BrandMeister Setup Contract](../developers/network-backend-contract.md).

## General API rules

- JSON request/response for normal control operations.
- Server-side authorization on every protected operation.
- Observer / Operator / Admin role enforcement.
- Browser same-origin protection for state-changing requests.
- Opaque browser sessions and separately revocable future device credentials.
- Secrets may be supplied/replaced but are never returned to a client after storage or in validation/test results.
- Capability discovery is preferred over hard-coded server version checks.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

An OpenAPI document will become the machine-readable contract before external clients are considered stable.
