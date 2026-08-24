# Safely Uninstalling YWD-DMR

YWD-DMR is designed to be removable without damaging the rest of the Linux system. The uninstaller removes only paths, services, accounts, and firewall rules that the YWD-DMR installer can prove it owns. It does **not** uninstall shared packages such as Git, Go, ALSA/PipeWire components, Avahi, system libraries, nginx, BPQ, or anything else another application may use.

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
- the installed uninstaller helper;
- a UFW rule only when the YWD-DMR installer created and tagged that rule.

It preserves:

- `/etc/ywd-dmr` station/configuration data other than temporary firewall ownership metadata;
- `/var/lib/ywd-dmr` state and locally installed vocoder plugins;
- `/var/log/ywd-dmr` logs;
- `/var/backups/ywd-dmr` backups;
- the restricted `ywd-dmr` service account, because preserved data still belongs to that account;
- any firewall rule that existed before YWD-DMR or was created manually by the administrator.

This is the normal choice when you are testing an install/uninstall/reinstall cycle.

From a source checkout, the equivalent command is:

```bash
sudo bash scripts/uninstall.sh
```

## Firewall cleanup safety

When the installer creates a UFW LAN rule, it tags the rule with:

```text
YWD-DMR managed LAN
```

and records ownership metadata in:

```text
/etc/ywd-dmr/firewall.conf
```

The uninstaller uses both pieces of information. If the metadata says the rule was not created by YWD-DMR, the rule is left untouched. If ownership metadata is incomplete or the firewall backend is not safely recognized, the uninstaller warns instead of guessing.

An equivalent firewall rule that already existed is considered **user/system-owned**, even if it happens to allow the same port and subnet.

See [Firewall and LAN Access](firewall.md) for the full policy.

## Full removal — purge YWD-DMR data too

```bash
sudo ywd-dmr uninstall --purge-data
```

This removes the application plus YWD-DMR configuration, local/user vocoder plugins, state/history, logs, and YWD-DMR-managed backups. Installer-owned firewall integration is removed as part of the same operation.

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

Nothing is deleted in dry-run mode. Installer-owned firewall removal is printed as a command instead of being executed.

## What the uninstaller is allowed to remove

The path allowlist currently contains only YWD-DMR-owned locations:

- `/opt/ywd-dmr`
- `/etc/ywd-dmr`
- `/var/lib/ywd-dmr`
- `/var/log/ywd-dmr`
- `/var/backups/ywd-dmr`
- `/etc/systemd/system/ywd-dmrd.service`
- `/etc/systemd/system/ywd-dmrd.service.d`
- `/usr/local/bin/ywd-dmr`
- `/usr/local/sbin/ywd-dmr-uninstall`
- `/etc/ywd-dmr/firewall.conf`

The script refuses to recursively remove an unexpected directory.

Firewall cleanup is separate from filesystem deletion. Only the exact supported UFW rule described by YWD-DMR ownership metadata and the YWD-DMR managed comment may be removed automatically.

## Service-account safety

The installer creates the restricted system account `ywd-dmr` only when that name is free. If an unrelated account named `ywd-dmr` already exists, installation stops rather than hijacking it.

A root-owned marker in `/etc/ywd-dmr/install-owned-user` records that the installer created the service account. During a full purge, the uninstaller removes the account only when that marker exists **and** the account still has the expected restricted home/shell profile. If the profile has been changed, the account is left alone.

## Suggested install/uninstall test

1. Install with `sudo ./scripts/install.sh` (or `--lan-test`).
2. Run `sudo ./scripts/verify-install.sh`.
3. If UFW is active, note whether verification reports an installer-managed or existing/user-owned rule.
4. Run `sudo ywd-dmr uninstall` and confirm the service/application are gone while `/etc/ywd-dmr` and `/var/lib/ywd-dmr` remain.
5. If the firewall rule was installer-managed, confirm it was removed. If it was pre-existing, confirm it remains.
6. Reinstall and confirm the preserved listener/config returns and any needed managed firewall rule is recreated.
7. Run `sudo ywd-dmr uninstall --purge-data` and confirm the final safety backup is printed.
8. For an absolutely clean test, use `--purge-data --no-backup` only after you no longer need the safety archive.

## Adding installed files or integrations later

Whenever development adds another system path, service, helper, account, firewall rule, or persistent directory, the installer ownership documentation and uninstall behavior must be updated in the **same change**. See [Installation Ownership](../developers/install-ownership.md).