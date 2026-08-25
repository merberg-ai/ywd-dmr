# DMR Network Backend and BrandMeister Setup Contract

YWD-DMR treats DMR networking as a daemon-owned backend rather than wiring BrandMeister details directly into the WebUI. BrandMeister is the first backend, but the API and runtime must be able to support another DMR network later without inventing a second control model.

## Safety rule

Network setup uses a stricter transaction than station identity:

```text
candidate
   -> local validation
   -> real backend connectivity/authentication test
   -> durable known-good commit
```

A candidate that only passes local field validation is **not** known-good. A successful UDP socket open is also not enough. The network test is successful only after the BrandMeister/Homebrew master accepts login, password authentication, and the software endpoint configuration.

The durable network commit remains closed until this real tester is validated on the installed appliance.

## Current Alpha1 candidate

The protected request shape is:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your BrandMeister hotspot password"
}
```

The station callsign, base DMR ID, and ESSID are **not** duplicated in this network object. They come from the already committed station identity.

Fields:

- `backend` — currently only `brandmeister` is accepted.
- `master_address` — hostname or IP address only; no `http://`, `https://`, path, or embedded port.
- `master_port` — UDP master port. `0` normalizes to the Homebrew default `62031`.
- `password` — BrandMeister Hotspot Security password. It is accepted only as request input and is never returned by validation/test responses.

## Protected local validation API

Implemented and Pi 5 validated:

```text
POST /api/v1/setup/network/validate
```

This endpoint requires an Admin session and is subject to the global browser same-origin mutation rule. The request contains a credential, so it is not a public pre-login form helper.

Example successful response:

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

The password is deliberately replaced by `password_set`.

A syntactically valid JSON request with invalid fields returns HTTP `200` with `valid: false` and field errors. Malformed/unknown JSON returns HTTP `400`. Missing/invalid authentication returns HTTP `401`; cross-origin browser mutations return HTTP `403` before the endpoint runs.

The installed Raspberry Pi 5 validation proved normalization, password redaction, strict JSON/method behavior, authorization/origin protection, and that validation does not change the known-good revision or create a rollback snapshot. See [BrandMeister Candidate Validation Notes](network-validation-notes.md).

## Protected live test API

Implemented on `dev` and awaiting installed-machine validation:

```text
POST /api/v1/setup/network/test
```

The request body is the same network candidate used by `/validate`.

Before any UDP packet is sent, the endpoint requires:

1. a live Admin session;
2. a locally valid network candidate;
3. an already committed, readable station identity;
4. the real network tester service.

A missing station identity returns HTTP `409`. Invalid local fields return HTTP `400` and the tester does not run. Browser same-origin protection applies before the handler.

The endpoint has a 10-second overall deadline. Normal network-test outcomes use HTTP `200` and a password-free body such as:

```json
{
  "ok": true,
  "backend": "brandmeister",
  "reason": "ok",
  "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
  "duration_ms": 123
}
```

A failed live test still returns a structured, non-secret result. Machine-readable reasons are:

```text
ok
login
auth
config
timeout
network
unavailable
```

These mean:

- `login` — the master rejected the initial DMR/hotspot ID login;
- `auth` — the master rejected the Hotspot Security password response;
- `config` — the master rejected the endpoint configuration, or the stored station identity cannot be represented in the Homebrew config block;
- `timeout` — no acceptable reply arrived before the bounded deadline;
- `network` — DNS/socket/UDP path failure;
- `unavailable` — the master closed the test session or returned an unusable acknowledgement.

No result contains the password, salt, SHA-256 response, or other challenge material.

## BrandMeister/Homebrew probe

The Alpha1 tester performs the established Homebrew handshake:

```text
RPTL + device ID
  <- RPTACK + 4-byte salt

RPTK + device ID + SHA256(salt || password)
  <- RPTACK

RPTC + device ID + 294-byte configuration block
  <- RPTACK

RPTCL + device ID
```

The tester never sends a `DMRD` voice/data frame.

For BrandMeister device-ID derivation:

```text
ESSID 0     -> base DMR ID
ESSID 1..99 -> (base DMR ID * 100) + ESSID
```

For example, a base ID ending in seven digits with ESSID `01` becomes the usual nine-digit hotspot form. This derivation stays in the backend rather than changing the canonical station identity.

## Software-endpoint RPTC configuration

Homebrew `RPTC` is a fixed 302-byte packet. YWD-DMR is a software DMR endpoint with no RF transmitter, so the temporary test does not invent RF characteristics. Its test configuration uses:

- committed station callsign;
- RX frequency `000000000`;
- TX frequency `000000000`;
- power `00`;
- color code `01` as a neutral protocol field;
- zero latitude/longitude/height;
- `YWD-DMR software` location text;
- `Software DMR client` description;
- slot/mode marker `4` for simplex/DMO-style endpoint behavior;
- YWD-DMR project/software identifiers.

The Homebrew callsign field is eight characters wide. The generic station model remains broader for future backends, but the BrandMeister test returns `reason: config` if the stored callsign cannot fit that protocol field rather than silently truncating it.

Whether BrandMeister accepts the neutral zero-RF software configuration is deliberately a real-machine test question. We will not replace those fields with fake amateur-radio frequencies merely to make a test pass.

## Retry and timeout behavior

Each handshake stage is short and bounded. The current tester sends each stage at most twice with a roughly 1.5-second per-attempt wait, while the HTTP handler imposes a 10-second overall deadline.

A master `MSTNAK` is mapped according to the stage being attempted:

- after `RPTL` -> `login`;
- after `RPTK` -> `auth`;
- after `RPTC` -> `config`.

A master `MSTCL` becomes `unavailable`. Unrelated packets are ignored until the current stage deadline rather than being mistaken for success.

## Automated protocol tests

The real tester is exercised against local UDP test masters, not the public network. Automated coverage verifies:

- base-ID and ESSID hotspot-ID derivation;
- exact 8-byte `RPTL` framing;
- four-byte salt handling;
- `RPTK` SHA-256 construction from `salt || password`;
- fixed 302-byte `RPTC` framing;
- software/zero-RF configuration fields;
- no password in the RPTC packet;
- explicit `RPTCL` close after success;
- `MSTNAK` during authentication maps to `reason: auth`;
- bounded timeout behavior;
- over-width Homebrew callsign returns `reason: config`.

HTTP tests additionally verify that `/network/test` requires committed identity, rejects invalid/cross-origin candidates before invoking the tester, passes the known-good identity to the tester, does not echo the password, and does not advance the known-good revision.

## What the live test must not do

The setup tester is a credential/configuration probe, not the production network session. It must:

- never transmit DMR voice/data;
- never log or return the hotspot password;
- never persist a network candidate;
- never treat a socket open or local validation as successful connectivity;
- close the test session after the master accepts configuration;
- leave known-good configuration untouched on success **and** failure.

## Durable commit comes later

The current known-good schema stores only station identity. Network persistence will be added only after the real BrandMeister tester works on the installed Pi. That change requires an explicit schema/migration decision rather than silently changing schema 1.

When network persistence lands:

- the password may be replaced by a client but must never be returned by normal read/status APIs;
- current and previous snapshots must continue to rotate atomically;
- setup state advances beyond `identity_complete / network` only after a successful tested commit;
- failed network testing must not change the revision or rollback snapshot.

## Current status

- [x] Backend-neutral network candidate model.
- [x] BrandMeister local validation and default port normalization.
- [x] Password-redacted normalized response.
- [x] Admin-protected `POST /api/v1/setup/network/validate`.
- [x] Installed Pi validation of protected network candidate validation.
- [x] Backend-neutral connectivity-test result/reason interface.
- [x] Real bounded BrandMeister/Homebrew `RPTL -> RPTK -> RPTC -> RPTCL` tester implementation.
- [x] Local UDP protocol tests for framing, auth rejection, timeout, and secret handling.
- [x] Admin-protected `POST /api/v1/setup/network/test` wired to the real tester.
- [ ] Installed Pi source/runtime validation of the live test endpoint.
- [ ] Successful real BrandMeister handshake with operator credentials.
- [ ] Known-good schema migration for tested network configuration.
- [ ] Protected network commit endpoint.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
