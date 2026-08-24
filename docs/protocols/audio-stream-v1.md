# YWD Audio Stream v1

The lightweight baseline remote-audio transport is a binary authenticated WebSocket shared by WebUI and future native clients.

Baseline audio format:

```text
PCM: signed 16-bit little endian
Rate: 8000 Hz
Channels: mono
Nominal packet duration: 20 ms
Samples per packet: 160
Payload audio bytes: 320
```

The client should perform microphone resampling, level metering, and optional inexpensive DSP when practical. The daemon owns bounded jitter/buffer behavior, call state, permissions, and the vocoder path.

Transmit audio is accepted only while the authenticated client owns a valid TX lease. Dropped lease heartbeats or the server TX timeout immediately end transmission.

Future transports such as Opus or WebRTC may be added as optional capabilities without removing the simple PCM baseline.
