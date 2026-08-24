# Architecture

## Design rules

YWD-DMR is a headless DMR appliance with client applications. The WebUI is the first client, not the owner of radio state.

Non-negotiable rules:

- Raspberry Pi Zero / ARMv6 is the minimum performance baseline.
- Linux is the platform; Raspberry Pi is an optimized deployment target, not a hard dependency.
- `ywd-dmrd` owns DMR network state, call state, PTT, TX safety, permissions, configuration, and recovery.
- Clients do inexpensive client-side work such as rendering, filtering local UI state, microphone resampling, meters, and optional DSP.
- Control API, event stream, audio stream, and vocoder protocol are separately versioned.
- WebUI, Control API, Event API, and browser Audio Stream share one configurable frontend TCP listener; the default port is 8989.
- The installer must preflight the selected listener before changing or starting services, and must never steal a port from an existing application.
- Vocoders run out-of-process and may be independently installed or updated.
- Locally built/unsigned vocoder plugins are allowed.
- The core must start and remain useful with no vocoder installed.
- DMR networks are backend abstractions; BrandMeister is first, not hardwired forever.
- Settings use validate/test/commit behavior rather than blindly mutating known-good configuration.
- Updates use backup, atomic release install, migrations, health checks, and rollback.
- Documentation changes ship with implementation changes.

## Major components

```text
clients/
  WebUI
  future Android app
       |
       |  one configurable frontend TCP listener
       |  default: 8989
       |
       +-- Control API v1 (HTTP/JSON)
       +-- Event API v1 (WebSocket)
       +-- Audio Stream v1 (binary WebSocket)
       |
       v
   ywd-dmrd
       |
       +-- auth / roles / device sessions
       +-- state machine / PTT lease manager
       +-- audio router
       +-- DMR network backend
       +-- history / configuration / diagnostics
       |
       +-- Vocoder API v1 (local out-of-process IPC)
              |
              +-- hardware plugin
              +-- software plugin supplied by user
              +-- future remote plugin
```

Using one frontend listener keeps LAN access, reverse proxies, firewall rules, Android pairing, and troubleshooting simple. Vocoder IPC should prefer local Unix sockets rather than consuming additional TCP ports unless a future remote-vocoder backend explicitly needs a network listener.

## Security boundary

The daemon is authoritative. Hiding a button in a browser is never considered authorization. Every control operation must be authorized server-side. TX additionally requires a renewable server-owned lease and a time-out timer so a dead client cannot leave the system transmitting.

The daemon's compiled fallback listener is loopback-only. The appliance installer may configure LAN access only through the documented first-run security/claim workflow and after verifying the selected port is available.

Third-party vocoder processes must not require root by design. Future plugin sandboxing should be possible without changing the protocol.

## Storage layout target

```text
/opt/ywd-dmr/releases/<version>/   application releases
/opt/ywd-dmr/current               active release symlink
/opt/ywd-dmr/previous              rollback release symlink
/etc/ywd-dmr/                      system configuration
/var/lib/ywd-dmr/                  database and state
/var/lib/ywd-dmr/plugins/          user-managed plugins
/var/log/ywd-dmr/                  rotating logs
/var/backups/ywd-dmr/              protected backups
```

Application code, configuration, user data, plugins, logs, and backups must remain separate so updates cannot casually destroy local state.
