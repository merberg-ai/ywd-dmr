# Setup and Security Phase

This phase turns the tested YWD-DMR appliance into something a normal operator can safely configure from a browser before long-lived DMR networking and audio are enabled.

## Goal

The phase is complete when a fresh installation can be claimed once, an administrator can enter and validate station identity, settings can be tested before they replace known-good configuration, and protected setup/control operations are safe for the WebUI and future native clients.

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
  -> protected non-persisting BrandMeister live test
```

The identity exercise proved revision `1 -> 2`, snapshot ownership/mode, invalid-candidate preservation, restart persistence, and recovery from the API-created previous snapshot.

The BrandMeister candidate-validation and live-test exercises proved that network credentials can be checked without changing the identity-only known-good revision or creating a rollback snapshot.

## 1. Radio identity

First-run setup collects callsign, base DMR ID, and ESSID. Server-side validation and durable identity commit are implemented and Pi 5 validated through:

```text
POST /api/v1/setup/identity/validate
POST /api/v1/setup/identity/commit
```

## 2. Known-good configuration store

The daemon keeps:

```text
/var/lib/ywd-dmr/known-good.json
/var/lib/ywd-dmr/known-good.previous.json
```

The current schema remains identity-only. Network settings and the BrandMeister password are still not persisted.

## 3. Setup status

```text
GET /api/v1/setup/status
```

Configuration health is reported as `missing`, `loaded`, `recovered`, or `error`. Recovery from the previous snapshot is explicit in both API state and the journal.

## 4. Claim, login, roles, browser safety

Implemented and Pi 5 validated:

```text
POST /api/v1/setup/claim
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Role hierarchy:

```text
Observer < Operator < Admin
```

Browser mutations require same-origin metadata when the browser supplies Origin/Sec-Fetch-Site headers. Direct non-browser API clients remain supported.

## 5. Protected identity commit

```text
POST /api/v1/setup/identity/commit
```

Identity has no external service dependency, so its transaction is:

```text
candidate -> normalize/validate -> durable commit
```

The installed Pi exercise proved revision rotation, restart persistence, invalid-candidate preservation, and real previous-snapshot recovery.

## 6. BrandMeister network transaction

Network configuration has an external dependency and therefore uses:

```text
candidate
  -> local validation
  -> real connectivity/authentication/configuration test
  -> durable commit
```

Local validation alone is never called a successful network test.

Current candidate:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "registration_frequency_hz": 446525000,
  "password": "your hotspot password"
}
```

Callsign/DMR ID/ESSID are not duplicated; the backend receives the already known-good station identity.

`registration_frequency_hz` is explicit Homebrew/BrandMeister registration metadata. For the current simplex/software endpoint the same value is reported as RX and TX. It does not add an RF transmit path.

## 7. Protected local network validation

```text
POST /api/v1/setup/network/validate
```

Admin-only and browser same-origin protected. The response never echoes the password; it returns normalized non-secret fields plus `password_set`.

The initial installed Pi validation proved authorization/origin handling, normalization, strict JSON/method behavior, password redaction, and byte-for-byte known-good preservation.

The request now also validates `registration_frequency_hz` before a live test can run.

## 8. Real BrandMeister connectivity test

```text
POST /api/v1/setup/network/test
```

Admin-only, same-origin protected, and deliberately non-persisting. It requires an already committed station identity and locally valid candidate.

The bounded wire probe performs:

```text
RPTL -> RPTACK + salt
RPTK -> RPTACK
RPTC -> RPTACK
RPTCL
```

It has no `DMRD` transmit path and sends no DMR voice/data during setup testing.

Machine-readable results:

```text
ok
login
auth
config
timeout
network
unavailable
```

## 9. Installed live BrandMeister results

The live-test infrastructure itself is Pi 5 proven:

- missing identity fails closed with HTTP `409`;
- cross-origin request fails before UDP;
- a reserved `.invalid` master produces `reason: network`;
- failed live tests leave known-good revision 1 unchanged;
- failed live tests create no rollback snapshot;
- cleanup returns the test box to fresh unclaimed/missing state.

The real master test used base DMR ID `3196104`, ESSID `02`, and derived Homebrew device ID `319610402`.

The first credential attempt produced `reason: auth`. After the operator supplied the verified BrandMeister Hotspot Security password, the next attempt produced:

```text
reason: config
```

That proves the real master accepted:

```text
RPTL device-ID login       PASS
RPTACK challenge           PASS
RPTK password response     PASS
RPTACK authentication      PASS
RPTC zero-frequency config REJECTED
```

The RPTK construction was independently cross-checked against current G4KLX DMRGateway behavior and matches the established `SHA256(salt || password)` protocol.

## 10. RPTC registration-frequency correction

The rejected RPTC had deliberately reported zero RX frequency, zero TX frequency, and zero power because YWD-DMR has no RF transmitter.

BrandMeister hotspot guidance expects valid frequency metadata for Homebrew registration. YWD-DMR now keeps the no-RF architecture honest by asking the operator for an explicit nominal `registration_frequency_hz` instead of silently inventing one.

The current test RPTC uses:

- operator registration frequency in RX;
- the same operator registration frequency in TX;
- informational power `01`;
- color code `01`;
- slot/mode `4` for simplex/DMO;
- zero coordinates/height until real location settings exist;
- YWD-DMR software/client labels.

The informational power is protocol metadata, not an RF-power claim.

Automated tests cover the registration-frequency validation, RPTC RX/TX bytes, informational power value, password secrecy, protocol framing, auth rejection, timeout, and HTTP non-persistence rules.

See [BrandMeister Live Test Notes](brandmeister-live-test-notes.md) and [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md).

## 11. Durable BrandMeister network commit

The commit remains intentionally closed. The next sequence is:

1. source/build/install gate for the registration-frequency RPTC update;
2. focused real BrandMeister retry;
3. require complete `ok` before persistence work;
4. design an explicit known-good schema/migration for network settings/secrets;
5. add protected network commit through the same current/previous snapshot transaction;
6. then build the long-lived BrandMeister connection/reconnect state machine.

## 12. Guided WebUI wizard

The WebUI will use the same setup APIs as future Android/CLI clients, with plain-language explanations for DMR ID/ESSID, master server, registration frequency, Hotspot Security password, and structured failure reasons.

## Security boundary during development

The current LAN test dashboard remains development-only. Do **not** router-forward or publicly expose it.

There is still no durable network commit, long-lived BrandMeister session, radio control, or PTT endpoint. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, and device credentials are separate work.

## Current implementation status

- [x] Radio identity validation and public validation API.
- [x] Known-good configuration store and startup recovery model.
- [x] One-time claim and durable first Admin.
- [x] Password login/logout/throttling.
- [x] Observer / Operator / Admin authorization.
- [x] Browser same-origin mutation protection.
- [x] Protected station-identity commit.
- [x] Full Pi 5 identity revision/rotation/recovery validation.
- [x] Backend-neutral network candidate model.
- [x] BrandMeister local validation and default-port normalization.
- [x] Admin-protected network validation with password redaction.
- [x] Installed Pi network-candidate validation.
- [x] Real bounded Homebrew connectivity/authentication tester.
- [x] Protected non-persisting network test API.
- [x] Installed Pi live-test safety/non-persistence validation.
- [x] Real BrandMeister device-ID login accepted.
- [x] Real BrandMeister Hotspot Security authentication accepted.
- [x] Zero-frequency RPTC rejection isolated.
- [x] Operator registration-frequency metadata added to RPTC candidate/test path.
- [ ] Installed Pi validation of the RPTC registration-frequency update.
- [ ] Complete real BrandMeister `ok` handshake.
- [ ] Known-good schema migration for network configuration.
- [ ] Tested protected network commit API.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each operational/security slice has automated coverage and relevant real-hardware validation. `main` remains the last promoted known-good milestone.
