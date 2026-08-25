# First-run Setup

The production browser wizard is not complete yet, but the daemon-side setup flow now includes station-identity validation and durable commit, one-time installation claim, normal administrator login/logout, role authorization, browser same-origin mutation protection, protected BrandMeister candidate validation, and the first real non-persisting BrandMeister connectivity test.

The finished wizard will use large, plain-language steps:

1. Claim the new YWD-DMR installation using a one-time setup code.
2. Create the administrator account. There is no default `admin/admin` password.
3. Enter and save callsign, DMR ID, and ESSID.
4. Select BrandMeister and a master server.
5. Enter the BrandMeister Hotspot Security password.
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

The installed Pi 5 exercise proved revision 1 and revision 2 commits, normalized identity storage, `0600 ywd-dmr:ywd-dmr` current/previous snapshots, invalid-candidate preservation, restart persistence, and automatic recovery from the API-created previous snapshot after deliberate corruption of current.

### Validate BrandMeister settings locally

```text
POST /api/v1/setup/network/validate
```

It requires an Admin session because the request contains the BrandMeister Hotspot Security password.

Example request:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your BrandMeister hotspot password"
}
```

Current BrandMeister guidance limits the Hotspot Security password to 20 characters. YWD-DMR therefore rejects an empty password, a password longer than 20 characters, or one containing control characters before any network test begins. BrandMeister also recommends avoiding special characters; YWD-DMR does not currently invent a stricter character whitelist beyond the documented length and control-character rules.

The response never returns the password. It only returns normalized non-secret fields and whether a password was supplied. Leaving `master_port` at `0` uses the Homebrew default `62031`.

This local validator is installed Pi 5 validated. It does **not** contact BrandMeister and does not save anything.

### Test the real BrandMeister master

A real, still non-persisting test endpoint is now implemented on `dev`:

```text
POST /api/v1/setup/network/test
```

It uses the same network request shape as `/network/validate`, but it also requires an already saved station identity. YWD-DMR reads callsign/DMR ID/ESSID from known-good state so the network request cannot silently substitute another station identity.

The temporary test performs only the Homebrew setup handshake:

```text
RPTL -> login acknowledgement and salt
RPTK -> password authentication acknowledgement
RPTC -> software endpoint configuration acknowledgement
RPTCL -> close temporary session
```

No DMR voice/data is transmitted.

The test is bounded and returns a plain-language result with one of these reason codes:

```text
ok
login
auth
config
timeout
network
unavailable
```

Examples:

- `login` — BrandMeister rejected the DMR/hotspot ID;
- `auth` — the Hotspot Security password was rejected;
- `config` — the software endpoint configuration was rejected;
- `timeout` — the master did not answer in time;
- `network` — DNS/socket/network path problem.

The password, challenge salt, and password hash are never returned.

### Software endpoint, not a fake radio

YWD-DMR is being built as a software DMR endpoint without an RF transmitter. For the temporary BrandMeister setup test it therefore sends neutral zero RX/TX frequency and power fields rather than pretending that it has an RF frequency.

If BrandMeister rejects that representation, the test should report `config`. We will adjust the software-endpoint representation based on real protocol behavior, not invent fake radio details to force a successful result.

### What is intentionally not available yet

There is still no network **commit** endpoint.

Even a successful BrandMeister test does not save the master or password. The required transaction remains:

```text
candidate -> local validation -> real master login/auth/config test -> durable commit
```

The known-good schema is still identity-only. Network persistence and its migration/rollback rules come only after the live test is proven on the Pi.

## Current safety note

The current LAN test build is development software. Keep it on a trusted local network only. Do not forward the YWD-DMR frontend port through a router or expose it directly to the public internet.

Production HTTPS/WSS, trusted reverse-proxy behavior, Secure-cookie deployment, long-lived BrandMeister networking, radio controls, and PTT are not finished.

For more detail, see [Setup and Security Phase](../developers/setup-security-phase.md), [BrandMeister Candidate Validation Notes](../developers/network-validation-notes.md), and [DMR Network Backend and BrandMeister Setup Contract](../developers/network-backend-contract.md).
