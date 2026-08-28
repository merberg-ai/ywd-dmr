# YWD Control API v1

Base path: `/api/v1/`

The Control API is the authoritative request/response API used by the WebUI, future Android client, and other approved clients. The WebUI does not receive private internal shortcuts that bypass the public client model.

## Foundation endpoints

Read-only:

- `GET /api/v1/health`
- `GET /api/v1/system`
- `GET /api/v1/status`
- `GET /api/v1/capabilities`

Mutating radio/PTT controls remain absent until TX leases, TX timeout behavior, and the DMR runtime exist.

## Browser mutation rule

State-changing browser requests (`POST`, `PUT`, `PATCH`, `DELETE`) pass through a global same-origin check before the API endpoint runs. `Origin`, when present, must match the request scheme and Host; `Sec-Fetch-Site`, when present, must be `same-origin` or `none`.

Cross-origin mutations return HTTP `403`. Direct non-browser API clients that send neither browser header remain supported.

## Role authorization

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live opaque session and enforce their minimum role.

## Setup status

```text
GET /api/v1/setup/status
```

This daemon-owned summary never returns stored identity, master credentials, passwords, or session tokens.

Identity-only durable configuration reports approximately:

```json
{
  "stage": "identity_complete",
  "next_step": "network",
  "configuration": {
    "state": "loaded",
    "revision": 1,
    "identity_configured": true,
    "network_configured": false,
    "recovered": false
  }
}
```

After a durable tested network commit:

```json
{
  "stage": "network_complete",
  "next_step": "audio",
  "configuration": {
    "state": "loaded",
    "revision": 2,
    "identity_configured": true,
    "network_configured": true,
    "recovered": false
  }
}
```

## One-time installation claim

```text
POST /api/v1/setup/claim
```

This is the one deliberate unauthenticated setup mutation. It requires the high-entropy bootstrap code retrieved locally with:

```bash
sudo ywd-dmr claim-code
```

Successful claim creates the first Admin and sets the opaque `ywd_dmr_session` only as an HttpOnly `SameSite=Strict` cookie. On HTTPS the cookie is also `Secure`.

## Administrator login/logout/session

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username/password use a generic HTTP `401` result. Sessions and login throttling are memory-only; daemon restart logs browser sessions out while durable Admin credentials remain.

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

## Commit station identity

```text
POST /api/v1/setup/identity/commit
Content-Type: application/json
```

Admin-only and browser same-origin protected. The daemon normalizes/validates, commits through the known-good store, and advances setup state only after durable commit succeeds.

If a tested schema-2 network already exists, an identity commit preserves the network metadata/secret and creates a new schema-2 revision rather than silently erasing network setup.

## BrandMeister candidate request

The following three protected endpoints use the same request shape:

```text
POST /api/v1/setup/network/validate
POST /api/v1/setup/network/test
POST /api/v1/setup/network/test-and-commit
```

Request:

```json
{
  "backend": "brandmeister",
  "master_address": "3103.master.brandmeister.network",
  "master_port": 62031,
  "registration_frequency_hz": 446525000,
  "password": "your BrandMeister hotspot password"
}
```

Fields:

- `backend` — currently `brandmeister`.
- `master_address` — hostname/IP only, not a URL or host:port string.
- `master_port` — `0` normalizes to Homebrew default `62031`.
- `registration_frequency_hz` — nominal Homebrew registration metadata from `100000000` through `999999999` Hz. Simplex registration uses the same value as RX and TX. It does not create an RF transmit path.
- `password` — BrandMeister Hotspot Security password, non-empty, at most 20 characters, no control characters.

All three endpoints require Admin authentication because the request contains a credential. Browser same-origin protection applies.

## Validate DMR network candidate

```text
POST /api/v1/setup/network/validate
```

Local validation only. It does not contact BrandMeister or persist anything.

Successful response:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "3103.master.brandmeister.network",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "errors": []
}
```

The password is never echoed.

## Test BrandMeister without persistence

```text
POST /api/v1/setup/network/test
```

This remains the diagnostic operation. It requires committed station identity, runs the bounded real Homebrew login/auth/config handshake, sends `RPTCL`, and returns the test result without changing durable network state.

Normal result:

```json
{
  "ok": true,
  "backend": "brandmeister",
  "reason": "ok",
  "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
  "duration_ms": 294
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

The Pi 5 has proven `reason: ok` against the public master using the YWD-DMR numeric/date-style Homebrew software identifier and `MMDVM_DMO` compatibility profile.

`/network/test` must not change configuration revision, rotate snapshots, persist the network candidate, or advance setup stage.

## Test and commit BrandMeister configuration

```text
POST /api/v1/setup/network/test-and-commit
```

This is the durable network operation.

The daemon performs:

```text
request candidate
  -> strict JSON decode
  -> local normalize/validate
  -> read committed station identity
  -> real BrandMeister RPTL/RPTK/RPTC test
  -> if test fails: stop, committed=false
  -> if test succeeds: commit the exact same normalized candidate
```

There is no reusable "test passed" token. The same in-memory candidate is tested and committed in one request.

### Successful commit

Example:

```json
{
  "committed": true,
  "revision": 2,
  "network": {
    "backend": "brandmeister",
    "master_address": "3103.master.brandmeister.network",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "test": {
    "ok": true,
    "backend": "brandmeister",
    "reason": "ok",
    "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
    "duration_ms": 294
  }
}
```

The password is not returned.

A successful first network commit promotes identity-only schema 1 to schema 2, rotates the prior known-good snapshot to `known-good.previous.json`, and stores the Hotspot Security password in a restricted revision-bound secret file rather than in normal known-good JSON.

### Test rejected — no commit

A normal BrandMeister rejection remains HTTP `200` because the API operation itself completed:

```json
{
  "committed": false,
  "test": {
    "ok": false,
    "backend": "brandmeister",
    "reason": "auth",
    "message": "...",
    "duration_ms": 200
  }
}
```

Known-good revision and setup state remain unchanged.

### Durable write failure after successful test

If the real network test succeeds but durable storage fails, the endpoint returns HTTP `500` with a generic storage error and the non-secret successful test result. It never reports a commit that did not reach durable known-good state.

## Homebrew wire behavior used by setup testing

```text
RPTL -> RPTACK + salt
RPTK -> RPTACK
RPTC -> RPTACK
RPTCL
```

`RPTK` contains SHA-256 of `salt || password`. `RPTC` is the fixed 302-byte Homebrew configuration packet. Setup testing contains no `DMRD` voice/data transmit path.

BrandMeister device ID:

```text
ESSID 0     -> base DMR ID
ESSID 1..99 -> (base DMR ID * 100) + ESSID
```

The accepted current registration profile uses a YWD-DMR-owned numeric/date-style software/version field plus `MMDVM_DMO` as the simplex Homebrew compatibility profile. Normal product/version reporting still identifies the application as YWD-DMR.

## Secret contract

A client may supply or replace the network password but may not retrieve it later.

Durable schema-2 configuration contains only:

```json
"password_set": true
```

The secret is stored separately in a restricted daemon-only revision-bound file. API responses, setup status, WebUI result output, validation output, and normal logs must never expose it.

## General API rules

- JSON request/response for normal control operations.
- Strict JSON request decoding; unknown fields rejected.
- Server-side authorization on every protected operation.
- Observer / Operator / Admin role enforcement.
- Browser same-origin protection for state-changing requests.
- Opaque browser sessions and separately revocable future device credentials.
- Secrets may be supplied/replaced but are never returned after storage.
- Capability discovery is preferred over hard-coded server version checks.
- API v1 remains compatible for the lifetime of v1; incompatible changes require a new protocol version.

See [Known-good Configuration Store](../developers/configuration-store.md), [DMR Network Backend and BrandMeister Setup Contract](../developers/network-backend-contract.md), and [BrandMeister Live Test Notes](../developers/brandmeister-live-test-notes.md).
