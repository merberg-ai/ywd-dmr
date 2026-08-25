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
- [ ] Admin / Operator / Observer authorization model — implementation and automated tests are on `dev`; installed validation remains
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

The active work establishes one authoritative setup/configuration/security contract before BrandMeister and audio work expands. See [Setup and Security Phase](setup-security-phase.md), [Known-good Configuration Store](configuration-store.md), and [Authorization and Browser Mutation Protection](authorization-model.md).

Current progress:

- [x] Radio identity input and normalized model.
- [x] Callsign, base DMR ID, and ESSID validation.
- [x] `POST /api/v1/setup/identity/validate` as a non-mutating API.
- [x] Installed Pi 5 exercise of identity validation and strict HTTP/JSON behavior.
- [x] Durable known-good configuration file-store implementation with schema/revision, atomic writes, rollback snapshot, invalid-candidate protection, and recovery tests.
- [x] Config-store unit suite, full Go suite, vet, and build passed on Pi 5 at the `2a889bb` checkpoint.
- [x] Daemon startup loads known-good configuration and records missing/loaded/recovered/error state.
- [x] Read-only `GET /api/v1/setup/status` with method/API tests.
- [x] Pi 5 installed-runtime exercise of missing -> loaded -> recovered -> missing setup status, including the expected recovery journal warning and final healthy daemon.
- [x] One-time high-entropy claim-code generation in restricted persistent state.
- [x] Local `sudo ywd-dmr claim-code` retrieval path.
- [x] First-admin password verifier format using standard-library PBKDF2-HMAC-SHA256 with random salt and stored iteration count.
- [x] `POST /api/v1/setup/claim` with one-time semantics and opaque HttpOnly `SameSite=Strict` session cookie.
- [x] `GET /api/v1/auth/session` current-session inspection.
- [x] Pi 5 installed-appliance exercise of wrong-code rejection, successful claim, claim-code deletion, one-time reuse rejection, claimed restart persistence, memory-only session invalidation on restart, no code regeneration while claimed, cleanup, and final health.
- [x] First Pi 5 PBKDF2/claim timing captured: `0.516708s` at 310000 iterations.
- [x] `POST /api/v1/auth/login` implementation using the persisted password verifier and a fresh opaque in-memory session.
- [x] Generic wrong-username/wrong-password authentication failure behavior.
- [x] In-memory direct-client login throttle: five failures in five minutes -> 60-second block.
- [x] `POST /api/v1/auth/logout` session invalidation and cookie expiration.
- [x] Automated login/logout/throttle tests.
- [x] Pi 5 installed-appliance validation of login after restart, generic failures, throttling, logout, restart-cleared sessions/throttle, cleanup, and final health.
- [x] Pi 5 login timings captured: wrong username `0.520663s`, wrong password `0.520534s`, valid login `0.519940s`, post-restart valid login `0.517669s`.
- [x] Observer / Operator / Admin role hierarchy and fail-closed role comparison implemented.
- [x] Reusable cookie-session authorization middleware implemented with authenticated-principal request context.
- [x] Browser Origin / `Sec-Fetch-Site` protection implemented for state-changing methods, while preserving direct non-browser API clients.
- [x] Automated authorization and browser-origin tests added on `dev`.
- [ ] Pi 5 installed-appliance validation of the role/origin security slice.
- [ ] Protected configuration validate/test/commit workflow.
- [ ] Guided WebUI first-run wizard.

The current LAN test dashboard remains development-only. Do not router-forward or publicly expose it. One-time claim and normal login are validated; role/origin middleware is the active security gate before configuration commits, radio/network controls, or secret reads are opened.
