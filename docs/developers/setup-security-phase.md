# Setup and Security Phase

This phase turns the tested YWD-DMR appliance into something a normal operator can safely configure from a browser before DMR networking and audio are enabled.

## Goal

The phase is complete when a fresh installation can be claimed once, an administrator can enter and validate the station identity, settings can be tested before they replace known-good configuration, and the daemon can safely expose protected setup/control operations to the WebUI and future Android client.

This work intentionally comes before BrandMeister connection code. Network credentials, DMR identity, audio choices, and vocoder settings all need one authoritative configuration/security model rather than separate ad-hoc files or browser-only state.

## Implementation order

### 1. Radio identity model and validation

First-run setup collects three pieces of radio identity:

- **Callsign** — the operator/station's base amateur-radio callsign.
- **DMR ID** — the base numeric DMR ID, kept separate from any hotspot suffix.
- **ESSID** — a number from 0 through 99 used by networks/backends that need a distinct hotspot/device identity.

The daemon owns normalization and validation. Clients may perform friendly local checks, but server validation is authoritative.

Initial API:

```text
POST /api/v1/setup/identity/validate
```

This endpoint is deliberately non-mutating. It does not save configuration, create an account, connect to a DMR network, or transmit anything.

The first slice was exercised on the installed Raspberry Pi 5 appliance. A valid lowercase/space-padded callsign normalized correctly, invalid callsign/DMR-ID/ESSID values returned three field errors, an unknown JSON field returned HTTP 400, and GET against the POST-only endpoint returned HTTP 405 with `Allow: POST`.

### 2. Known-good configuration store

The daemon has a small durable configuration store with operations equivalent to:

```text
load current known-good
validate candidate
commit candidate
recover from previous snapshot
```

The persistent store belongs under `/var/lib/ywd-dmr/`, which is writable by the restricted `ywd-dmr` service account. The protected listener/service environment under `/etc/ywd-dmr/` remains separate.

The first implementation deliberately uses atomic JSON snapshots and only the Go standard library. It keeps:

```text
/var/lib/ywd-dmr/known-good.json
/var/lib/ywd-dmr/known-good.previous.json
```

A candidate is normalized and validated before any durable state changes. On later commits, the previous known-good value is written to the rollback snapshot before the current snapshot is atomically replaced. If the current snapshot is unreadable or invalid, load may recover from a valid previous snapshot and explicitly report that recovery state.

See [Known-good Configuration Store](configuration-store.md) for the schema, atomic-write rules, rollback behavior, and security boundary.

The configuration-store unit suite, full Go suite, vet, and normal-user build passed on the Raspberry Pi 5 at the `2a889bb` development checkpoint.

### 3. Daemon-owned setup status

The daemon loads the known-good store at startup and records only the minimum setup metadata needed by clients. It exposes:

```text
GET /api/v1/setup/status
```

The public setup status never returns the stored callsign/DMR ID/ESSID or any future secret. It reports whether configuration is missing, loaded, recovered from the previous snapshot, or unreadable; whether identity is configured; and the current daemon-owned revision when available.

A startup recovery from `known-good.previous.json` is visible as `configuration.state: recovered` and is also written to the daemon log as a warning. A hard load failure is exposed only as `configuration.state: error`; detailed filesystem/decoder errors remain server-side.

This runtime path was exercised on the installed Raspberry Pi 5. The daemon correctly reported `missing`, then `loaded` after a valid fixture was installed, then `recovered` after the current snapshot was deliberately corrupted while a valid previous snapshot remained. The service journal emitted the expected recovery warning. Removing both fixtures and restarting returned the daemon to `missing`, and final health diagnostics passed.

### 4. One-time installation claim

A fresh installation has a real bootstrap security state.

When no valid `/var/lib/ywd-dmr/security.json` exists, the daemon creates a 120-bit one-time claim code in:

```text
/var/lib/ywd-dmr/claim-code
```

The file is mode `0600` inside the restricted state directory and is intended to be retrieved locally with:

```bash
sudo ywd-dmr claim-code
```

The claim API is:

```text
POST /api/v1/setup/claim
```

The request contains the claim code plus the first administrator username/password. The code is normalized for copy/paste convenience but compared in constant time. A wrong code does not change state. A successful claim creates the durable administrator password verifier, burns the bootstrap path, and creates an opaque in-memory browser session.

Installed-appliance validation on the Raspberry Pi 5 is complete. The first Pi 5 claim/PBKDF2 timing measurement was `0.516708s`.

See [One-time Claim Validation Notes](claim-validation-notes.md) for the exact real-machine results.

### 5. Administrator login

Normal administrator password login after daemon restart is implemented and real-machine validated.

Implemented endpoints:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username and wrong password use the same generic failure and both execute the password KDF. Login sessions and throttling remain memory-only, so daemon restart clears them while durable administrator state survives.

Installed Pi 5 validation passed. Wrong username and wrong password took `0.520663s` and `0.520534s`; valid login took `0.519940s`. Logout invalidated the token. Five failures returned `401`, the sixth returned `429` with `Retry-After: 59`, and restart cleared the throttle while preserving durable admin state.

See [Administrator Authentication Validation Notes](auth-validation-notes.md) for the exact real-machine results.

### 6. Roles and browser mutation protection

The server-side role hierarchy is:

```text
Observer < Operator < Admin
```

Unknown roles fail closed. Protected handlers require a live session, enforce a minimum role, and receive the authenticated principal through request context.

State-changing browser requests (`POST`, `PUT`, `PATCH`, `DELETE`) pass through global same-origin protection. `Origin`, when present, must match the request origin; `Sec-Fetch-Site`, when present, must be `same-origin` or `none`. Direct API clients remain usable because they do not normally send browser-origin headers.

This foundation is now installed-appliance validated on the Raspberry Pi 5. The runtime sequence was:

```text
direct API mutation        200
same-origin browser        200
cross-origin Origin        403
cross-site fetch metadata  403
same-site/different-origin 403
cross-origin read-only GET 200
```

The unclaimed bootstrap code remained intact and final health passed.

See [Authorization and Browser Mutation Protection](authorization-model.md) and [Authorization Validation Notes](authorization-validation-notes.md).

### 7. Protected station-identity commit

The first normal protected configuration mutation is implemented on `dev`:

```text
POST /api/v1/setup/identity/commit
```

It requires an Admin session and also passes through the global browser same-origin protection.

The endpoint does not invent a second setup/configuration path. It submits the request to the existing known-good store as an untrusted candidate. Identity has no external service to probe, so this slice uses:

```text
candidate -> normalize/validate -> durable commit
```

A successful commit returns the normalized identity and daemon-owned revision, then advances runtime setup state only after durable storage succeeds:

```text
claimed / identity incomplete
-> identity complete / next step network
```

A second valid commit increments the revision and lets the store rotate the old current snapshot to `known-good.previous.json`. Invalid candidates, unauthenticated requests, and rejected cross-origin requests must not replace known-good state or advance runtime setup state.

Automated API tests cover missing-session rejection, normalization/persistence, runtime stage advancement, invalid-candidate preservation, second-commit revision increment, and cross-origin rejection before mutation.

Installed Raspberry Pi validation is the next gate for this slice. It should prove the actual file modes/owners, revision 1 -> revision 2 behavior, restart persistence, previous-snapshot rotation, invalid-candidate protection, and final cleanup/health.

### 8. BrandMeister configuration

Only after the protected configuration contract is proven should the first network backend accept DMR identity, master selection, credentials, reconnect policy, and destination/talkgroup configuration.

Network settings will use the fuller transaction:

```text
candidate -> validate -> connectivity test -> commit
```

A failed BrandMeister test must not destroy the last known-good configuration.

### 9. Guided WebUI wizard

The WebUI becomes a client of the same setup APIs used by future Android/CLI clients. It should use plain-language pages, explain DMR terms in context, and offer useful retry/troubleshooting guidance when a test fails.

## Security boundary during development

The current LAN test dashboard is still not production-ready. Do **not** router-forward or publicly expose it.

The public identity-validation endpoint is non-mutating. `POST /api/v1/setup/claim` remains the one deliberate unauthenticated setup mutation and requires the locally retrieved high-entropy bootstrap code.

Normal login/logout, role authorization, and browser-origin filtering exist. The new identity-commit endpoint is Admin-only. BrandMeister/network controls, radio controls, PTT, HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, and device credentials remain separate work.

## Current implementation status

- [x] Radio identity input/normalized model.
- [x] Callsign, DMR ID, and ESSID server-side validation.
- [x] `POST /api/v1/setup/identity/validate`.
- [x] Installed Pi 5 validation of identity/API behavior.
- [x] Durable known-good configuration repository implementation.
- [x] Atomic persistent storage and rollback/recovery unit tests.
- [x] Configuration-store test suite, full Go suite, vet, and build passed on Pi 5.
- [x] Daemon-owned setup-state model and startup load/recovery state.
- [x] Installed Pi 5 runtime exercise of missing/loaded/recovered/missing configuration state.
- [x] One-time claim implementation and installed-appliance validation.
- [x] Administrator password login/logout/throttling and installed-appliance validation.
- [x] Observer / Operator / Admin role hierarchy and reusable authorization middleware.
- [x] Browser Origin / `Sec-Fetch-Site` mutation protection.
- [x] Installed-appliance validation of role/origin security foundation.
- [x] Admin-protected `POST /api/v1/setup/identity/commit` implementation.
- [x] Automated protected identity commit tests.
- [ ] Installed-appliance validation of protected identity commit and rollback rotation.
- [ ] BrandMeister candidate/test/commit configuration API.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each slice has automated tests and the new operational/security behavior has been exercised on real hardware where relevant. `main` remains the last promoted known-good milestone.
