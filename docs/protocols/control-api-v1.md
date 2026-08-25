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

## Setup status

```text
GET /api/v1/setup/status
```

This is a read-only daemon-owned setup summary. It never returns the stored station identity, credentials, tokens, passwords, or other protected settings.

A new unclaimed installation with no readable known-good configuration reports:

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

Configuration state may be `missing`, `loaded`, `recovered`, or `error`. After the installation is claimed, daemon-owned `stage`/`next_step` values move to `claimed`/`identity` when identity is missing, or `identity_complete`/`network` when a known-good identity is already present.

Other methods return HTTP `405` with `Allow: GET`.

## One-time installation claim

```text
POST /api/v1/setup/claim
Content-Type: application/json
```

This is the one deliberate unauthenticated mutating setup operation. A fresh appliance can be claimed only by presenting the high-entropy bootstrap code retrieved locally from the appliance with `sudo ywd-dmr claim-code`.

Request:

```json
{
  "claim_code": "YOUR-ONE-TIME-CODE",
  "username": "sysop",
  "password": "choose a long administrator password"
}
```

A successful claim returns HTTP `201` with non-secret administrator/session metadata:

```json
{
  "claimed": true,
  "username": "sysop",
  "role": "admin",
  "expires_at": "..."
}
```

The opaque session token is set only in the `ywd_dmr_session` HttpOnly cookie and is never returned in JSON. The cookie is `SameSite=Strict`; on HTTPS requests it is also marked `Secure`. Current LAN-test HTTP cannot use the Secure cookie flag, which is one reason this development build must not be publicly exposed.

Claim failure behavior:

- malformed/unknown JSON -> HTTP `400`;
- invalid username/password policy -> HTTP `400` with field errors;
- wrong claim code -> HTTP `403` with a generic claim-failed response;
- installation already claimed -> HTTP `409`;
- internal durable-state failure -> HTTP `500` without leaking internal details.

A successful durable claim consumes the bootstrap path. Reusing the claim request returns `409`, and daemon startup never regenerates a claim code merely because the old plaintext code is gone.

The full installed claim path has been validated on the Raspberry Pi 5, including durable claimed restart behavior, one-time code deletion/reuse rejection, and memory-only session invalidation after restart.

## Administrator login

```text
POST /api/v1/auth/login
Content-Type: application/json
```

Request:

```json
{
  "username": "sysop",
  "password": "your administrator password"
}
```

A successful login returns HTTP `200` with non-secret session metadata:

```json
{
  "authenticated": true,
  "username": "sysop",
  "role": "admin",
  "expires_at": "..."
}
```

The new opaque session token is set only in the `ywd_dmr_session` HttpOnly cookie. It is never returned in JSON. Sessions remain memory-only and are therefore cleared by daemon restart.

The login verifier uses the persisted PBKDF2-HMAC-SHA256 administrator password record created during claim. Wrong username and wrong password both return the same generic response:

```text
HTTP 401
{"error":"authentication failed"}
```

The password KDF is still evaluated when the username is wrong so the normal failure path does not become a cheap timing oracle.

Current direct-client login throttling is:

```text
5 failed logins from one direct client IP inside 5 minutes
-> block that IP for 60 seconds
```

While blocked, login returns HTTP `429` with `Retry-After` and:

```json
{
  "error": "login temporarily unavailable"
}
```

The throttle is memory-only and currently uses the direct TCP peer address. It deliberately does not trust `X-Forwarded-For`; proxy-aware behavior will be defined together with the future HTTPS/reverse-proxy deployment contract.

If the installation has not yet been claimed, login returns HTTP `409` with `{"error":"installation is not claimed"}`. Malformed or unknown JSON returns HTTP `400`. Other methods return HTTP `405` with `Allow: POST`.

## Administrator logout

```text
POST /api/v1/auth/logout
```

Logout invalidates the current in-memory session when a session cookie is present and sends an expired `ywd_dmr_session` cookie to the client. It returns HTTP `204` with no response body. Calling logout without a live session is harmless and still clears the browser cookie state.

Other methods return HTTP `405` with `Allow: POST`.

## Current session inspection

```text
GET /api/v1/auth/session
```

This endpoint reports whether the request carries a currently valid in-memory session cookie. Unauthenticated/expired/missing-cookie requests return HTTP `200` with:

```json
{
  "authenticated": false
}
```

A valid session returns only:

```json
{
  "authenticated": true,
  "username": "sysop",
  "role": "admin",
  "expires_at": "..."
}
```

The token itself is never echoed. Current sessions are intentionally memory-only, so daemon restart invalidates them while the durable claimed/admin state remains.

## Setup validation

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

This endpoint is deliberately **non-mutating**. It does not save the values. Configuration commits remain blocked until server-side role authorization and browser mutation protections are implemented.

See [Setup and Security Phase](../developers/setup-security-phase.md) for the ordering of persistence, claim/auth, roles, and setup commits.

## Planned rules

- JSON request/response for normal control operations.
- Server-side authorization on every protected operation.
- Origin/CSRF protection for authenticated browser mutations.
- Observer / Operator / Admin role enforcement.
- Opaque browser sessions and separately revocable device credentials.
- Secrets may be replaced but are never returned to a client after storage.
- Capability discovery is preferred over hard-coding server version checks in Android/WebUI.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

An OpenAPI document will become the machine-readable contract before external clients are considered stable.
