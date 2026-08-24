# Safely Uninstalling YWD-DMR

YWD-DMR is designed to be removable without damaging the rest of the Linux system. The uninstaller must only remove files, services, users, and directories that belong to YWD-DMR.

It must **not** remove shared packages such as Git, Go, ALSA/PipeWire components, Avahi, system libraries, or anything else that another program may use.

## Normal removal

During development, run:

```bash
sudo bash scripts/uninstall.sh
```

The normal mode removes the YWD-DMR application and service while preserving your settings, local vocoder plugins, history, and backups. This is the safest choice if you may reinstall later.

The finished appliance will expose the same operation through the WebUI and the `ywd-dmr` maintenance command so normal users will not need to remember the script name.

## Full removal

To remove YWD-DMR configuration, local vocoder plugins, history, logs, and YWD-DMR-managed backups too:

```bash
sudo bash scripts/uninstall.sh --purge-data
```

For safety, this creates one final backup outside the YWD-DMR data directories before anything is purged. The uninstaller prints the exact backup filename when it finishes.

A full purge requires typing:

```text
REMOVE YWD-DMR
```

This is intentional. A single accidental click or pasted command should not silently erase a station configuration or a locally built vocoder plugin.

## Absolutely no retained backup

Advanced users who really want no YWD-DMR data retained may use:

```bash
sudo bash scripts/uninstall.sh --purge-data --no-backup
```

Use this only when you are certain you do not need the configuration or local plugins again.

## Preview without changing anything

To see what the uninstaller would do:

```bash
sudo bash scripts/uninstall.sh --purge-data --dry-run
```

Nothing is deleted in dry-run mode.

## What the uninstaller is allowed to remove

The initial installation layout reserves these YWD-DMR-owned locations:

- `/opt/ywd-dmr` — application releases
- `/etc/ywd-dmr` — system configuration
- `/var/lib/ywd-dmr` — database, state, and user/local plugins
- `/var/log/ywd-dmr` — logs
- `/var/backups/ywd-dmr` — YWD-DMR-managed backups
- `/etc/systemd/system/ywd-dmrd.service` — core daemon service
- `/etc/systemd/system/ywd-dmrd.service.d` — YWD-DMR service overrides
- `/usr/local/bin/ywd-dmr` — maintenance command
- `/usr/local/sbin/ywd-dmr-uninstall` — installed uninstaller helper

The script contains an allowlist and refuses to recursively remove an unexpected directory.

## Service account safety

A future production installer may create a dedicated `ywd-dmr` service account. The uninstaller may remove that account only when an installer ownership marker exists and the account still matches the restricted service-account profile created by YWD-DMR. If it has been changed into a normal user account, the uninstaller leaves it alone.

## Adding new installed files later

Whenever development adds another system path, service, helper, or persistent directory, the installer ownership documentation and uninstall behavior must be updated in the **same change**. See [Installation Ownership](../developers/install-ownership.md).
