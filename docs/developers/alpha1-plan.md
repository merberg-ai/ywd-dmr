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
- [x] Pi validation of live-test authorization, failure classification, and non-persistence
- [x] Real BrandMeister device-ID login accepted
- [x] Real BrandMeister Hotspot Security authentication accepted
- [x] Zero-frequency RPTC rejection isolated to configuration stage
- [x] Explicit Homebrew registration-frequency metadata added for focused retest
- [ ] Complete real BrandMeister RPTC acceptance / `ok` result
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
  -> protected real BrandMeister test
```

The live test is non-persisting by design. A reserved `.invalid` master produced a structured `network` result while leaving known-good revision 1 byte-for-byte unchanged and creating no rollback snapshot.

## Current Alpha1 focus — finish BrandMeister RPTC registration

The real BrandMeister probe has now reached the public master at `3103.master.brandmeister.network:62031` using the operator's actual DMR identity/ESSID.

The first credential attempt correctly produced `reason: auth`. After the operator supplied the verified Hotspot Security password, the next result was:

```text
reason: config
```

That proves:

```text
RPTL login                PASS
RPTACK challenge          PASS
RPTK Hotspot Security     PASS
RPTACK authentication     PASS
RPTC zero-RF registration REJECTED
```

The auth packet was also cross-checked against current G4KLX DMRGateway behavior and matches the established `SHA256(salt || password)` framing.

BrandMeister hotspot guidance expects real frequency metadata in Homebrew registration. YWD-DMR therefore no longer reports zero RX/TX frequency. The network candidate now includes:

```text
registration_frequency_hz
```

The tester places that operator-supplied value in both RX and TX RPTC fields for simplex/DMO registration and uses informational power `01`. This remains registration metadata only; YWD-DMR still has no RF transmit path.

The next gate is a focused Pi source/build/install test of this RPTC change followed by one real BrandMeister retry. If it returns `ok`, network schema/migration and protected commit work can begin. If it remains `config`, the remaining RPTC fields will be narrowed without weakening the non-persistence or no-RF rules.

See [BrandMeister Live Test Notes](brandmeister-live-test-notes.md), [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md), and [Control API v1](../protocols/control-api-v1.md).

The current LAN test dashboard remains development-only. Do not router-forward or publicly expose it. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, persistent multi-user/device credentials, radio controls, and PTT safety remain later work.
