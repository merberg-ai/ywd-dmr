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

A candidate that only passes local field validation is **not** known-good. YWD-DMR must never report a successful network test merely because a hostname and password are syntactically valid.

The durable network commit remains closed until the real connectivity tester exists and passes.

## Current Alpha1 candidate

The current protected request shape is:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "password": "your BrandMeister hotspot password"
}
```

The station callsign, base DMR ID, and ESSID are **not** duplicated in this network object. They come from the already committed station identity. A backend can derive the network/device identity it needs from that authoritative record.

Fields:

- `backend` — currently only `brandmeister` is accepted.
- `master_address` — hostname or IP address only; no `http://`, `https://`, path, or embedded port.
- `master_port` — UDP master port. `0` normalizes to the BrandMeister/Homebrew default `62031`.
- `password` — BrandMeister hotspot password used by the real login test. It is never returned by a validation/test response.

## Protected validation API

Implemented on `dev`:

```text
POST /api/v1/setup/network/validate
```

This endpoint requires an Admin session and is subject to the global browser same-origin mutation rule even though it does not write durable configuration. The request contains a credential, so it is not a public pre-login form helper.

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

The password is deliberately replaced by the non-secret boolean `password_set`.

A syntactically valid JSON request with invalid fields returns HTTP `200` with `valid: false` and field errors. Malformed/unknown JSON still returns HTTP `400`. Missing/invalid authentication returns HTTP `401`; cross-origin browser mutations are rejected with HTTP `403` before the endpoint runs.

Current validation rejects:

- any backend other than BrandMeister;
- empty/invalid hostnames or IP addresses;
- URLs, embedded ports, paths, spaces, or malformed host labels;
- ports outside `1..65535` after default normalization;
- empty/all-whitespace passwords, passwords over 256 characters, or passwords containing control characters.

Validation does not write `known-good.json`, contact a master, or alter setup stage.

## Backend connectivity-test interface

`internal/dmrnet` now defines a backend-neutral tester contract. The real BrandMeister implementation will receive:

- the already known-good normalized station identity;
- the normalized network candidate including the password;
- a request context/deadline.

Its result is safe to expose and contains no password or challenge material.

Machine-readable result reasons are:

```text
ok
login
auth
config
timeout
network
unavailable
```

`login`, `auth`, and `config` deliberately follow the stages of the MMDVM/Homebrew master handshake so the WebUI can give useful troubleshooting instead of a generic failure.

## BrandMeister/Homebrew handshake target

The initial implementation target follows the established Homebrew DMR master handshake used by MMDVM/DMRGateway:

```text
RPTL + repeater/device ID
  <- RPTACK + salt

RPTK + repeater/device ID + SHA256(salt || password)
  <- RPTACK

RPTC + repeater/device ID + configuration data
  <- RPTACK

connected/test successful
```

A master `MSTNAK` during those phases can therefore be surfaced as:

- waiting for first acknowledgement -> `login`;
- waiting after `RPTK` -> `auth`;
- waiting after `RPTC` -> `config`.

No reply before the deadline is `timeout`; DNS/socket/local-network failures use `network`.

The protocol behavior and default UDP port `62031` are consistent with the current upstream G4KLX DMRGateway implementation (`DMRNetwork.cpp`) and its sample configuration.

## What the live test must not do

The setup tester is a short-lived credential/configuration probe, not the production network session. It must:

- use a bounded timeout;
- never transmit DMR voice/data;
- never log or return the hotspot password;
- never persist the candidate merely because the socket opened;
- close the test session cleanly after the master accepts login/auth/config;
- distinguish bad password from unreachable master where the protocol permits;
- leave the currently active known-good configuration untouched on every failure.

## Durable commit comes later

The current known-good schema stores only station identity. Network persistence will be added only after the real tester works. That change must include an explicit schema/migration decision rather than silently changing the meaning of schema 1.

When network persistence lands:

- the password may be replaced by a client but must never be returned by normal read/status APIs;
- the current known-good and previous rollback snapshots must continue to rotate atomically;
- setup state should advance from `identity_complete / network` only after a successful tested commit;
- failed network testing must not change the revision or previous snapshot.

## Current status

- [x] Backend-neutral network candidate model.
- [x] BrandMeister local validation and default port normalization.
- [x] Password-redacted normalized response.
- [x] Admin-protected `POST /api/v1/setup/network/validate`.
- [x] Backend-neutral connectivity-test result/reason interface.
- [x] Automated validation/API tests added on `dev`.
- [ ] Installed Pi validation of the protected network-validation endpoint.
- [ ] Real BrandMeister/Homebrew connectivity/authentication tester.
- [ ] Protected test endpoint using the real tester.
- [ ] Known-good schema migration for tested network configuration.
- [ ] Protected network commit endpoint.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
