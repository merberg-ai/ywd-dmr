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

State-changing browser requests (`POST`, `PUT`, `PATCH`, `DELETE`) pass through a global same-origin check before the API endpoint runs. `Origin`, when present, must match the request scheme and Host; `Sec-Fetch-Site`, when present, must be `same-origin` or `none`. Cross-site and same-site/different-origin mutations return HTTP `403` with `{"error":"same-origin request required"}`.

Read-only GET requests are not blocked by this filter. Direct non-browser API clients such as curl normally send neither header and remain supported. This behavior is Raspberry Pi 5 validated.

## Role authorization contract

Protected hierarchy:

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live opaque session, enforce their minimum role, return HTTP `401` for missing/invalid authentication and HTTP `403` for insufficient role, and receive the authenticated principal through request context.

## Setup status

```text
GET /api/v1/setup/status
```

This read-only daemon-owned summary never returns stored identity, credentials, tokens, passwords, or other protected settings. A fresh installation reports `unclaimed`; once claimed and identity is durable, setup moves to `identity_complete / network`.

## One-time installation claim

```text
POST /api/v1/setup/claim
```

This is the one deliberate unauthenticated setup mutation. It requires the high-entropy bootstrap code retrieved locally with:

```bash
sudo ywd-dmr claim-code
```

Successful claim returns HTTP `201` with non-secret Admin/session metadata and sets the opaque `ywd_dmr_session` only as an HttpOnly `SameSite=Strict` cookie. The token is never returned in JSON. On HTTPS the cookie is also `Secure`.

## Administrator login/logout/session

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username and wrong password use the same generic HTTP `401` failure and execute the password KDF. Five failures from one direct client IP inside five minutes cause a 60-second memory-only block. Successful login sets a fresh opaque cookie; daemon restart intentionally clears sessions/throttle while preserving the durable Admin account.

## Validate station identity

```text
POST /api/v1/setup/identity/validate
Content-Type: application/json
```

Public and non-mutating. Request:

```json
{
  "callsign": "N0CALL",
  "dmr_id": 1234567,
  "essid": 1
}
```

Syntactically valid requests with invalid fields return HTTP `200` with `valid: false`; malformed/unknown JSON returns HTTP `400`.

## Commit station identity

```text
POST /api/v1/setup/identity/commit
Content-Type: application/json
```

Admin-only and browser same-origin protected. The daemon normalizes and validates the candidate, commits it through the known-good store, then advances runtime setup state only after durable commit succeeds.

The installed Pi 5 validation proved revision `1 -> 2`, `0600 ywd-dmr:ywd-dmr` current/previous snapshots, invalid-candidate preservation, restart persistence, and recovery of API-created previous revision 1 after deliberate corruption of current revision 2.

## Validate DMR network candidate

```text
POST /api/v1/setup/network/validate
Content-Type: application/json
```

Admin-only because the request contains the BrandMeister Hotspot Security password. Browser same-origin protection also applies.

Current request:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "registration_frequency_hz": 446525000,
  "password": "your BrandMeister hotspot password"
}
```

Fields:

- `backend` — currently `brandmeister`.
- `master_address` — hostname/IP only, not a URL or host:port string.
- `master_port` — `0` normalizes to Homebrew default `62031`.
- `registration_frequency_hz` — nominal Homebrew registration frequency from `100000000` through `999999999` Hz. The BrandMeister tester reports this same value as RX and TX for simplex/DMO registration. It is metadata only and does not create an RF transmit path.
- `password` — BrandMeister Hotspot Security password, non-empty, at most 20 characters, no control characters.

Successful validation response contains only non-secret normalized data:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "master.example.net",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "errors": []
}
```

The password is never echoed. Validation does not contact BrandMeister, write known-good configuration, increment a revision, or advance setup stage.

A syntactically valid request with invalid fields returns HTTP `200` with `valid: false` and field errors. Malformed/unknown JSON returns HTTP `400`. Missing/invalid Admin authentication returns HTTP `401`; cross-origin browser requests return HTTP `403` before validation runs.

## Test BrandMeister connectivity and credentials

```text
POST /api/v1/setup/network/test
Content-Type: application/json
```

Admin-only, same-origin protected for browsers, and deliberately non-persisting. It uses the same request shape and validation rules as `/network/validate`.

Before opening UDP, the daemon requires a locally valid network candidate, an already committed/readable station identity, and the real network tester service. If station identity has not been committed, the endpoint returns HTTP `409`.

Normal test outcomes return HTTP `200` with a non-secret result:

```json
{
  "ok": true,
  "backend": "brandmeister",
  "reason": "ok",
  "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
  "duration_ms": 123
}
```

Possible `reason` values:

```text
ok
login
auth
config
timeout
network
unavailable
```

A failed live test is a normal test result rather than an HTTP server failure. The response never contains the submitted password, the master salt, or the SHA-256 authentication response.

### Wire behavior

The tester performs only:

```text
RPTL -> RPTACK+salt
RPTK -> RPTACK
RPTC -> RPTACK
RPTCL
```

`RPTK` contains SHA-256 of `salt || password`. `RPTC` is the fixed Homebrew 302-byte configuration packet. The tester contains no `DMRD` transmit path.

### RPTC registration metadata

For the current simplex/software registration the tester places `registration_frequency_hz` into both the 9-byte RX and TX frequency fields. It uses informational power `01`, color code `01`, slot/mode marker `4`, and zero location/height pending later station-location settings.

The frequency and power fields are Homebrew registration metadata. They do not imply that YWD-DMR owns or keys RF hardware.

### Station identity used by the test

The test always reads callsign/DMR ID/ESSID from the known-good configuration store. BrandMeister device ID is derived as:

```text
ESSID 0     -> base DMR ID
ESSID 1..99 -> (base DMR ID * 100) + ESSID
```

The Homebrew callsign field is eight characters wide. If the stored generic station callsign cannot fit, the live test returns `reason: config` rather than silently truncating it.

### Installed live-test status

The Pi 5 has proven the endpoint's authorization, origin protection, failure classification, and non-persistence rules. A real BrandMeister retry with the verified Hotspot Security credential reached `reason: config`, proving that both `RPTL` login and `RPTK` authentication were accepted. The original zero-frequency/zero-power RPTC was the remaining rejected stage.

The current `dev` request now includes explicit registration-frequency metadata for the focused RPTC retest.

### Persistence rule

`/network/test` must not change the known-good revision, create/rotate `known-good.previous.json`, persist master/frequency/password data, or advance setup state. Those rules apply on both success and failure.

## Network commit contract

The durable network commit endpoint is **not exposed yet**.

Network configuration remains:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

The current known-good schema remains identity-only. A network commit requires an explicit schema/migration change after the complete live tester succeeds on the installed Pi.

See [DMR Network Backend and BrandMeister Setup Contract](../developers/network-backend-contract.md), [BrandMeister Candidate Validation Notes](../developers/network-validation-notes.md), and [BrandMeister Live Test Notes](../developers/brandmeister-live-test-notes.md).

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
