# Installing YWD-DMR

> **Current status:** YWD-DMR is in its foundation stage. The final one-command appliance installer is not ready yet. These instructions are for development testing only.

## What the finished installer is supposed to feel like

A normal user should eventually run one command, open the address printed by the installer, and finish setup in the browser. Users should not need to understand systemd, ALSA device names, JSON files, Git internals, or Linux port-management commands.

The installer will automatically check the operating system, CPU, memory, network, storage, time synchronization, audio devices, supported vocoder devices, and the requested YWD-DMR listening port before changing the system.

### Frontend port

YWD-DMR uses **8989/TCP** as its default frontend port.

The WebUI, REST API, event WebSocket, and browser audio stream share the same listener, so a normal installation needs only one YWD-DMR frontend port.

Before installation, YWD-DMR must check that port 8989 is available. If another application is already using it, the installer will not disturb that application. It will offer a free alternative and allow the user to choose another port or cancel.

See [YWD-DMR Listening Port](../operations/listening-ports.md) for the full port and bind-address policy.

## Development run

On a Linux development machine with Go installed:

```bash
git clone https://github.com/merberg-ai/ywd-dmr.git
cd ywd-dmr
git checkout dev
./scripts/run-dev.sh
```

By default the development server listens only on `127.0.0.1:8989` for safety.

Open:

```text
http://127.0.0.1:8989/
```

You can check the default port first with:

```bash
bash scripts/check-port.sh 8989
```

To test from another device on a trusted LAN, an advanced user can temporarily run:

```bash
YWD_DMR_LISTEN=0.0.0.0:8989 ./scripts/run-dev.sh
```

Do **not** expose this early development server directly to the public internet. Authentication has intentionally not been enabled yet and all mutating radio controls remain disabled.
