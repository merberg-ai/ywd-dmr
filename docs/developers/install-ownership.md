# Installation Ownership and Safe Removal

YWD-DMR must behave like a polite appliance installer: it may clean up what it owns, but it must never guess that a shared Linux component or firewall rule belongs to it.

## Ownership rule

Every path or system integration created outside the release/application tree must have a documented owner and removal policy before it is added to the installer.

The production installer will maintain a machine-readable installation manifest under `/var/lib/ywd-dmr/`. The manifest is advisory, not a license to delete arbitrary paths: the uninstaller must still validate manifest entries against YWD-DMR allowlisted prefixes and known service/helper/integration names.

A corrupted or hand-edited manifest must never turn into `rm -rf` against an arbitrary path or deletion of an unrelated firewall rule.

## Initial reserved paths

| Path | Purpose | Normal uninstall | Full purge |
| --- | --- | --- | --- |
| `/opt/ywd-dmr` | application releases | remove | remove |
| `/etc/ywd-dmr` | configuration | preserve | remove |
| `/etc/ywd-dmr/firewall.conf` | non-sensitive firewall ownership metadata | remove after firewall cleanup | remove after firewall cleanup |
| `/var/lib/ywd-dmr` | database, state, plugins | preserve | remove |
| `/var/log/ywd-dmr` | application logs | preserve | remove |
| `/var/backups/ywd-dmr` | managed backups | preserve | remove after safety backup |
| `/etc/systemd/system/ywd-dmrd.service` | daemon unit | remove | remove |
| `/etc/systemd/system/ywd-dmrd.service.d` | daemon overrides | remove | remove |
| `/usr/local/bin/ywd-dmr` | maintenance CLI | remove | remove |
| `/usr/local/sbin/ywd-dmr-uninstall` | uninstall helper | remove | remove |

Local/user vocoder plugins belong under `/var/lib/ywd-dmr/plugins` so normal core updates and normal software-only uninstall do not overwrite or delete them.

## Protected configuration versus public runtime metadata

YWD-DMR deliberately separates information that may eventually contain credentials from harmless information local tools need to display.

`/etc/ywd-dmr/ywd-dmr.env` is protected daemon configuration. It is owned by `root:ywd-dmr` with mode `0640`. Future BrandMeister credentials, API tokens, and other secrets must never be moved into a world-readable file merely to make a helper command convenient.

`/etc/ywd-dmr/runtime.conf` contains only non-sensitive runtime metadata needed by local user-facing tools. The initial file contains the configured listen address/port and is owned by `root:root` with mode `0644`.

`/etc/ywd-dmr/firewall.conf` contains only non-sensitive firewall integration metadata: provider, source subnet, TCP port, installer-ownership flag, and the managed rule comment. It is also `root:root` mode `0644` so diagnostics can explain the firewall state without exposing secrets.

The `ywd-dmr` maintenance command reads public metadata for commands such as `ywd-dmr url` and `ywd-dmr diagnose`. This lets an ordinary local user inspect the appliance without gaining read access to protected daemon configuration.

Installer rollback must restore protected/public listener state together so the maintenance CLI never advertises a different listener from the daemon that was restored.

## Firewall ownership

Firewall ownership is stricter than simple port matching.

If an equivalent firewall rule already exists before YWD-DMR needs it, that rule remains user/system-owned. YWD-DMR may use the access it provides but must record `YWD_DMR_FIREWALL_MANAGED=0` and must not remove that rule during uninstall.

When YWD-DMR creates a UFW rule itself, it:

1. limits the source to the selected LAN subnet rather than `Anywhere`;
2. tags the rule with the exact comment `YWD-DMR managed LAN`;
3. records the rule as installer-owned in `/etc/ywd-dmr/firewall.conf`.

Automatic removal requires matching ownership metadata and a supported, identifiable YWD-DMR-managed rule. If either side is missing or ambiguous, the uninstaller warns and leaves the firewall alone rather than guessing.

Firewall rules are operational integration rather than station data. Therefore an installer-owned rule is removed during both normal software-only uninstall and full purge. Preserved configuration can recreate a needed managed rule on reinstall.

Current automatic firewall support is UFW only. Other firewall packages are shared system components and are not installed, enabled, disabled, or reconfigured automatically merely because YWD-DMR is present.

See [Firewall and LAN Access](../operations/firewall.md).

## Shared packages are never owned

If YWD-DMR installs or depends on a distribution package, that does **not** make the package YWD-DMR-owned. The uninstaller must not automatically remove packages such as networking tools, audio libraries, Avahi, Git, UFW, or system runtimes because other applications may depend on them.

## Service account

If the installer creates the `ywd-dmr` service account, it must also create an ownership marker containing the exact marker value expected by the uninstaller. Removal is permitted only when both are true:

1. the marker says the account was created by the YWD-DMR installer; and
2. the account still has the restricted service-account home/shell profile YWD-DMR created.

If either check fails, leave the account in place and report it to the user.

## Change checklist

Any change that adds, moves, or removes an installed system path or integration must update all of these together:

- installer logic;
- uninstall logic;
- this document;
- user-facing install/uninstall documentation;
- firewall/integration documentation when applicable;
- backup/update logic if the path contains persistent state;
- support-bundle/redaction logic if the path may contain secrets.

This rule is part of the project's definition of done.