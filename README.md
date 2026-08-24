# YWD-DMR

**YWD-DMR** is a lightweight, web-based DMR transceiver appliance for Linux and Raspberry Pi. It is designed to connect a user directly to DMR networks without RF hardware, using system or remote-client audio and a modular, user-installable vocoder backend.

> **Project status:** very early development / architecture foundation. Do not depend on this repository for on-air operation yet.

## Project goals

- Run well on a Raspberry Pi Zero while scaling up on faster Raspberry Pi and Linux systems.
- Keep the WebUI as a client of a headless, API-first core so future Android and desktop clients can use the same backend.
- Provide an easy, guided setup that does not assume Linux expertise.
- Support pluggable DMR network backends, with BrandMeister first.
- Support out-of-process, independently updatable vocoder plugins.
- Allow locally built and unsigned vocoder plugins; YWD-DMR does not require project signing for third-party plugins.
- Continue operating in a useful no-vocoder mode for setup, metadata, diagnostics, and network management.
- Use a lightweight REST + event-stream + audio-stream client protocol.
- Make security, roles, PTT safety, diagnostics, backup, update, rollback, and recovery part of the architecture from the beginning.
- Keep detailed, plain-language documentation in `/docs/`, shipped with each release and exposed from the WebUI.

## Planned architecture

```text
WebUI / Android / future clients
        |
        |  YWD Control API v1
        |  YWD Event API v1
        |  YWD Audio Stream v1
        v
+-----------------------------+
|          ywd-dmrd           |
| auth / roles / call state   |
| PTT leases / audio routing  |
| DMR network / history       |
+---------------+-------------+
                |
        YWD Vocoder API v1
                |
       user-selected backend
```

The core daemon remains authoritative for network state, calls, PTT, permissions, configuration, and safety. Clients may do inexpensive client-side work such as UI rendering, microphone resampling, level metering, and optional audio processing.

## Minimum target

The Raspberry Pi Zero / ARMv6 is the minimum performance target. YWD-DMR is not Raspberry-Pi-only: the same core should run on normal Linux systems and automatically take advantage of stronger hardware where useful.

## First milestone: `0.1.0-alpha1` — First QSO

The initial on-air milestone is intentionally narrow:

1. Install cleanly on supported Linux/Raspberry Pi systems.
2. Complete first-run setup through the WebUI.
3. Connect to BrandMeister.
4. Load one compatible vocoder backend.
5. Select local or remote-client audio.
6. Select a talkgroup.
7. Receive DMR audio.
8. Hold PTT and transmit safely using a server-owned TX lease.
9. Complete a Parrot test.
10. Expose useful status, logs, diagnostics, and recovery tools.

## Documentation rule

A change is not complete until the matching documentation is updated. User-visible, operational, architectural, API, configuration, installation, update, and troubleshooting changes must update `/docs/` in the same change.

Start with [the documentation index](docs/README.md).

## Development

The repository is being built API-first and dependency-light. The initial core daemon is written in Go, with vocoder backends deliberately kept out-of-process so plugin authors are free to use other languages and hardware interfaces.

See [docs/developers/architecture.md](docs/developers/architecture.md) and [CONTRIBUTING.md](CONTRIBUTING.md) as the project fills out.

## License

A final project license is not yet declared. Licensing must be decided before incorporating or adapting code from external DMR implementations. Do not copy code into this repository merely because it is publicly visible.
