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

## Browser mutation rule

State-changing browser requests pass through a global same-origin check before the API endpoint runs. The protected HTTP methods are:

```text
POST
PUT
PATCH
DELETE
```

For requests that include browser origin metadata:

- `Origin`, when present, must match the request scheme and Host;
- `Sec-Fetch-Site`, when present, must be `same-origin` or `none`;
- cross-site and same-site/different-origin browser mutation requests return HTTP `403` with `{"error":"same-origin request required"}`.

Read-only GET requests are not rejected by this browser-mutation filter. Direct non-browser API clients such as curl normally send neither `Origin` nor `Sec-Fetch-Site` and remain supported.

This behavior is real-machine validated on the Raspberry Pi 5. YWD-DMR does not currently trust forwarded proxy headers for this decision. HTTPS reverse-proxy support needs an explicit trusted-proxy contract before it is considered production supported.

## Role authorization contract

The protected-operation hierarchy is:

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live opaque session, enforce their minimum server-side role, return HTTP `401` for missing/invalid authentication and HTTP `403` for insufficient role, and receive the authenticated principal through request context.

The first claimed account is Admin. Operator/Observer account management is not exposed yet.

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

Configuration state may be `missing`, `loaded`, `recovered`, or `error`. After the installation is claimed, daemon-owned `stage`/`next_step` values move to `claimed`/`identity` when identity is missing, or `identity_complete`/`network` when a known-good identity is present.

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

The full installed claim path has been validated on the Raspberry Pi 5.

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

Wrong username and wrong password both return:

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

The installed Pi 5 login path is validated, including generic wrong-credential behavior, successful restart login, logout invalidation, five-failure throttling, `429`/`Retry-After`, restart-cleared sessions/throttle, and durable admin persistence.

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

Unauthenticated/expired/missing-cookie requests return HTTP `200` with:

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

The token itself is never echoed.

## Setup validation

### Validate station identity

```text
POST /api/v1/setup/identity/validate
Content-Type: application/json
```

This endpoint is public and deliberately non-mutating. It exists so setup clients can normalize and validate form input without changing durable state.

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

A syntactically valid request with invalid setup fields still returns HTTP `200` because field validation is a normal result. Malformed JSON, unknown JSON fields, multiple top-level JSON values, or an oversized request return HTTP `400`.

Current validation rules:

- Callsign is trimmed, converted to uppercase, must be 3 to 12 ASCII letters/numbers, and must contain at least one letter and one digit.
- Base DMR ID must be from 1 through 9999999.
- ESSID must be from 0 through 99.

### Commit station identity

```text
POST /api/v1/setup/identity/commit
Content-Type: application/json
```

This is the first normal protected setup mutation. It requires a live **Admin** session and is also subject to the global browser same-origin rule.

Request:

```json
{
  "callsign": "N0CALL",
  "dmr_id": 1234567,
  "essid": 1
}
```

The daemon treats the request as an untrusted configuration candidate. It normalizes and validates the identity, commits it through the existing known-good configuration store, then advances daemon setup state only after durable commit succeeds.

A successful first commit returns HTTP `200`:

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

Subsequent successful commits increment the known-good revision and rotate the prior known-good snapshot using the existing configuration-store rules.

Failure behavior:

- missing or invalid session -> HTTP `401` with `{"error":"authentication required"}`;
- authenticated principal below Admin -> HTTP `403` with `{"error":"forbidden"}`;
- browser cross-origin mutation -> HTTP `403` with `{"error":"same-origin request required"}`;
- malformed/unknown JSON -> HTTP `400`;
- invalid identity candidate -> HTTP `400` with `error: invalid configuration candidate` and field errors;
- durable configuration-store failure -> HTTP `500` with a generic commit-failed response.

An invalid or unauthorized request must not replace the current known-good configuration or advance runtime setup state.

Identity has no external network dependency to test, so this transaction is:

```text
candidate -> normalize/validate -> durable commit
```

BrandMeister/network settings will later use the fuller path:

```text
candidate -> validate -> connectivity test -> commit
```

The identity-commit implementation and automated tests are on `dev`; installed Raspberry Pi validation is the next gate.

See [Setup and Security Phase](../developers/setup-security-phase.md), [Known-good Configuration Store](../developers/configuration-store.md), and [Authorization and Browser Mutation Protection](../developers/authorization-model.md).

## Planned rules

- JSON request/response for normal control operations.
- Server-side authorization on every protected operation.
- Observer / Operator / Admin role enforcement.
- Browser same-origin protection for state-changing requests.
- Opaque browser sessions and separately revocable device credentials.
- Secrets may be replaced but are never returned to a client after storage.
- Capability discovery is preferred over hard-coding server version checks in Android/WebUI.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

An OpenAPI document will become the machine-readable contract before external clients are considered stable.
