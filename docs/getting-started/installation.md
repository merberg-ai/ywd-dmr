# Installing YWD-DMR

> **Current status:** YWD-DMR is in its foundation stage. The final one-command appliance installer is not ready yet. These instructions are for development testing only.

## What the finished installer is supposed to feel like

A normal user should eventually run one command, open the address printed by the installer, and finish setup in the browser. Users should not need to understand systemd, ALSA device names, JSON files, or Git internals.

The installer will automatically check the operating system, CPU, memory, network, storage, time synchronization, audio devices, and supported vocoder devices before changing the system.

## Development run

On a Linux development machine with Go installed:

```bash
git clone https://github.com/merberg-ai/ywd-dmr.git
cd ywd-dmr
./scripts/run-dev.sh
```

By default the development server listens only on `127.0.0.1:8090` for safety.

Open:

```text
http://127.0.0.1:8090/
```

To test from another device on a trusted LAN, an advanced user can temporarily run:

```bash
YWD_DMR_LISTEN=0.0.0.0:8090 ./scripts/run-dev.sh
```

Do **not** expose this early development server directly to the public internet. Authentication has intentionally not been enabled yet and all mutating radio controls remain disabled.
