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
- [x] First Admin-protected known-good configuration mutation and real rollback rotation/recovery validation on Pi 5
- [ ] Structured logging and support bundle
- [ ] SQLite event/history persistence where relational storage is justified

## DMR/network

- [x] Network-backend connectivity-test interface and structured failure reasons
- [x] First BrandMeister candidate model and protected local validation API
- [ ] Real BrandMeister/Homebrew login-auth-config tester
- [ ] Protected network test endpoint
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

The tested appliance foundation was promoted from `dev` to `main` through PR #2. `main` remains that known-good milestone while Alpha1 development continues on `dev`.

## Completed setup/security/configuration chain

The complete first protected setup chain is now proven on the Raspberry Pi 5:

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
```

Real-machine identity-commit validation proved revision `1 -> 2`, correct `0600 ywd-dmr:ywd-dmr` snapshot ownership, byte-for-byte preservation after a rejected invalid candidate, restart loading of revision `2`, and recovery of API-created previous revision `1` after deliberate corruption of the current snapshot. Cleanup returned the box to an unclaimed/missing-config state with a fresh claim code and healthy daemon.

See [Protected Station-Identity Commit Validation Notes](identity-commit-validation-notes.md).

## Current Alpha1 focus — BrandMeister setup contract

BrandMeister work can now build on a proven security/configuration foundation instead of creating its own special-case persistence.

Current progress:

- [x] Backend-neutral network candidate model.
- [x] BrandMeister backend identifier and default UDP port `62031` normalization.
- [x] Protected `POST /api/v1/setup/network/validate` endpoint.
- [x] Password-redacted validation response using `password_set` rather than returning the supplied secret.
- [x] Backend-neutral connectivity-test result interface with structured reasons such as `login`, `auth`, `config`, `timeout`, and `network`.
- [x] Automated network candidate/API tests added on `dev`.
- [ ] Installed Pi validation of protected BrandMeister candidate validation.
- [ ] Real short-lived Homebrew handshake tester.
- [ ] Test endpoint that only reports success after login/auth/config are actually accepted by the master.
- [ ] Network schema migration + tested commit into the same known-good/previous-snapshot transaction.
- [ ] Long-lived BrandMeister session/reconnect state machine.

The active transaction rule is:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

Local validation alone is never allowed to become a fake connectivity success.

See [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md) and [Setup and Security Phase](setup-security-phase.md).

The current LAN test dashboard remains development-only. Do not router-forward or publicly expose it. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, persistent multi-user/device credentials, radio controls, and PTT safety remain later work.
