# First-run Setup

The production browser wizard is not complete yet, but the daemon-side setup flow now includes station-identity validation and durable commit, one-time installation claim, normal administrator login/logout, role authorization, browser same-origin mutation protection, and the first protected BrandMeister candidate validation.

The finished wizard will use large, plain-language steps:

1. Claim the new YWD-DMR installation using a one-time setup code.
2. Create the administrator account. There is no default `admin/admin` password.
3. Enter and save callsign, DMR ID, and ESSID.
4. Select BrandMeister and a master server.
5. Enter the BrandMeister hotspot password.
6. Test the real master connection and credentials before saving them as active.
7. Choose and test microphone and speaker devices.
8. Detect or configure a vocoder backend, or continue safely in no-vocoder mode.
9. Run a complete station health check.
10. Finish setup and open the dashboard.

A failed network test should explain whether the problem looks like the DMR identity/login, hotspot password, master configuration, or a timeout/network problem rather than showing only a generic error.

## What works now on `dev`

### Setup status

```text
GET /api/v1/setup/status
```

A fresh unclaimed install reports `stage: unclaimed` and `next_step: claim`. After a claimed Admin successfully commits station identity, setup advances to `stage: identity_complete` and `next_step: network`.

### Retrieve the one-time claim code locally

```bash
sudo ywd-dmr claim-code
```

The code remains stable across daemon restarts while the appliance is unclaimed. It is consumed by a successful claim and is not regenerated after the installation has been claimed.

### Claim and normal Admin login

Implemented and Pi 5 validated:

```text
POST /api/v1/setup/claim
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

A successful claim creates the first Admin and burns the bootstrap path. Normal login creates a fresh opaque HttpOnly session. Sessions and login throttling are memory-only, while the durable administrator account survives restart.

### Roles and browser safety

Server-side role hierarchy:

```text
Observer < Operator < Admin
```

State-changing browser requests pass through same-origin protection. Direct API clients remain usable without pretending to be browsers. This foundation is installed-machine validated.

### Validate station identity without saving

```text
POST /api/v1/setup/identity/validate
```

The daemon normalizes and validates callsign, base DMR ID, and ESSID. This endpoint is public and changes nothing.

### Save station identity

```text
POST /api/v1/setup/identity/commit
```

This endpoint is Admin-only and browser same-origin protected.

The complete installed Pi 5 exercise is now successful. It proved:

- revision 1 and revision 2 commits;
- normalized identity storage;
- `0600 ywd-dmr:ywd-dmr` current and previous snapshots;
- invalid-candidate preservation;
- restart loading of the newest known-good revision;
- automatic recovery of the API-created previous revision after deliberately corrupting current;
- explicit `recovered` setup state and service-journal warning;
- clean return to a fresh unclaimed/no-config test state.

### Validate BrandMeister settings locally

The next setup step now has a protected local validator:

```text
POST /api/v1/setup/network/validate
```

It requires an Admin session because the request contains the BrandMeister hotspot password.

Example request:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your BrandMeister hotspot password"
}
```

The response never returns the password. It only returns normalized non-secret fields and whether a password was supplied:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "master.example.net",
    "master_port": 62031,
    "password_set": true
  },
  "errors": []
}
```

Leaving `master_port` at `0` tells the current Alpha1 validator to use the standard BrandMeister/Homebrew port `62031`.

This is only a **local settings check**. It does not contact BrandMeister and does not save anything.

### What is intentionally not available yet

There is no network `test` or `commit` endpoint yet.

The daemon will not pretend that a valid-looking hostname/password means the connection works. The required network transaction is:

```text
candidate -> local validation -> real master login/auth/config test -> durable commit
```

The next backend work is the short-lived BrandMeister/Homebrew handshake probe. Only after that real test works will network configuration be allowed into the known-good store.

## Current safety note

The current LAN test build is development software. Keep it on a trusted local network only. Do not forward the YWD-DMR frontend port through a router or expose it directly to the public internet.

Production HTTPS/WSS, trusted reverse-proxy behavior, Secure-cookie deployment, long-lived BrandMeister networking, radio controls, and PTT are not finished.

For more detail, see [Setup and Security Phase](../developers/setup-security-phase.md), [Protected Station-Identity Commit Validation Notes](../developers/identity-commit-validation-notes.md), and [DMR Network Backend and BrandMeister Setup Contract](../developers/network-backend-contract.md).
