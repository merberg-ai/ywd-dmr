# LAN Admin Test Console

The development WebUI includes an **Admin Test Console** so repeated YWD-DMR setup and BrandMeister testing can be performed from a browser instead of long shell/curl blocks.

This is a development convenience, not a security bypass. The console uses the same `/api/v1/` endpoints, authentication, role checks, browser same-origin protection, validation rules, and durable configuration transaction as any future client.

## Safety boundary

The current YWD-DMR frontend is a **LAN test build**.

Do not:

- router-forward the YWD-DMR listener;
- expose it directly to the public internet;
- treat the current HTTP listener as production remote access.

Future public access requires the planned HTTPS/WSS and trusted-proxy deployment contract.

The console does **not** provide shell access, sudo access, arbitrary file operations, or an API that bypasses normal authorization.

## What the console can do

The console currently provides browser controls for:

1. reading daemon health and setup status;
2. claiming a fresh appliance with the one-time local claim code;
3. Admin login and logout;
4. committing station identity;
5. locally validating a BrandMeister candidate;
6. running the real short-lived non-persisting BrandMeister test;
7. testing and durably committing the exact same accepted network candidate;
8. displaying structured password-redacted results;
9. refreshing setup/session state.

## Claiming a fresh development install

Retrieve the one-time code locally:

```bash
sudo ywd-dmr claim-code
```

Paste it into **Claim fresh appliance**, choose the Admin username/password, and press **Claim & Sign In**.

## Password handling

The console:

- uses browser password inputs;
- does not write passwords to Local Storage or Session Storage;
- does not put passwords in URLs;
- does not print submitted passwords into the result panel;
- clears Admin passwords after claim/login attempts;
- clears the BrandMeister password after live test or test-and-commit;
- applies a client-side redaction pass before displaying responses.

The daemon remains authoritative. Server responses do not echo the Hotspot Security password, challenge salt, or authentication digest.

After a successful durable network commit, the Hotspot Security password is stored only in the daemon's restricted revision-bound secret store. Normal known-good JSON contains only `password_set: true`.

## Station identity

```text
POST /api/v1/setup/identity/commit
```

Admin-only. Callsign, base DMR ID, and ESSID are committed through the known-good configuration path.

Once tested network configuration exists, a later identity commit preserves that network and its credential in the new revision instead of silently dropping it.

## BrandMeister candidate

Fields:

```text
master hostname/IP
UDP port
registration frequency in Hz
Hotspot Security password
```

Registration frequency is Homebrew metadata only; YWD-DMR still does not key RF.

The field expects **Hz**, not MHz:

```text
147.420 MHz -> 147420000 Hz
446.525 MHz -> 446525000 Hz
```

## Validate Candidate

```text
POST /api/v1/setup/network/validate
```

Local validation/normalization only. No network I/O and no persistence.

## Run Live BM Test

```text
POST /api/v1/setup/network/test
```

Runs a fresh bounded real Homebrew login/auth/config handshake and then closes the temporary session. It sends no `DMRD` voice/data and changes no durable network state.

This remains the preferred diagnostic button when you only want to check credentials/connectivity.

## Test & Commit Network

```text
POST /api/v1/setup/network/test-and-commit
```

This button is intentionally more explicit because it changes durable configuration.

The browser asks for confirmation, then the daemon performs:

```text
candidate
  -> local validation
  -> real BrandMeister login/auth/config test
  -> only if accepted: durable schema-2 commit
```

The test and commit happen in **one request**. There is no reusable "test passed" token that could accidentally approve a changed form.

If BrandMeister rejects the candidate, the result shows:

```json
{
  "committed": false,
  "test": {
    "ok": false,
    "reason": "auth"
  }
}
```

and the existing known-good revision remains unchanged.

On success the result includes the new revision and only the non-secret network summary:

```json
{
  "committed": true,
  "revision": 2,
  "network": {
    "backend": "brandmeister",
    "master_address": "3103.master.brandmeister.network",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "test": {
    "ok": true,
    "reason": "ok"
  }
}
```

After success, setup status advances to approximately:

```text
stage: network_complete
next step: audio
```

The dashboard Network card shows **CONFIGURED** even though the long-lived BrandMeister runtime is not connected yet. **CONFIGURED is not the same as CONNECTED.**

## Current proven Homebrew compatibility

The Pi 5 has now produced a real `ok` result with:

```text
RPTL login                         PASS
RPTK Hotspot Security              PASS
RPTC configuration                 PASS
YWD-owned numeric software ID      PASS
MMDVM_DMO compatibility profile    PASS
RPTCL close                        SENT
```

No voice/data was transmitted by the setup test.

## Result panel

Typical network test reasons:

```text
ok
login
auth
config
timeout
network
unavailable
```

The console now treats an HTTP-200 response containing `ok:false`, `valid:false`, or `committed:false` as an unsuccessful operation visually rather than coloring every HTTP 200 as success.

## What is intentionally missing

There is still no browser button to wipe security state, delete known-good configuration, restart services, run shell commands, or perform an arbitrary appliance reset.

Those operations need explicit daemon-side contracts and strong development/LAN gating before they belong in the UI.

## Relationship to the future production WebUI

This console is scaffolding for proving the real daemon APIs. Proven pieces can later become the guided setup wizard while developer diagnostics remain clearly separated.
