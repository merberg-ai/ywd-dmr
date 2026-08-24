# YWD Vocoder API v1

YWD-DMR does not require the core application to contain an AMBE implementation. Vocoders run as independent processes and communicate through a small versioned protocol.

## Goals

- User-installable and independently updatable.
- Third-party/local plugins do not need YWD project signing.
- A plugin crash must not crash `ywd-dmrd`.
- Plugin authors may use C, C++, Rust, Go, Python, or another suitable language.
- The daemon should be able to restart a failed plugin.
- The protocol must permit future local Unix-socket and remote transports.

## Discovery/manifest fields

A plugin should declare at least:

```json
{
  "name": "example-vocoder",
  "version": "0.1.0",
  "api_min": 1,
  "api_max": 1,
  "capabilities": ["encode", "decode"],
  "codec": "ambe2",
  "pcm_rate": 8000,
  "channels": 1
}
```

Unsigned does not mean invisible: the UI should clearly identify local/third-party plugins, source/path, version, capabilities, and a file hash where practical.

The exact binary frame envelope is intentionally not frozen in this foundation commit. It will be documented here before the first vocoder implementation is considered API v1 compatible.
