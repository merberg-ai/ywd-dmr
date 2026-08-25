# First-run Setup

The production browser wizard is not complete yet, but the daemon-side setup flow now includes station-identity validation and commit, durable configuration-state reporting, one-time installation claim, normal administrator login/logout, role authorization, and browser same-origin mutation protection.

The finished wizard will use large, plain-language steps:

1. Claim the new YWD-DMR installation using a one-time setup code.
2. Create the administrator account. There is no default `admin/admin` password.
3. Enter and save callsign, DMR ID, and ESSID.
4. Select the DMR network and master server.
5. Test network credentials before saving them as active.
6. Choose and test microphone and speaker devices.
7. Detect or configure a vocoder backend, or continue safely in no-vocoder mode.
8. Run a complete station health check.
9. Finish setup and open the dashboard.

If a test fails, the wizard should explain what failed in normal language and let the user retry or continue when the failed component is optional.

## What works now on `dev`

### Setup status

```text
GET /api/v1/setup/status
```

A fresh unclaimed install reports `stage: unclaimed` and `next_step: claim`. After a claimed administrator successfully commits station identity, setup advances to `stage: identity_complete` and `next_step: network`.

### Retrieve the one-time claim code locally

```bash
sudo ywd-dmr claim-code
```

The code remains stable across daemon restarts while the appliance is unclaimed. It is consumed by a successful claim and is not regenerated after the installation has been claimed.

### Claim the appliance

```text
POST /api/v1/setup/claim
```

A successful claim creates the first Admin account, removes the usable bootstrap path, and establishes an opaque HttpOnly browser session. The complete claim lifecycle is installed-machine validated on the Raspberry Pi 5.

### Administrator login after restart

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

Wrong username and wrong password deliberately produce the same generic authentication failure. Sessions and login throttling are memory-only; durable administrator state survives restart. The complete login/logout/throttle lifecycle is installed-machine validated on the Raspberry Pi 5.

### Roles and browser safety

The server-side role hierarchy is:

```text
Observer < Operator < Admin
```

The first claimed account is Admin. Unknown roles fail closed.

State-changing browser requests (`POST`, `PUT`, `PATCH`, `DELETE`) pass through same-origin protection. Direct API clients remain usable without browser headers. The role/origin foundation is installed-machine validated on the Raspberry Pi 5.

### Validate station identity without saving

```text
POST /api/v1/setup/identity/validate
```

The daemon normalizes and validates:

- **Callsign** — trimmed and converted to uppercase.
- **DMR ID** — the base numeric DMR ID, kept separate from a hotspot suffix.
- **ESSID** — a value from 0 through 99 for a network/device suffix when needed.

This endpoint changes nothing and remains useful for live form validation.

### Save station identity

The first normal protected setup mutation is now implemented:

```text
POST /api/v1/setup/identity/commit
```

It requires an Admin session. Browser callers must also be same-origin.

The daemon does not save identity to a separate setup file. It sends the request through the same known-good configuration store used for startup/recovery. The request is normalized and validated first; only a valid candidate can replace durable known-good state.

A first successful commit creates revision 1. Later successful commits increment the revision and preserve the prior known-good snapshot for rollback/recovery. Invalid, unauthenticated, or rejected cross-origin requests must leave the current known-good configuration untouched.

The implementation and automated tests are on `dev`. Installed Raspberry Pi validation of revision changes, file protection, restart persistence, previous-snapshot rotation, and invalid-candidate preservation is the current test step.

## Current safety note

The current LAN test build is still development software. Claim, login, role authorization, and browser-origin filtering are validated, and the first Admin-protected configuration mutation now exists. Production HTTPS/WSS, trusted reverse-proxy behavior, Secure-cookie deployment, BrandMeister controls, radio controls, and PTT are not finished.

Keep it on a trusted local network only. Do not forward the YWD-DMR frontend port through a router or expose it directly to the public internet.

For the implementation order and security boundary, see [Setup and Security Phase](../developers/setup-security-phase.md), [Known-good Configuration Store](../developers/configuration-store.md), and [Authorization and Browser Mutation Protection](../developers/authorization-model.md).
