# YWD Event API v1

The Event API will use an authenticated WebSocket so clients do not poll the Pi continuously.

Planned event families include:

- `rx.started`, `rx.updated`, `rx.ended`
- `tx.started`, `tx.ended`
- `destination.changed`
- `network.connected`, `network.disconnected`
- `vocoder.connected`, `vocoder.failed`
- `audio.device_changed`
- `talkgroup.static_added`, `talkgroup.static_removed`
- `system.warning`

Each event will carry a monotonically increasing sequence value and timestamp so clients can detect gaps and refresh state through the Control API when needed.

The WebSocket must enforce authentication, role authorization where applicable, Origin checks for browser sessions, message size limits, rate limits, and bounded queues suitable for a Pi Zero.
