# Setup and Security Phase

This phase turns the tested YWD-DMR appliance into something a normal operator can safely configure from a browser before DMR networking and audio are enabled.

## Goal

The phase is complete when a fresh installation can be claimed once, an administrator can enter and validate the station identity, settings can be tested before they replace known-good configuration, and the daemon can safely expose protected setup/control operations to the WebUI and future Android client.

This work intentionally comes before the long-lived BrandMeister connection. Network credentials, DMR identity, audio choices, and vocoder settings all need one authoritative configuration/security model rather than separate ad-hoc files or browser-only state.

## Proven setup/security foundation

The following chain is installed-appliance validated on the Raspberry Pi 5:

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
  -> protected BrandMeister candidate validation
```

The identity exercise proved revision `1 -> 2`, `0600 ywd-dmr:ywd-dmr` current/previous snapshots, invalid-candidate preservation, restart persistence, and real recovery from the API-created previous snapshot after deliberately corrupting current.

The protected network-validation exercise then proved that a BrandMeister candidate can be normalized and password-redacted without changing the identity-only known-good file, revision, rollback snapshot, or setup stage.

See [Protected Station-Identity Commit Validation Notes](identity-commit-validation-notes.md) and [BrandMeister Candidate Validation Notes](network-validation-notes.md).

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

The real protected identity API has proven this same rotation/recovery path on installed storage rather than only through unit fixtures.

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

Implemented and fully installed-machine validated:

```text
POST /api/v1/setup/identity/commit
```

It is Admin-only and same-origin protected for browsers.

Identity has no external service to test, so its transaction is:

```text
candidate -> normalize/validate -> durable commit
```

Runtime setup state advances only after durable commit succeeds. The Pi 5 validation proved revision increment, previous-snapshot rotation, restart persistence, invalid-candidate preservation, and recovery from an API-created previous snapshot.

## 8. BrandMeister/network candidate validation

Network configuration has an external dependency and therefore uses a stricter transaction:

```text
candidate
  -> local validation
  -> real connectivity/authentication test
  -> durable commit
```

Local validation alone is never called a successful network test.

The candidate contains:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your hotspot password"
}
```

Callsign/DMR ID/ESSID are not duplicated here. The backend receives the already known-good station identity.

Protected local validation is:

```text
POST /api/v1/setup/network/validate
```

This endpoint requires Admin authentication and browser same-origin protection. It never echoes the password; the normalized result only reports `password_set: true/false`.

The installed Pi 5 validation passed normalization, all four invalid-field errors, unknown JSON rejection, POST-only method enforcement, unauthenticated/cross-origin rejection, password redaction, and byte-for-byte known-good preservation. Schema 1 remained identity-only.

## 9. Real BrandMeister connectivity test — active slice

The first real but non-persisting network probe is now implemented on `dev`:

```text
POST /api/v1/setup/network/test
```

It requires:

1. a live Admin session;
2. a locally valid BrandMeister candidate;
3. an already committed/readable station identity;
4. the real network tester service.

A missing identity returns HTTP `409`. Invalid fields return HTTP `400`. Cross-origin browser requests are rejected before the tester runs.

The handler gives the test a 10-second overall deadline and returns structured, password-free results:

```text
ok
login
auth
config
timeout
network
unavailable
```

The wire probe is deliberately short-lived:

```text
RPTL -> RPTACK + salt
RPTK -> RPTACK
RPTC -> RPTACK
RPTCL
```

`RPTK` uses SHA-256 of `salt || password` as established by the Homebrew/MMDVM protocol. `RPTC` is the fixed 302-byte configuration packet.

The tester has no DMRD transmit path and therefore cannot send DMR voice/data during setup testing.

### BrandMeister device ID

The backend derives the device ID from the canonical identity:

```text
ESSID 0     -> base DMR ID
ESSID 1..99 -> (base DMR ID * 100) + ESSID
```

The generic identity remains unchanged.

### Software-endpoint configuration

YWD-DMR does not have an RF transmitter in this architecture, so the temporary RPTC probe does not invent RF values. It sends zero RX/TX frequencies and power, zero coordinates/height, and explicit YWD-DMR software/client labels. The slot/mode field is `4` for simplex/DMO-style software endpoint behavior.

The Homebrew callsign field is only eight characters wide. If the stored generic callsign cannot fit, the tester returns `reason: config` instead of silently truncating it.

Whether the real BrandMeister master accepts the zero-RF software endpoint is intentionally part of the next Pi/live test. We will change the protocol representation only based on actual master behavior, not by pretending that software has a radio frequency.

### Automated test coverage

Local UDP test masters prove:

- hotspot-ID derivation;
- `RPTL` packet framing;
- four-byte salt handling;
- `RPTK` password-hash construction;
- fixed 302-byte `RPTC` framing;
- explicit `RPTCL` close;
- auth `MSTNAK -> reason: auth`;
- bounded timeout handling;
- no password in RPTC/results;
- callsign-width failure as `reason: config`.

HTTP tests prove that committed identity is required, invalid/cross-origin candidates never invoke the tester, the tester receives known-good identity, responses do not echo the password, and successful testing does not advance the known-good revision.

See [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md).

## 10. Durable BrandMeister network commit

The network commit remains intentionally closed.

The next sequence after the live tester is proven is:

1. validate the installed test endpoint using local/failure cases;
2. perform one real BrandMeister handshake with operator credentials;
3. prove success/failure leaves current known-good state unchanged;
4. make an explicit known-good schema/migration change for network settings;
5. allow durable network commit only after the candidate passes the real test;
6. build the long-lived BrandMeister connection/reconnect state machine.

The current known-good schema remains identity-only and no BrandMeister password is persisted yet.

## 11. Guided WebUI wizard

The WebUI will use the same setup APIs as future Android/CLI clients, with plain-language DMR explanations and useful troubleshooting for rejected ID, wrong Hotspot Security password, master timeout, and configuration rejection.

## Security boundary during development

The current LAN test dashboard remains development-only. Do **not** router-forward or publicly expose it.

The public identity-validation endpoint is non-mutating. `POST /api/v1/setup/claim` remains the one deliberate unauthenticated setup mutation requiring the local bootstrap code. Login/logout, role authorization, browser-origin filtering, protected identity commit, and protected network candidate validation are installed-machine validated.

The live network test is Admin-only and does not persist the password. There is still no network commit, BrandMeister production session, radio control, or PTT endpoint. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, and device credentials are separate work.

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
- [x] Installed Pi validation of protected network candidate validation.
- [x] Backend-neutral connectivity-test result/reason interface.
- [x] Real bounded BrandMeister/Homebrew connectivity/authentication tester.
- [x] Local UDP Homebrew protocol tests.
- [x] Protected `POST /api/v1/setup/network/test` using the real tester.
- [ ] Pi source/runtime validation of the live test endpoint.
- [ ] Successful real BrandMeister handshake.
- [ ] Known-good schema migration for network configuration.
- [ ] Tested protected network commit API.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each operational/security slice has automated coverage and relevant real-hardware validation. `main` remains the last promoted known-good milestone.
