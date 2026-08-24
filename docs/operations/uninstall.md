# Safely Uninstalling YWD-DMR

YWD-DMR is designed to be removable without damaging the rest of the Linux system. The uninstaller removes only paths/services/accounts that the YWD-DMR installer owns. It does **not** uninstall shared packages such as Git, Go, ALSA/PipeWire components, Avahi, system libraries, nginx, BPQ, or anything else another application may use.

The development appliance installer installs the safe uninstaller at:

```text
/usr/local/sbin/ywd-dmr-uninstall
```

and exposes it through the friendlier maintenance command.

## Normal removal — keep your YWD-DMR data

```bash
sudo ywd-dmr uninstall
```

This removes:

- the installed YWD-DMR application releases;
- the `ywd-dmrd.service` systemd unit;
- the maintenance command;
- the installed uninstaller helper.

It preserves:

- `/etc/ywd-dmr` configuration;
- `/var/lib/ywd-dmr` state and locally installed vocoder plugins;
- `/var/log/ywd-dmr` logs;
- `/var/backups/ywd-dmr` backups;
- the restricted `ywd-dmr` service account, because preserved data still belongs to that account.

This is the normal choice when you are testing an install/uninstall/reinstall cycle.

From a source checkout, the equivalent command is:

```bash
sudo bash scripts/uninstall.sh
```

## Full removal — purge YWD-DMR data too

```bash
sudo ywd-dmr uninstall --purge-data
```

This removes the application plus YWD-DMR configuration, local/user vocoder plugins, state/history, logs, and YWD-DMR-managed backups.

Before purging, the uninstaller creates one final protected archive outside the normal YWD-DMR directory tree, for example:

```text
/var/backups/ywd-dmr-uninstall-20260824-070000.tar.gz
```

If that safety backup fails, the purge stops before removing the persistent data.

A full purge requires typing:

```text
REMOVE YWD-DMR
```

This is deliberate. An accidental click or pasted command should not silently erase a station configuration or a locally built vocoder plugin.

## Absolutely no retained backup

Advanced/test use only:

```bash
sudo ywd-dmr uninstall --purge-data --no-backup
```

This is the cleanest possible removal and is useful when validating that a fresh install really starts from scratch.

## Preview without changing anything

```bash
sudo ywd-dmr uninstall --purge-data --dry-run
```

Nothing is deleted in dry-run mode.

## What the uninstaller is allowed to remove

The allowlist currently contains only YWD-DMR-owned locations:

- `/opt/ywd-dmr`
- `/etc/ywd-dmr`
- `/var/lib/ywd-dmr`
- `/var/log/ywd-dmr`
- `/var/backups/ywd-dmr`
- `/etc/systemd/system/ywd-dmrd.service`
- `/etc/systemd/system/ywd-dmrd.service.d`
- `/usr/local/bin/ywd-dmr`
- `/usr/local/sbin/ywd-dmr-uninstall`

The script refuses to recursively remove an unexpected directory.

## Service-account safety

The installer creates the restricted system account `ywd-dmr` only when that name is free. If an unrelated account named `ywd-dmr` already exists, installation stops rather than hijacking it.

A root-owned marker in `/etc/ywd-dmr/install-owned-user` records that the installer created the service account. During a full purge, the uninstaller removes the account only when that marker exists **and** the account still has the expected restricted home/shell profile. If the profile has been changed, the account is left alone.

## Suggested install/uninstall test

1. Install with `sudo ./scripts/install.sh` (or `--lan-test`).
2. Run `sudo ./scripts/verify-install.sh`.
3. Run `sudo ywd-dmr uninstall` and confirm the service/application are gone while `/etc/ywd-dmr` and `/var/lib/ywd-dmr` remain.
4. Reinstall and confirm the preserved listener/config returns.
5. Run `sudo ywd-dmr uninstall --purge-data` and confirm the final safety backup is printed.
6. For an absolutely clean test, use `--purge-data --no-backup` only after you no longer need the safety archive.

## Adding installed files later

Whenever development adds another system path, service, helper, account, or persistent directory, the installer ownership documentation and uninstall behavior must be updated in the **same change**. See [Installation Ownership](../developers/install-ownership.md).
