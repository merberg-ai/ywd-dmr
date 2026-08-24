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
- [ ] Authentication and one-time first-run claim
- [ ] Admin / Operator / Observer authorization model
- [ ] Structured logging and support bundle
- [ ] SQLite event/history persistence where relational storage is justified

## DMR/network

- [ ] Network-backend interface
- [ ] BrandMeister backend
- [ ] DMR ID / ESSID / master configuration
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

Before DMR/network work grew, the `dev` appliance workflow was exercised on real Linux hosts:

1. [x] fresh install on the default port 8989;
2. [x] occupied-port behavior for preserved configuration;
3. [x] service restart and boot persistence on a fresh Raspberry Pi 5 ARM64 host;
4. [x] branded WebUI and versioned documentation delivery;
5. [x] UFW-active LAN install with an installer-created managed rule;
6. [x] UFW-active LAN install with an equivalent pre-existing rule that remains user-owned;
7. [x] normal uninstall while preserving configuration/data;
8. [x] reinstall using preserved configuration;
9. [x] full purge with safety backup and post-purge verification;
10. [x] clean reinstall after purge, including fresh-install free-port suggestion when default port 8989 is occupied.

### Real-machine results

On an Ubuntu host with active UFW and existing ham-radio/network services, YWD-DMR successfully:

- installed on `0.0.0.0:8989` without disturbing other listeners;
- started and remained healthy under systemd;
- served the branded WebUI to another device on the LAN;
- passed installed-appliance verification;
- completed software-only uninstall while preserving persistent configuration/data;
- reinstalled from the preserved state;
- detected an equivalent UFW LAN rule as user-owned and used it without claiming ownership;
- created, verified, diagnosed, and later removed exactly its own tagged managed UFW rule;
- preserved unrelated firewall rules and services;
- passed authoritative software-only and purge uninstall verification;
- refused installation when a preserved configured port belonged to another process;
- deliberately moved the listener to another port and preserved that choice across reinstall.

On a Raspberry Pi 5 running a freshly installed ARM64 OS, YWD-DMR setup and installation completed from a clean machine. A real reboot returned `ywd-dmrd.service` enabled and active with the listener and health API intact. The same Pi completed a full purge with a protected external safety archive, passed the purge verifier, then completed a genuinely clean reinstall while default port 8989 was intentionally occupied; the installer selected free port 8990 without disturbing the unrelated listener.

The installer-created-rule tests exposed and fixed a UFW command-grammar compatibility issue. Final pre-promotion testing also exposed and fixed root-owned checkout build artifacts left by a sudo installer build. After that fix, a sudo reinstall returned `dist/` ownership to the normal user and an immediate normal-user build succeeded.

The final promotion gate passed shell syntax, maintenance CLI regression, managed-UFW grammar regression, `go test ./...`, `go vet ./...`, normal-user build, and `git diff --check`. GitHub CI also passed on the promotion PR.

The tested appliance foundation was promoted from `dev` to `main` through PR #2. `dev` was then fast-forwarded to that merge commit before new Alpha1 development resumed.

## Current Alpha1 focus — setup, security, and configuration

The active work establishes one authoritative setup/configuration/security contract before BrandMeister and audio work expands. See [Setup and Security Phase](setup-security-phase.md) and [Known-good Configuration Store](configuration-store.md).

Current progress:

- [x] Radio identity input and normalized model.
- [x] Callsign, base DMR ID, and ESSID validation.
- [x] `POST /api/v1/setup/identity/validate` as a non-mutating API.
- [x] Unit/API coverage for normalization, invalid fields, JSON contract, and method restrictions.
- [x] Installed Pi 5 exercise of valid/invalid identity requests, strict JSON handling, and POST-only method enforcement.
- [x] Durable known-good configuration file-store implementation.
- [x] Schema/revision envelope, atomic writes, one previous rollback snapshot, invalid-candidate protection, and recovery tests.
- [x] Config-store unit suite, full Go suite, vet, and build passed on Pi 5 at the `2a889bb` checkpoint.
- [x] Daemon startup loads known-good configuration and records missing/loaded/recovered/error state.
- [x] Read-only `GET /api/v1/setup/status` with method/API tests.
- [ ] Pi 5 installed-runtime exercise of missing/loaded/recovered setup status.
- [ ] One-time installation claim.
- [ ] Administrator authentication/session handling.
- [ ] Observer / Operator / Admin server-side authorization.
- [ ] Protected configuration validate/test/commit workflow.
- [ ] Guided WebUI first-run wizard.

The current LAN test dashboard remains unauthenticated. Until claim/authentication is implemented, do not router-forward or publicly expose it. The identity-validation endpoint is intentionally allowed before authentication only because it stores nothing, reveals no protected data, and controls no radio/network state. The setup-status endpoint is also read-only and exposes only coarse configuration health metadata; it does not return the stored identity or secrets. The configuration file store is still not reachable through a mutating HTTP endpoint.
