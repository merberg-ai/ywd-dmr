# Firewall and LAN Access

YWD-DMR may be installed on a Linux computer that already runs other ham-radio, web, or network services. Firewall handling therefore follows one rule above all others:

> YWD-DMR may manage only a firewall rule that the YWD-DMR installer created and recorded as its own.

It must never delete or rewrite an unrelated firewall rule just because that rule happens to use the same port.

## Current automatic support: UFW

The development installer currently knows how to work safely with **UFW**, the firewall commonly used on Ubuntu and Raspberry Pi/Debian-style systems.

Other firewall systems may be detected, but YWD-DMR does not modify them automatically yet. In particular, an active `firewalld` installation is reported to the user and left untouched.

YWD-DMR does not install or enable UFW merely because it is absent. A firewall package is shared system software and is not owned by YWD-DMR.

## Local-only installs

A local-only installation listens on:

```text
127.0.0.1:8989
```

No incoming LAN firewall rule is needed, so the installer does not create one.

## LAN installs

A LAN test installation currently uses:

```bash
sudo ./scripts/install.sh --lan-test
```

If UFW is installed and active, the installer:

1. identifies the IPv4 interface used by the default route;
2. finds that interface's directly connected LAN subnet, such as `192.168.1.0/24`;
3. checks whether an existing UFW rule already allows the selected YWD-DMR TCP port from that subnet;
4. if no rule exists, asks permission to add a LAN-only rule;
5. adds the rule only after the new YWD-DMR daemon has passed its health check.

A typical installer-created rule is equivalent to:

```bash
ufw allow from 192.168.1.0/24 to any port 8989 proto tcp comment 'YWD-DMR managed LAN'
```

The source is the local subnet, **not `Anywhere`**. YWD-DMR must not automatically create a broad public rule for its frontend.

## Existing firewall rules

If an equivalent UFW rule already exists, YWD-DMR uses it but does **not** claim ownership of it.

For example, if the administrator previously created:

```text
8989/tcp ALLOW IN 192.168.1.0/24
```

YWD-DMR records that LAN access depends on an existing/user-owned rule. The uninstaller leaves that rule alone.

This remains true even if the existing rule was originally added manually while troubleshooting YWD-DMR. The installer did not create it, so automatic removal would be guessing.

## Installer-owned rules

When YWD-DMR itself creates the rule, it uses the comment:

```text
YWD-DMR managed LAN
```

and records non-sensitive ownership metadata in:

```text
/etc/ywd-dmr/firewall.conf
```

That metadata contains the firewall provider, source subnet, TCP port, and whether the rule is installer-owned. It contains no passwords or tokens and is readable by the maintenance tools.

The tagged comment and ownership metadata are both safety signals. The uninstaller refuses to guess at an untagged rule.

## Uninstall behavior

Both normal software-only removal and full purge remove an installer-owned YWD-DMR firewall rule because the rule is operational integration for software that is no longer running.

A user-owned or pre-existing rule is never removed automatically.

Normal uninstall still preserves station configuration, data, plugins, logs, backups, and the restricted service account as documented in [Safely Uninstalling YWD-DMR](uninstall.md).

## Advanced options

Skip automatic firewall handling completely:

```bash
sudo ./scripts/install.sh --lan-test --no-firewall
```

Override the automatically detected LAN subnet:

```bash
sudo ./scripts/install.sh --lan-test --ufw-source 192.168.1.0/24
```

`--ufw-source` is intended for unusual routing or multi-interface systems where the automatically selected directly connected subnet is not the network that should reach YWD-DMR.

## If subnet detection fails

YWD-DMR deliberately refuses to fall back to an `Anywhere` rule.

If UFW is active but the installer cannot confidently identify a LAN subnet, it reports the problem and leaves the firewall unchanged. The administrator can rerun with `--ufw-source CIDR` or configure the firewall manually.

## Security reminder during development

The current `--lan-test` dashboard has no authentication yet. Do not port-forward the YWD-DMR frontend from a router, expose the selected port publicly, or create an Internet-wide firewall rule for this development build.

Once first-run claiming and authentication are implemented, LAN access can become the normal friendly setup path while the same LAN-only firewall principle remains in place.