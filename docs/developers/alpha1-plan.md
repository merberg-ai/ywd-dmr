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
- [x] Explicit Homebrew registration-frequency metadata added and Pi retested
- [x] Fresh ESSID/device-ID conflict hypothesis isolated
- [x] `MMDVM_DMO` package/profile compatibility isolated
- [x] Real BrandMeister RPTC acceptance / `ok` result
- [x] YWD-DMR-owned numeric/date-style Homebrew software identifier accepted
- [x] Explicit schema-2 design for tested network configuration implemented on `dev`
- [x] Revision-bound restricted Hotspot Security secret storage implemented on `dev`
- [x] Protected one-request network test-and-commit endpoint implemented on `dev`
- [ ] Pi 5 installed validation of schema-2 test-and-commit
- [ ] Pi 5 restart/rotation/recovery validation with matching network secrets
- [ ] BrandMeister long-lived backend
- [ ] Connection/reconnect state machine
- [ ] DMR ID / ESSID / master configuration integration into long-lived runtime
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

- [x] LAN-only development Admin Test Console using the real v1 APIs
- [x] Admin Test Console live BrandMeister test
- [x] Admin Test Console tested network commit control on `dev`
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

On the Raspberry Pi 5, the protected chain has proven:

```text
one-time claim
  -> durable Admin credential
  -> memory-only login session
  -> Admin authorization
  -> browser same-origin protection
  -> identity candidate validation
  -> atomic identity known-good commit
  -> revision rotation
  -> previous-snapshot recovery
  -> protected BrandMeister candidate validation
  -> protected real BrandMeister setup test
  -> RPTL accepted
  -> Hotspot Security accepted
  -> RPTC accepted
```

The final accepted setup probe used a YWD-DMR-owned numeric/date-style Homebrew software/version identifier plus the `MMDVM_DMO` simplex compatibility profile. This means YWD-DMR does not need to report another project's exact version string.

## Current Alpha1 focus — prove durable tested network configuration

Protocol-field discovery is complete enough to stop changing the working setup packet.

The current implementation target is:

```text
candidate
  -> local validation
  -> real BrandMeister RPTL/RPTK/RPTC test
  -> if accepted, commit that exact candidate
  -> schema 2 known-good state
  -> matching revision-bound secret
```

Schema 1 remains frozen identity-only. Schema 2 adds only non-secret network metadata. The BrandMeister Hotspot Security password lives in a separate mode-`0600` revision-bound secret file under a mode-`0700` daemon secret directory.

The new protected endpoint is:

```text
POST /api/v1/setup/network/test-and-commit
```

The test and commit happen inside one request rather than issuing a reusable proof token. A normal BrandMeister failure returns `committed: false` and must leave the current revision untouched.

The LAN Admin Test Console now exposes **Test & Commit Network** for this exact gate. After a successful durable commit the setup summary becomes `network_complete`, but the Network status is only **CONFIGURED** until the future long-lived BrandMeister backend is actually connected.

The next Pi 5 test must prove:

1. source tests/build/install pass;
2. existing identity survives upgrade;
3. a failed test-and-commit changes nothing;
4. a successful test-and-commit creates schema 2 and a restricted secret file;
5. API/UI never expose the password;
6. restart loads schema 2 and reports `network_complete`;
7. a second successful commit rotates current -> previous with matching secret revisions;
8. deliberate current corruption recovers the previous schema-2 revision and its matching credential.

Only after that gate passes should development move into the **long-lived BrandMeister connection/reconnect state machine**.

See [LAN Admin Test Console](admin-test-console.md), [BrandMeister Live Test Notes](brandmeister-live-test-notes.md), [Known-good Configuration Store](configuration-store.md), [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md), and [Control API v1](../protocols/control-api-v1.md).

The current LAN dashboard remains development-only. Do not router-forward or publicly expose it. HTTPS/WSS trusted-proxy deployment, Secure-cookie deployment, persistent multi-user/device credentials, and PTT safety remain later work.
