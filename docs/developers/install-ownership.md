# Installation Ownership and Safe Removal

YWD-DMR must behave like a polite appliance installer: it may clean up what it owns, but it must never guess that a shared Linux component belongs to it.

## Ownership rule

Every path created outside the release/application tree must have a documented owner and removal policy before it is added to the installer.

The production installer will maintain a machine-readable installation manifest under `/var/lib/ywd-dmr/`. The manifest is advisory, not a license to delete arbitrary paths: the uninstaller must still validate manifest entries against YWD-DMR allowlisted prefixes and known service/helper names.

A corrupted or hand-edited manifest must never turn into `rm -rf` against an arbitrary path.

## Initial reserved paths

| Path | Purpose | Normal uninstall | Full purge |
| --- | --- | --- | --- |
| `/opt/ywd-dmr` | application releases | remove | remove |
| `/etc/ywd-dmr` | configuration | preserve | remove |
| `/var/lib/ywd-dmr` | database, state, plugins | preserve | remove |
| `/var/log/ywd-dmr` | application logs | preserve | remove |
| `/var/backups/ywd-dmr` | managed backups | preserve | remove after safety backup |
| `/etc/systemd/system/ywd-dmrd.service` | daemon unit | remove | remove |
| `/etc/systemd/system/ywd-dmrd.service.d` | daemon overrides | remove | remove |
| `/usr/local/bin/ywd-dmr` | maintenance CLI | remove | remove |
| `/usr/local/sbin/ywd-dmr-uninstall` | uninstall helper | remove | remove |

Local/user vocoder plugins belong under `/var/lib/ywd-dmr/plugins` so normal core updates and normal software-only uninstall do not overwrite or delete them.

## Shared packages are never owned

If YWD-DMR installs or depends on a distribution package, that does **not** make the package YWD-DMR-owned. The uninstaller must not automatically remove packages such as networking tools, audio libraries, Avahi, Git, or system runtimes because other applications may depend on them.

## Service account

If the installer creates the `ywd-dmr` service account, it must also create an ownership marker containing the exact marker value expected by the uninstaller. Removal is permitted only when both are true:

1. the marker says the account was created by the YWD-DMR installer; and
2. the account still has the restricted service-account home/shell profile YWD-DMR created.

If either check fails, leave the account in place and report it to the user.

## Change checklist

Any change that adds, moves, or removes an installed system path must update all of these together:

- installer logic;
- uninstall logic;
- this document;
- user-facing uninstall documentation;
- backup/update logic if the path contains persistent state;
- support-bundle/redaction logic if the path may contain secrets.

This rule is part of the project's definition of done.
