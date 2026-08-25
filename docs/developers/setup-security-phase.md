# Setup and Security Phase

This phase turns the tested YWD-DMR appliance into something a normal operator can safely configure from a browser before DMR networking and audio are enabled.

## Goal

The phase is complete when a fresh installation can be claimed once, an administrator can enter and validate the station identity, settings can be tested before they replace known-good configuration, and the daemon can safely expose protected setup/control operations to the WebUI and future Android client.

This work intentionally comes before the long-lived BrandMeister connection. Network credentials, DMR identity, audio choices, and vocoder settings all need one authoritative configuration/security model rather than separate ad-hoc files or browser-only state.

## Proven setup/security foundation

The following chain is now installed-appliance validated on the Raspberry Pi 5:

```text
one-time claim
  -> durable Admin password verifier
  -> memory-only authenticated session
  -> Observer / Operator / Admin authorization
  -> browser same-origin mutation protection
  -> protected station-identity candidate
  -> atomic known-good commit
  -> previous-snapshot rotation
  -> restart persistence and explicit recovery
```

The real identity-commit exercise produced revision `1`, then revision `2`, confirmed both snapshots as mode `0600` and owner `ywd-dmr:ywd-dmr`, proved a rejected invalid candidate did not change either file, loaded revision `2` after restart, then deliberately corrupted current revision `2` and recovered the API-created previous revision `1` snapshot with an explicit runtime `recovered` state and journal warning.

See [Protected Station-Identity Commit Validation Notes](identity-commit-validation-notes.md).

## 1. Radio identity model and validation

First-run setup collects:

- **Callsign** — the operator/station's base amateur-radio callsign.
- **DMR ID** — the base numeric DMR ID, separate from any hotspot suffix.
- **ESSID** — a number from 0 through 99 used by networks/backends that need a distinct hotspot/device identity.

The public, non-mutating validation API is:

```text
POST /api/v1/setup/identity/validate
```

The daemon owns normalization and validation. Clients may perform friendly local checks, but server validation is authoritative.

## 2. Known-good configuration store

The daemon keeps:

```text
/var/lib/ywd-dmr/known-good.json
/var/lib/ywd-dmr/known-good.previous.json
```

A candidate is validated before durable state changes. Later commits rotate the old current snapshot to the previous rollback snapshot before atomically replacing current. If current is unreadable or invalid, startup can explicitly recover a valid previous snapshot.

The real protected identity API has now proven this same rotation/recovery path on installed storage rather than only through unit fixtures.

See [Known-good Configuration Store](configuration-store.md).

## 3. Daemon-owned setup status

```text
GET /api/v1/setup/status
```

The public setup status reports coarse configuration health and setup progress without returning stored identity or secrets.

Configuration state may be:

```text
missing
loaded
recovered
error
```

A recovered previous snapshot is never silently treated as normal; setup status and the service journal both say recovery occurred.

## 4. One-time installation claim

A fresh appliance creates a high-entropy claim code in the restricted state directory and exposes it locally through:

```bash
sudo ywd-dmr claim-code
```

The deliberate unauthenticated bootstrap mutation is:

```text
POST /api/v1/setup/claim
```

Successful claim creates the first Admin password verifier, burns the one-time bootstrap path, and creates an opaque memory-only browser session. Claim is fully installed-appliance validated. The first Pi 5 PBKDF2/claim timing measurement was `0.516708s`.

See [One-time Claim Validation Notes](claim-validation-notes.md).

## 5. Administrator login

Implemented and real-machine validated:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username and wrong password use the same generic failure and execute the password KDF. Login sessions and throttling are memory-only; daemon restart clears them while durable Admin state survives.

Pi 5 measurements included wrong username `0.520663s`, wrong password `0.520534s`, valid login `0.519940s`, and post-restart valid login `0.517669s`.

See [Administrator Authentication Validation Notes](auth-validation-notes.md).

## 6. Roles and browser mutation protection

Server-side hierarchy:

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live session and minimum role.

State-changing browser methods (`POST`, `PUT`, `PATCH`, `DELETE`) pass through same-origin protection. `Origin`, when present, must match the request origin; `Sec-Fetch-Site`, when present, must be `same-origin` or `none`. Direct non-browser API clients remain usable.

This layer is installed-machine validated. See [Authorization and Browser Mutation Protection](authorization-model.md) and [Authorization Validation Notes](authorization-validation-notes.md).

## 7. Protected station-identity commit

Implemented and now fully installed-machine validated:

```text
POST /api/v1/setup/identity/commit
```

It is Admin-only and same-origin protected for browsers.

Identity has no external service to test, so its transaction is:

```text
candidate -> normalize/validate -> durable commit
```

Runtime setup state advances only after durable commit succeeds. The Pi 5 validation proved revision increment, previous-snapshot rotation, restart persistence, invalid-candidate preservation, and recovery from an API-created previous snapshot.

## 8. BrandMeister/network configuration — active slice

Network configuration adds an external dependency and therefore uses a stricter transaction:

```text
candidate
  -> local validation
  -> real connectivity/authentication test
  -> durable commit
```

Local validation alone must never be called a successful network test.

### Candidate model

The first request model contains:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your hotspot password"
}
```

CallsSign/DMR ID/ESSID are not duplicated here. The network backend receives the already known-good station identity.

`master_port: 0` normalizes to the normal BrandMeister/Homebrew UDP port `62031`.

### Protected local validation

Implemented on `dev`:

```text
POST /api/v1/setup/network/validate
```

This endpoint requires Admin authentication because the request contains the BrandMeister hotspot password. It is also subject to browser same-origin filtering.

Its response never echoes the password. A valid result returns only a non-secret normalized summary such as:

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

No known-good file, revision, or setup stage changes during local network validation.

### Backend test contract

`internal/dmrnet` defines the real connectivity-test interface before the live tester is written. Structured result reasons include:

```text
ok
login
auth
config
timeout
network
unavailable
```

The first real BrandMeister tester will map the established Homebrew master handshake stages into these reasons. That lets the WebUI distinguish a rejected ID, wrong hotspot password, rejected configuration, and an unreachable master.

The actual network test/commit endpoints remain intentionally closed until the real tester exists.

See [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md).

## 9. BrandMeister live tester and durable network commit

The next implementation sequence is:

1. implement a bounded short-lived Homebrew login/auth/config probe;
2. expose a protected test endpoint that uses it;
3. prove that failed tests do not modify known-good state;
4. make an explicit known-good schema/migration change for network settings;
5. allow durable network commit only after the candidate passes the real test;
6. then build the long-lived connection/reconnect state machine.

The tester must never transmit DMR voice/data, log the password, or turn a DNS/UDP socket-open result into a false credential success.

## 10. Guided WebUI wizard

The WebUI will use the same setup APIs as future Android/CLI clients, with plain-language DMR explanations and useful troubleshooting for failed network tests.

## Security boundary during development

The current LAN test dashboard remains development-only. Do **not** router-forward or publicly expose it.

The public identity-validation endpoint is non-mutating. `POST /api/v1/setup/claim` remains the one deliberate unauthenticated setup mutation requiring the local bootstrap code. Normal login/logout, role authorization, browser-origin filtering, and protected identity commit are validated.

The new network-validation endpoint is Admin-only because it accepts a secret. There is still no network commit, BrandMeister production session, radio control, or PTT endpoint. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, and device credentials are separate work.

## Current implementation status

- [x] Radio identity validation and public validation API.
- [x] Known-good configuration store and startup recovery model.
- [x] One-time claim and durable first Admin.
- [x] Password login/logout/throttling.
- [x] Observer / Operator / Admin authorization.
- [x] Browser same-origin mutation protection.
- [x] Protected station-identity commit.
- [x] Full Pi 5 identity commit / revision rotation / previous-snapshot recovery validation.
- [x] Backend-neutral network candidate model.
- [x] BrandMeister local validation and default-port normalization.
- [x] Admin-protected `POST /api/v1/setup/network/validate`.
- [x] Password-redacted network validation response.
- [x] Backend-neutral connectivity-test result/reason interface.
- [x] Automated network candidate/API tests on `dev`.
- [ ] Installed Pi validation of protected network candidate validation.
- [ ] Real BrandMeister/Homebrew connectivity/authentication tester.
- [ ] Protected network test API.
- [ ] Known-good schema migration for network configuration.
- [ ] Tested protected network commit API.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each operational/security slice has automated coverage and relevant real-hardware validation. `main` remains the last promoted known-good milestone.
