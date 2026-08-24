# Installing YWD-DMR

> **Current status:** YWD-DMR is still in early development, but the first appliance-style installer is now available for install/uninstall testing from the `dev` branch. DMR/BrandMeister setup is not implemented yet.

## The goal

A finished YWD-DMR install should feel like installing an appliance, not administering a Linux server. A normal user should run one command, let YWD-DMR check the computer, open the address printed at the end, and finish radio setup in the browser.

The current development installer focuses on the foundation we want to prove first:

- build/test the current source before changing the installed service;
- check the requested listening port;
- never steal a port from another program;
- detect active UFW when LAN access is requested and offer a LAN-only rule;
- never claim or remove an existing firewall rule it did not create;
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

If UFW is active, YWD-DMR attempts to identify the directly connected IPv4 LAN subnet. If an equivalent rule does not already exist, it asks before adding a rule limited to that subnet and the selected YWD-DMR TCP port. It does not create an `Anywhere` rule.

For example, on a `192.168.1.0/24` LAN using the default port, the installer-created rule allows only:

```text
192.168.1.0/24 -> 8989/tcp
```

If a matching UFW rule already exists, YWD-DMR uses it but records it as user/system-owned so uninstall will leave it alone.

See [Firewall and LAN Access](../operations/firewall.md) for the complete safety rules.

**Important:** LAN test mode is temporary development behavior. Do not forward the selected port through your router, expose it to the public internet, or place the current unauthenticated build directly on a public interface. Once first-run claiming/authentication is implemented, normal LAN access can become the friendly default.

### Choose a different port

```bash
sudo ./scripts/install.sh --port 8995
```

Or combine it with LAN test mode:

```bash
sudo ./scripts/install.sh --port 8995 --lan-test
```

### Firewall overrides

Skip automatic firewall handling completely:

```bash
sudo ./scripts/install.sh --lan-test --no-firewall
```

For a multi-interface or unusual routing setup, override the detected UFW source subnet:

```bash
sudo ./scripts/install.sh --lan-test --ufw-source 192.168.1.0/24
```

YWD-DMR does not install or enable UFW merely because it is absent. It also does not automatically modify firewalld or an unknown firewall backend in the current development build.

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

When firewall integration is recorded, `ywd-dmr diagnose` also reports whether the UFW rule is installer-managed or an existing/user-owned rule.

## What gets installed

The development appliance installer uses the same layout planned for production:

```text
/opt/ywd-dmr/releases/<release>/   application payloads
/opt/ywd-dmr/current               active release symlink
/opt/ywd-dmr/previous              previous release symlink when available
/etc/ywd-dmr/ywd-dmr.env           protected daemon configuration
/etc/ywd-dmr/runtime.conf           non-sensitive runtime metadata
/etc/ywd-dmr/firewall.conf          non-sensitive firewall ownership metadata, when applicable
/etc/ywd-dmr/install-owned-user     installer ownership marker
/var/lib/ywd-dmr/                  persistent state and user plugins
/var/log/ywd-dmr/                  YWD-DMR log storage
/var/backups/ywd-dmr/              YWD-DMR backups
/etc/systemd/system/ywd-dmrd.service
/usr/local/bin/ywd-dmr
/usr/local/sbin/ywd-dmr-uninstall
```

Application files are owned by root and are read-only to the runtime service. Persistent data is owned by the restricted `ywd-dmr` account. YWD-DMR does not run as root.

The protected daemon configuration is intentionally not world-readable because it will later hold credentials and tokens. Public runtime and firewall metadata must never contain secrets.

## Re-running the installer

Re-running the development installer is allowed. Existing YWD-DMR configuration is preserved unless you explicitly choose a new listener option. A new release directory is installed and the old `current` release becomes `previous`.

The installer builds/tests first and stops the currently installed YWD-DMR service only when the replacement payload is ready. If the replacement fails its health check and a previous release exists, the installer attempts to restore the previous release and both its protected and public runtime configuration.

The development installer itself runs under `sudo`, so its source build also runs with root privileges. `scripts/build.sh` therefore repairs the generated checkout `dist` directory and `dist/ywd-dmrd` ownership back to the original sudo user when it exits. This prevents an install from making the next normal-user developer build fail with `permission denied`. `scripts/build.sh` also accepts `YWD_DMR_BUILD_OUTPUT=/path/to/output` for callers that need a different generated-binary location.

Older `dev` builds from before this ownership fix may already have left `dist` or `dist/ywd-dmrd` owned by root. That is only a generated build artifact; remove or correct that old artifact once before retesting the current build.

Firewall changes occur only after the new daemon passes its health check. If the selected port or LAN changes, an installer-owned UFW rule may be replaced. Existing/user-owned firewall rules are never taken over simply because they happen to match.

## Required development tools

The current `dev` installer compiles from source, so it needs Go. On apt-based systems it can install missing `golang-go`, `curl`, or `iproute2` packages automatically.

Production GitHub releases are planned to ship verified prebuilt binaries, so ordinary users will not eventually need a Go compiler just to install YWD-DMR.

## Removing the test installation

For the normal safe removal that keeps configuration/plugins for a later reinstall:

```bash
sudo ywd-dmr uninstall
```

An installer-created firewall rule is removed because the service is no longer present. A rule that existed before YWD-DMR is left untouched.

For a complete test cleanup, see [Safely Uninstalling YWD-DMR](../operations/uninstall.md).