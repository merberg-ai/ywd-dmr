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

## Current install-test focus

Before DMR/network work grows, exercise the `dev` appliance workflow on a real Linux host:

1. [x] fresh install on the default port 8989;
2. [ ] occupied-port behavior;
3. [ ] service restart and boot behavior;
4. [x] branded WebUI and versioned documentation delivery;
5. [x] UFW-active LAN install with an installer-created managed rule;
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
- reported firewall ownership correctly through verification and diagnostics;
- created a tagged `YWD-DMR managed LAN` UFW rule for `192.168.1.0/24 -> 8989/tcp` when no equivalent rule existed;
- verified the installer-created rule as YWD-DMR-owned and reported it correctly through diagnostics;
- removed exactly the installer-owned managed UFW rule during normal uninstall while leaving unrelated UFW rules untouched;
- removed firewall ownership metadata while preserving `/etc/ywd-dmr`, `/var/lib/ywd-dmr`, `/var/log/ywd-dmr`, `/var/backups/ywd-dmr`, and the restricted service account;
- passed the authoritative post-uninstall verifier with the application tree, systemd unit, maintenance CLI, installed uninstaller, firewall metadata, and managed UFW rule all absent while persistent data and the service account remained intact.

The installer-created-rule test exposed a UFW compatibility bug: the first implementation incorrectly used `--force` with a full `allow`/`delete allow` rule specification. The daemon remained healthy and verification correctly reported the firewall setup failure. The UFW commands were corrected to use normal full-rule syntax without `--force`, then the managed-rule installation and cleanup paths passed on the real host.

A later test used `command -v ywd-dmr` immediately after uninstall and reported a possible remaining CLI. Because Bash can retain the old command path in its per-shell hash table even after `/usr/local/bin/ywd-dmr` is deleted, post-uninstall verification now checks the physical path directly and documents `hash -r` for clearing the interactive shell cache. The physical-path check and authoritative uninstall verifier both passed.

Remaining appliance tests are occupied-port behavior, restart/boot persistence, full purge with safety backup, and a clean reinstall after purge.

Problems found here should be fixed before promoting this installer foundation to `main`.
