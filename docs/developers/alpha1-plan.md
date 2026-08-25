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
- [x] Authentication and one-time first-run claim — claim plus normal password login/logout/throttling are real-machine validated on Pi 5
- [x] Admin / Operator / Observer authorization model and browser mutation protection — implementation, automated tests, and Pi 5 runtime validation complete
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

On an Ubuntu host with active UFW and existing ham-radio/network services, YWD-DMR successfully installed, survived service/reinstall/uninstall cycles, preserved unrelated firewall/services, and passed the final promotion gate.

On a fresh Raspberry Pi 5 ARM64 host, the service survived reboot, full purge, protected safety backup, and a genuinely clean reinstall that selected port 8990 while the default port was intentionally occupied.

The tested appliance foundation was promoted from `dev` to `main` through PR #2. `main` remains that known-good milestone while Phase 2 security/configuration work continues on `dev`.

## Current Alpha1 focus — setup, security, and configuration

The active work establishes one authoritative setup/configuration/security contract before BrandMeister and audio work expands. See [Setup and Security Phase](setup-security-phase.md), [Known-good Configuration Store](configuration-store.md), and [Authorization and Browser Mutation Protection](authorization-model.md).

Current progress:

- [x] Radio identity input/normalized model and public non-mutating validation API.
- [x] Known-good file store with schema/revision, atomic writes, rollback snapshot, invalid-candidate protection, and recovery.
- [x] Daemon-owned setup status and Pi 5 missing/loaded/recovered/missing runtime validation.
- [x] One-time claim and first-admin password verifier with full Pi 5 installed validation.
- [x] Password login/logout, generic failures, memory-only throttle/session model, and full Pi 5 installed validation.
- [x] Observer / Operator / Admin hierarchy and fail-closed role comparison.
- [x] Reusable cookie-session authorization middleware with authenticated-principal request context.
- [x] Browser Origin / `Sec-Fetch-Site` protection for state-changing methods while preserving direct non-browser clients.
- [x] Pi 5 installed validation of direct/same-origin acceptance, cross-origin/cross-site rejection, read-only GET compatibility, bootstrap preservation, and final health.
- [x] First Admin-protected configuration mutation implemented: `POST /api/v1/setup/identity/commit`.
- [x] Automated identity-commit tests cover missing-session rejection, normalized durable commit, invalid-candidate preservation, revision increment, setup-state advancement, and cross-origin rejection before mutation.
- [ ] Pi 5 installed validation of identity commit, revision 1 -> 2, previous-snapshot rotation, restart persistence, invalid-candidate preservation, and cleanup.
- [ ] BrandMeister candidate -> validate -> connectivity test -> commit workflow.
- [ ] Guided WebUI first-run wizard.

The current LAN test dashboard remains development-only. Do not router-forward or publicly expose it. Claim, login, role authorization, and browser-origin protection are now validated; the active gate is the first real Admin-protected known-good configuration commit.
