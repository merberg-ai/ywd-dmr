# Installing YWD-DMR

> **Current status:** YWD-DMR is still in early development, but the first appliance-style installer is now available for install/uninstall testing from the `dev` branch. DMR/BrandMeister setup is not implemented yet.

## The goal

A finished YWD-DMR install should feel like installing an appliance, not administering a Linux server. A normal user should run one command, let YWD-DMR check the computer, open the address printed at the end, and finish radio setup in the browser.

The current development installer focuses on the foundation we want to prove first:

- build/test the current source before changing the installed service;
- check the requested listening port;
- never steal a port from another program;
- use a dedicated restricted `ywd-dmr` service account;
- keep application code separate from configuration/data/plugins;
- install a hardened systemd service;
- start the service automatically at boot;
- run a health check after installation;
- retain `current`/`previous` release slots for rollback work;
- install simple maintenance and safe-uninstall commands.

## Default frontend port

YWD-DMR uses **8989/TCP** by default. The WebUI, REST API, event WebSocket, and browser audio stream share this one listener.

If 8989 belongs to another program, the installer does not stop or modify it. On a fresh installation it can suggest a free port from 8990 through 8999, or you can choose any unprivileged TCP port from 1024 through 65535.

See [YWD-DMR Listening Port](../operations/listening-ports.md).

## Test the appliance installer

Clone the development branch:

```bash
git clone -b dev https://github.com/merberg-ai/ywd-dmr.git
cd ywd-dmr
```

Then install locally:

```bash
sudo ./scripts/install.sh
```

Because dashboard authentication is not implemented yet, the safe default listens only on this computer at:

```text
http://127.0.0.1:8989/
```

### Testing from another device on your LAN

For development testing from a phone or another computer on the same trusted home network:

```bash
sudo ./scripts/install.sh --lan-test
```

The installer prints the LAN address when it succeeds.

**Important:** LAN test mode is temporary development behavior. Do not forward the selected port through your router, expose it to the public internet, or place the current unauthenticated build directly on a public interface. Once first-run claiming/authentication is implemented, normal LAN access can become the friendly default.

### Choose a different port

```bash
sudo ./scripts/install.sh --port 8995
```

Or combine it with LAN test mode:

```bash
sudo ./scripts/install.sh --port 8995 --lan-test
```

## Verify the installation

After installation:

```bash
sudo ./scripts/verify-install.sh
```

You can also use the installed maintenance command as your normal user:

```bash
ywd-dmr status
ywd-dmr diagnose
ywd-dmr url
ywd-dmr logs
```

`ywd-dmr diagnose` and `ywd-dmr url` do not need access to passwords or future BrandMeister credentials. The installer writes only non-sensitive listener information to `/etc/ywd-dmr/runtime.conf`, which local users may read. Protected daemon configuration remains in `/etc/ywd-dmr/ywd-dmr.env` with restricted permissions.

## What gets installed

The development appliance installer uses the same layout planned for production:

```text
/opt/ywd-dmr/releases/<release>/   application payloads
/opt/ywd-dmr/current               active release symlink
/opt/ywd-dmr/previous              previous release symlink when available
/etc/ywd-dmr/ywd-dmr.env           protected daemon configuration
/etc/ywd-dmr/runtime.conf           non-sensitive runtime metadata
/etc/ywd-dmr/install-owned-user     installer ownership marker
/var/lib/ywd-dmr/                  persistent state and user plugins
/var/log/ywd-dmr/                  YWD-DMR log storage
/var/backups/ywd-dmr/              YWD-DMR backups
/etc/systemd/system/ywd-dmrd.service
/usr/local/bin/ywd-dmr
/usr/local/sbin/ywd-dmr-uninstall
```

Application files are owned by root and are read-only to the runtime service. Persistent data is owned by the restricted `ywd-dmr` account. YWD-DMR does not run as root.

The protected daemon configuration is intentionally not world-readable because it will later hold credentials and tokens. Public runtime metadata must never contain secrets.

## Re-running the installer

Re-running the development installer is allowed. Existing YWD-DMR configuration is preserved unless you explicitly choose a new listener option. A new release directory is installed and the old `current` release becomes `previous`.

The installer builds/tests first and stops the currently installed YWD-DMR service only when the replacement payload is ready. If the replacement fails its health check and a previous release exists, the installer attempts to restore the previous release and both its protected and public runtime configuration.

## Required development tools

The current `dev` installer compiles from source, so it needs Go. On apt-based systems it can install missing `golang-go`, `curl`, or `iproute2` packages automatically.

Production GitHub releases are planned to ship verified prebuilt binaries, so ordinary users will not eventually need a Go compiler just to install YWD-DMR.

## Removing the test installation

For the normal safe removal that keeps configuration/plugins for a later reinstall:

```bash
sudo ywd-dmr uninstall
```

For a complete test cleanup, see [Safely Uninstalling YWD-DMR](../operations/uninstall.md).
