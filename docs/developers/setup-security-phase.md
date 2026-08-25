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

The request contains the claim code plus the first administrator username/password. The code is normalized for copy/paste convenience but compared in constant time. A wrong code does not change state. A successful claim:

1. validates the administrator username/password policy;
2. creates a random password salt;
3. derives the stored password verifier with standard-library PBKDF2-HMAC-SHA256;
4. atomically writes `/var/lib/ywd-dmr/security.json` mode `0600`;
5. marks the installation claimed;
6. deletes the plaintext claim-code file on a best-effort basis;
7. creates an opaque in-memory administrator session and sets it only in an HttpOnly `SameSite=Strict` cookie.

The session token is not returned in JSON. On current HTTP LAN-test installs the cookie cannot use the browser `Secure` flag because the connection is not TLS; once normal remote/LAN security moves to HTTPS, authenticated cookies must be Secure as well.

Claim is deliberately one-time. Once the durable security document exists, claim requests return a conflict and daemon startup will never silently regenerate a new claim code. If an existing `security.json` is malformed or unsupported, daemon startup fails closed rather than treating the machine as a new unclaimed appliance.

The stored password record includes its algorithm, salt, and iteration count so the verifier can be upgraded later without guessing which KDF produced an old record. No plaintext administrator password is stored.

Installed-appliance validation on the Raspberry Pi 5 is complete. The successful real claim returned HTTP `201`, created `security.json` as `0600 ywd-dmr:ywd-dmr`, deleted the plaintext claim code, authenticated the new cookie session, rejected claim reuse with `409`, preserved claimed state across daemon restart, discarded the memory-only session on restart, and did not regenerate a claim code on the claimed appliance. Intentional cleanup returned the box to a healthy unclaimed state with a fresh code. The first Pi 5 claim/PBKDF2 timing measurement was `0.516708s`.

See [One-time Claim Validation Notes](claim-validation-notes.md) for the exact real-machine results.

### 5. Administrator login and protected authorization

Normal administrator password login after a daemon restart is now implemented on `dev` and is the active real-machine validation slice.

Implemented endpoints:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

`POST /api/v1/auth/login` verifies the supplied password against the persisted PBKDF2-HMAC-SHA256 record. Username comparison and password verification use the same generic invalid-credential response so the normal failure path does not disclose whether a username or password was wrong. The password KDF is still evaluated when the username is wrong, avoiding a cheap timing oracle.

A successful login creates a fresh opaque session token, stores only its hash in daemon memory, and sets the token only in the HttpOnly `SameSite=Strict` cookie. Restarting the daemon clears all sessions while leaving the durable claimed/admin state intact.

The first login throttle is intentionally small and understandable for an appliance UI:

```text
5 failed logins from one direct client IP inside 5 minutes
-> block further login attempts from that IP for 60 seconds
```

The throttle is memory-only and does not create a persistent lockout database. The daemon currently keys this from the direct TCP peer address and does not trust `X-Forwarded-For`; proxy-aware deployment rules will be defined when HTTPS/reverse-proxy support is introduced.

`POST /api/v1/auth/logout` invalidates the current in-memory session, when present, and expires the browser cookie. Logout is idempotent from the client's point of view.

Automated tests cover restart login from the persisted verifier, fresh-session creation, generic wrong-username/wrong-password behavior, login/logout HTTP behavior, session invalidation, method enforcement, and throttle state transitions. Real installed-appliance validation is still required before this login slice is marked complete.

The next authorization layer must add:

- server-side authorization middleware;
- Observer / Operator / Admin roles;
- origin/CSRF protections for authenticated mutating browser requests.

### 6. Protected configuration commit APIs

Once login/authorization exists, protected endpoints can submit candidate settings through the known-good transaction path.

The setup state progresses through daemon-owned states such as:

```text
unclaimed
claimed / identity incomplete
identity complete / network incomplete
network configured / audio incomplete
ready
```

Configuration changes follow validate -> test -> commit. A failed candidate or later BrandMeister connectivity test must not destroy the last known-good configuration.

### 7. BrandMeister configuration

Only after the configuration/security contract exists should the first network backend accept DMR identity, master selection, credentials, reconnect policy, and destination/talkgroup configuration.

Network settings follow validate -> test -> commit. A failed BrandMeister test must not destroy the last known-good configuration.

### 8. Guided WebUI wizard

The WebUI becomes a client of the same setup APIs used by future Android/CLI clients. It should use plain-language pages, explain DMR terms in context, and offer useful retry/troubleshooting guidance when a test fails.

## Security boundary during development

The current LAN test dashboard is still not a production-authenticated interface. Do **not** router-forward or publicly expose it.

The identity-validation endpoint is callable without authentication because it only normalizes caller-supplied non-secret data and changes no state. `GET /api/v1/setup/status` exposes only coarse setup/configuration health metadata. `POST /api/v1/setup/claim` is the one deliberate unauthenticated setup mutation and requires the high-entropy bootstrap code available only from the local appliance filesystem.

Normal login/logout now exist, but role authorization and origin/CSRF protections are not complete. No configuration commit, network control, radio control, or secret-read endpoint is exposed before that middleware exists.

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
- [x] One-time claim-code generation and fail-closed security initialization.
- [x] First-admin persistent password-verifier format using standard-library PBKDF2-HMAC-SHA256.
- [x] `POST /api/v1/setup/claim` with one-time semantics and opaque HttpOnly session cookie.
- [x] `GET /api/v1/auth/session` for read-only current-session inspection.
- [x] Local `sudo ywd-dmr claim-code` maintenance command.
- [x] Installed-appliance exercise of one-time claim, code deletion, claimed restart, and session behavior.
- [x] Administrator password-login/logout implementation with generic failures and in-memory throttling.
- [ ] Installed-appliance validation of login after restart, wrong credentials, throttle, logout, and restart-cleared sessions.
- [ ] Observer / Operator / Admin middleware and authorization tests.
- [ ] Origin/CSRF protection for authenticated browser mutations.
- [ ] Protected configuration validate/test/commit API.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each slice has automated tests and the new operational/security behavior has been exercised on real hardware where relevant. `main` remains the last promoted known-good milestone.
