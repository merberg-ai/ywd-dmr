# 0.1.0-alpha1 — First QSO Plan

The first on-air milestone is successful when a user can install YWD-DMR, finish setup, select BrandMeister Parrot, hold PTT, speak, release PTT, and hear the returned audio without RF hardware.

## Foundation

- [x] Repository and documentation skeleton
- [x] Minimal Go daemon
- [x] Read-only Control API scaffold
- [x] Responsive branded WebUI scaffold
- [x] Pi Zero/ARMv6 CI build target
- [x] Safe uninstall implementation and installation-ownership rules
- [x] Default frontend port 8989 and reusable port preflight helper
- [x] Development appliance installer for repeat install/uninstall testing
- [x] Dedicated non-root service account and hardened systemd service
- [x] Basic maintenance CLI and installed health-verification helper
- [x] UFW LAN detection, LAN-only rule offer, ownership tracking, and safe uninstall cleanup
- [x] Post-uninstall verifier for software-only and purge modes
- [x] Radio identity model and non-mutating setup validation API
- [x] Known-good configuration schema, atomic persistence, and rollback/recovery core
- [x] Daemon-owned read-only setup status
- [x] Authentication and one-time first-run claim — real-machine validated on Pi 5
- [x] Admin / Operator / Observer authorization and browser mutation protection — Pi 5 validated
- [x] First Admin-protected known-good configuration mutation and real rollback rotation/recovery validation on Pi 5
- [ ] Structured logging and support bundle
- [ ] SQLite event/history persistence where relational storage is justified

## DMR/network

- [x] Network-backend connectivity-test interface and structured failure reasons
- [x] First BrandMeister candidate model and protected local validation API
- [x] Pi 5 validation of protected BrandMeister candidate validation and password redaction
- [x] Real bounded BrandMeister/Homebrew login-auth-config tester
- [x] Protected non-persisting network test endpoint
- [ ] Pi validation of the live test endpoint and one real BrandMeister handshake
- [ ] Known-good schema migration for tested network configuration
- [ ] Protected network commit endpoint
- [ ] BrandMeister long-lived backend
- [ ] DMR ID / ESSID / master configuration integration
- [ ] Connection/reconnect state machine
- [ ] Talkgroup destination model
- [ ] Callsign/DMR-ID resolver abstraction

## Vocoder/audio

- [ ] Vocoder API v1 framing implementation
- [ ] Plugin launcher and capability discovery
- [ ] First supported hardware vocoder backend
- [ ] Local audio device enumeration
- [ ] Local 8 kHz mono PCM path
- [ ] Audio self-test
- [ ] No-vocoder mode status and UX

## Clients and safety

- [ ] Event WebSocket
- [ ] Binary Audio Stream v1
- [ ] Browser AudioWorklet client
- [ ] PTT lease manager
- [ ] TX timeout timer
- [ ] Single-TX-owner arbitration
- [ ] RX/TX dashboard state
- [ ] Last Heard basics

## Appliance workflow

- [x] Development installer port-conflict detection, free-port suggestion, and configurable bind/listen settings
- [x] Explicit port override persists across reinstall and keeps firewall integration aligned
- [x] Safe UFW integration for LAN installs without taking ownership of pre-existing rules
- [x] systemd unit and basic maintenance CLI
- [x] CLI safe uninstall workflow with preserve/purge modes
- [x] Install verification and post-start health check
- [x] Post-uninstall verification that checks physical installed paths rather than shell command lookup
- [ ] Production one-command GitHub-release installer using verified prebuilt packages
- [ ] Guided WebUI first-run wizard
- [ ] mDNS `ywd-dmr.local`
- [ ] WebUI uninstall/repair workflow
- [ ] GitHub release updater
- [ ] protected pre-update backups
- [ ] config migrations
- [ ] production post-update health checkpoint
- [ ] automatic rollback

## Completed appliance validation

The tested appliance foundation was promoted from `dev` to `main` through PR #2. `main` remains that known-good milestone while Alpha1 development continues on `dev`.

On the Raspberry Pi 5, the complete protected setup/configuration chain has now proven:

```text
one-time claim
  -> durable Admin credential
  -> memory-only login session
  -> Admin authorization
  -> browser same-origin protection
  -> identity candidate validation
  -> atomic known-good commit
  -> revision rotation
  -> previous-snapshot recovery
  -> protected BrandMeister candidate validation
```

The BrandMeister candidate-validation run used a reserved `.invalid` hostname on purpose. It proved local normalization, default port `62031`, password redaction, invalid-field reporting, authorization/origin protection, strict JSON/method behavior, and that network validation cannot change the identity-only known-good revision or create a rollback snapshot.

See [Protected Station-Identity Commit Validation Notes](identity-commit-validation-notes.md) and [BrandMeister Candidate Validation Notes](network-validation-notes.md).

## Current Alpha1 focus — first real BrandMeister wire test

A real short-lived Homebrew tester is now implemented on `dev`.

It performs only:

```text
RPTL -> RPTACK + salt
RPTK -> RPTACK
RPTC -> RPTACK
RPTCL
```

The password response is SHA-256 of `salt || password`. The test is bounded, sends no `DMRD` voice/data, persists nothing, and returns structured reasons including `login`, `auth`, `config`, `timeout`, `network`, and `unavailable`.

The protected endpoint is:

```text
POST /api/v1/setup/network/test
```

It requires Admin authorization, same-origin browser protection, valid network fields, and an already committed station identity. The tester always reads callsign/DMR ID/ESSID from known-good state; the network request cannot substitute a different identity.

Local UDP protocol tests cover packet framing, hotspot-ID derivation, salt/password hashing, fixed 302-byte software RPTC configuration, explicit close, auth rejection, timeout, and secret handling. HTTP tests cover prerequisites and the rule that a network test cannot advance the known-good revision.

The next gate is the installed Pi source/runtime test followed by one real BrandMeister handshake with the operator's actual master and Hotspot Security password. Even a successful live test will still **not** persist network settings; schema migration and commit remain a separate later gate.

See [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md), [Control API v1](../protocols/control-api-v1.md), and [Setup and Security Phase](setup-security-phase.md).

The current LAN test dashboard remains development-only. Do not router-forward or publicly expose it. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, persistent multi-user/device credentials, radio controls, and PTT safety remain later work.
