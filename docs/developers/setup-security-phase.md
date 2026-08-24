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

### 2. Known-good configuration store

Add a daemon-owned configuration repository with explicit operations equivalent to:

```text
load current known-good
validate candidate
test candidate
commit candidate
rollback/reject candidate
```

The persistent store belongs under `/var/lib/ywd-dmr/`, which is writable by the restricted `ywd-dmr` service account. The protected listener/service environment under `/etc/ywd-dmr/` remains separate.

The storage implementation must be suitable for the Raspberry Pi Zero / ARMv6 baseline. A storage dependency is not accepted merely because it works on a Pi 5; build size, memory use, startup cost, cross-compilation, and recovery behavior must be considered.

Secrets must never be returned by a normal configuration read API after storage.

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

The initial identity-validation endpoint is temporarily callable without authentication because it only normalizes user-supplied non-secret data and changes no state. Any endpoint that stores configuration, creates sessions/accounts, reveals protected information, or controls radio/network state must wait for the appropriate claim/auth boundary.

## First implementation slice

- [x] Radio identity input/normalized model.
- [x] Callsign, DMR ID, and ESSID server-side validation.
- [x] `POST /api/v1/setup/identity/validate`.
- [x] Unit/API tests for normalization, invalid fields, JSON contract, and method restrictions.
- [ ] Durable known-good configuration repository interface.
- [ ] Persistent storage implementation and recovery tests.
- [ ] Setup-state model.
- [ ] One-time claim flow.
- [ ] Administrator credential/session implementation.
- [ ] Observer / Operator / Admin middleware and authorization tests.
- [ ] Configuration validate/test/commit API.
- [ ] WebUI first-run wizard.

## Promotion rule

Keep this work on `dev` until each slice has automated tests and the new operational/security behavior has been exercised on real hardware where relevant. `main` remains the last promoted known-good milestone.
