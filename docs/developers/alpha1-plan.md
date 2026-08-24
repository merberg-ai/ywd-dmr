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
- [ ] Authentication and one-time first-run claim
- [ ] Admin / Operator / Observer authorization model
- [ ] Structured logging and support bundle
- [ ] Configuration schema and known-good transaction model beyond the current listener environment file
- [ ] SQLite event/config persistence

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
- [x] Safe UFW integration for LAN installs without taking ownership of pre-existing rules
- [x] systemd unit and basic maintenance CLI
- [x] CLI safe uninstall workflow with preserve/purge modes
- [x] Install verification and post-start health check
- [ ] Production one-command GitHub-release installer using verified prebuilt packages
- [ ] Guided WebUI first-run wizard
- [ ] mDNS `ywd-dmr.local`
- [ ] WebUI uninstall/repair workflow
- [ ] GitHub release updater
- [ ] protected pre-update backups
- [ ] config migrations
- [ ] production post-update health checkpoint
- [ ] automatic rollback

## Current install-test focus

Before DMR/network work grows, exercise the `dev` appliance workflow on a real Linux host:

1. [x] fresh install on the default port 8989;
2. [ ] occupied-port behavior;
3. [ ] service restart and boot behavior;
4. [x] branded WebUI and versioned documentation delivery;
5. [ ] UFW-active LAN install with an installer-created managed rule;
6. [x] UFW-active LAN install with an equivalent pre-existing rule that remains user-owned;
7. [x] normal uninstall while preserving configuration/data;
8. [x] reinstall using preserved configuration;
9. [ ] full purge with safety backup;
10. [ ] clean reinstall after purge.

### Real-machine results so far

On an Ubuntu host with active UFW and existing ham-radio/network services, YWD-DMR has successfully:

- installed on `0.0.0.0:8989` without disturbing other listeners;
- started and remained healthy under systemd;
- served the branded WebUI to another device on the LAN;
- passed installed-appliance verification;
- completed software-only uninstall while preserving persistent configuration/data;
- reinstalled from the preserved state;
- detected an existing `192.168.1.0/24 -> 8989/tcp` UFW rule as user-owned;
- used that rule without claiming ownership or changing it;
- reported firewall ownership correctly through verification and diagnostics.

The installer-created-rule test exposed a UFW compatibility bug: the first implementation incorrectly used `--force` with a full `allow`/`delete allow` rule specification. The daemon remained healthy and verification correctly reported the firewall setup failure. The UFW commands were corrected to use normal full-rule syntax without `--force`.

The remaining firewall test is the inverse path: let YWD-DMR create its own tagged LAN rule successfully, then confirm normal uninstall removes only that installer-owned rule.

Problems found here should be fixed before promoting this installer foundation to `main`.
