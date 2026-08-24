# YWD-DMR Listening Port

YWD-DMR is intended to install safely on a Raspberry Pi or Linux computer that may already be running other ham-radio, web, or network software.

## Default port

The default YWD-DMR frontend port is:

```text
8989/TCP
```

The WebUI and client APIs share this one port. A normal installation does **not** need separate ports for the dashboard, REST API, event WebSocket, or browser audio stream.

For example:

```text
http://ywd-dmr.local:8989/
http://192.168.1.42:8989/
```

Keeping the client services on one port makes firewall rules, reverse proxies, Android pairing, and troubleshooting much easier.

## What the installer must do

Before changing the system, the installer must check whether the requested TCP port is already in use.

If port 8989 is free, it can be selected as the default.

If port 8989 is already in use, the installer must:

1. Explain in plain language that another program is already using the port.
2. Show the existing listener/process when Linux can identify it.
3. Never stop, kill, disable, or reconfigure that program automatically.
4. Look for a reasonable free alternative, starting with ports 8990 through 8999.
5. Let the user accept the suggested port, enter another port, or cancel the installation.
6. Re-check the selected port immediately before the YWD-DMR service is started.

A normal YWD-DMR listener should use an unprivileged TCP port from 1024 through 65535. Ports below 1024 should not be suggested by the normal installer.

## Bind address

Port number and bind address are separate settings.

The finished installer/setup wizard should offer simple choices:

- **Local network access** — intended for normal use from phones, tablets, and other computers on the same trusted LAN.
- **This computer only** — bind to `127.0.0.1`; useful for development or when a reverse proxy on the same computer will provide access.
- **Advanced/custom address** — for an administrator who knows which interface/address should be used.

The daemon's built-in fallback remains loopback-only for safety. A production installation should write an explicit configured listen address after the first-run security/claim workflow has been established.

## Firewall relationship

A LAN listener and a firewall rule are separate pieces of configuration. Binding YWD-DMR to `0.0.0.0:8989` does not guarantee another LAN device can reach it if a host firewall blocks the traffic.

When UFW is active, the development installer can offer a LAN-only rule for the selected port. Existing firewall rules remain user/system-owned; only a rule actually created and tagged by YWD-DMR may be removed automatically later.

See [Firewall and LAN Access](firewall.md).

## Existing web servers and reverse proxies

YWD-DMR must not assume that nginx, Apache, Caddy, or another web server belongs to YWD-DMR. The installer must not overwrite or replace an existing web-server configuration.

An advanced user may choose to keep YWD-DMR on a loopback address such as:

```text
127.0.0.1:8989
```

and configure an existing reverse proxy separately.

## Checking a port during development

The repository includes a small preflight helper:

```bash
bash scripts/check-port.sh 8989
```

A successful result looks like:

```text
Port 8989 is available for YWD-DMR.
```

If the port is occupied, the helper exits without changing the other service.

## Changing the port later

The final WebUI and maintenance CLI should allow an administrator to change the frontend port safely. A change must be validated before replacing the known-good configuration. If the requested port is occupied, the current listener should remain working and the change should be rejected with a useful explanation.

Changing the port must be treated as one configuration transaction that also updates YWD-DMR-managed firewall integration when needed. The system should not leave an old managed firewall rule open after moving to a new port, and it must never delete a pre-existing/user-owned rule while making that change.

Changing the port should also update the address displayed by diagnostics, setup/help pages, mDNS guidance, and Android pairing information.