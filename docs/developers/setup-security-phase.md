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

The daemon now has a small durable configuration store with operations equivalent to:

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

The storage implementation remains suitable for the Raspberry Pi Zero / ARMv6 baseline: no database runtime is required for this small settings document, and the implementation cross-compiles with the rest of the standard-library-only daemon.

Secrets must never be returned by a normal configuration read API after storage. No BrandMeister or administrator secret is stored by this slice.

### 3. One-time claim and administrator authentication

A fresh installation starts unclaimed.

The claim flow must:

1. use a one-time setup secret/code;
2. allow creation of the first administrator without a default password;
3. invalidate the claim secret after successful use;
4. create an opaque browser session rather than exposing reusable credentials to JavaScript;
5. enforce authorization in the daemon, not by hiding WebUI buttons.

The roles remain:

- **Observer** — view status/history but cannot change radio state;
- **Operator** — normal radio operation, including authorized PTT later;
- **Admin** — configuration, account/device management, updates, and maintenance controls.

PTT will still require a separate renewable TX lease and timeout even for an authenticated Operator/Admin.

### 4. Setup status and configuration commit APIs

Once storage and claim/auth exist, add setup endpoints that let clients determine which step is required and submit validated candidates.

The API should report state such as:

```text
unclaimed
claimed / identity incomplete
identity complete / network incomplete
network configured / audio incomplete
ready
```

These are daemon states, not WebUI guesses.

### 5. BrandMeister configuration

Only after the configuration/security contract exists should the first network backend accept DMR identity, master selection, credentials, reconnect policy, and destination/talkgroup configuration.

Network settings follow validate -> test -> commit. A failed BrandMeister test must not destroy the last known-good configuration.

### 6. Guided WebUI wizard

The WebUI then becomes a client of the same setup APIs used by future Android/CLI clients. It should use plain-language pages, explain DMR terms in context, and offer useful retry/troubleshooting guidance when a test fails.

## Security boundary during development

The current LAN test dashboard remains unauthenticated. Do **not** router-forward or publicly expose it.

The identity-validation endpoint is temporarily callable without authentication because it only normalizes user-supplied non-secret data and changes no state. The new file store is not exposed through a mutating HTTP endpoint yet. Any endpoint that stores configuration, creates sessions/accounts, reveals protected information, or controls radio/network state must wait for the appropriate claim/auth boundary.

## Current implementation status

- [x] Radio identity input/normalized model.
- [x] Callsign, DMR ID, and ESSID server-side validation.
- [x] `POST /api/v1/setup/identity/validate`.
- [x] Unit/API tests for normalization, invalid fields, JSON contract, and method restrictions.
- [x] Installed Pi 5 validation of the identity endpoint and HTTP behavior.
- [x] Durable known-good configuration repository implementation.
- [x] Atomic persistent storage and rollback/recovery unit tests.
- [ ] Installed-appliance exercise of the configuration store once daemon wiring exists.
- [ ] Setup-state model.
- [ ] One-time claim flow.
- [ ] Administrator credential/session implementation.
- [ ] Observer / Operator / Admin middleware and authorization tests.
- [ ] Configuration validate/test/commit API.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each slice has automated tests and the new operational/security behavior has been exercised on real hardware where relevant. `main` remains the last promoted known-good milestone.
