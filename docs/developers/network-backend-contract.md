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

A candidate that only passes local field validation is **not** known-good. A successful UDP socket open is also not enough. The network test is successful only after the BrandMeister/Homebrew master accepts login, password authentication, and the endpoint configuration.

The durable network commit remains closed until a complete real BrandMeister handshake succeeds on the installed appliance.

## Current Alpha1 candidate

The protected request shape is:

```json
{
  "backend": "brandmeister",
  "master_address": "master.example.net",
  "master_port": 62031,
  "registration_frequency_hz": 446525000,
  "password": "your BrandMeister hotspot password"
}
```

The station callsign, base DMR ID, and ESSID are **not** duplicated in this network object. They come from the already committed station identity.

Fields:

- `backend` — currently only `brandmeister` is accepted.
- `master_address` — hostname or IP address only; no `http://`, `https://`, path, or embedded port.
- `master_port` — UDP master port. `0` normalizes to the Homebrew default `62031`.
- `registration_frequency_hz` — nominal Homebrew/BrandMeister registration frequency. The current Alpha1 API accepts `100000000..999999999` Hz. For simplex/DMO registration the tester reports the same value as RX and TX. This is registration metadata only; it does not create an RF transmit path.
- `password` — BrandMeister Hotspot Security password. Current BrandMeister guidance limits it to 20 characters. YWD-DMR rejects empty, over-20-character, and control-character-containing passwords before any live test.

The password is accepted only as request input and is never returned by validation/test responses.

## Why a registration frequency exists in a software endpoint

The first live authenticated BrandMeister test proved that the master accepted `RPTL` and `RPTK` but rejected YWD-DMR's original `RPTC` packet when it reported zero RX/TX frequency and zero power.

BrandMeister's hotspot guidance expects valid frequency metadata for Homebrew/MMDVM clients. For simplex hotspots it specifically instructs operators to use the same valid UHF amateur frequency in both receive and transmit fields.

YWD-DMR therefore asks the operator for a nominal registration frequency instead of silently inventing one. The value is used only in Homebrew registration metadata. YWD-DMR still has no RF transmitter in this phase.

The test `RPTC` also uses informational power `01` instead of zero. This is a protocol-compatibility metadata value, not a claim that YWD-DMR is transmitting one watt.

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
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "errors": []
}
```

The password is deliberately replaced by `password_set`.

A syntactically valid JSON request with invalid fields returns HTTP `200` with `valid: false` and field errors. Malformed/unknown JSON returns HTTP `400`. Missing/invalid authentication returns HTTP `401`; cross-origin browser mutations return HTTP `403` before the endpoint runs.

The installed Raspberry Pi 5 validation proved normalization, password redaction, strict JSON/method behavior, authorization/origin protection, and that validation does not change the known-good revision or create a rollback snapshot.

## Protected live test API

Implemented and installed-machine validated for its safety/control behavior:

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

No result contains the password, salt, SHA-256 response, or other challenge material.

## Real BrandMeister results so far

Installed release `dev-4483403-20260824203604` proved the live-test safety boundary and reached the public master at `3103.master.brandmeister.network:62031` using base DMR ID `3196104`, ESSID `02`, and derived device ID `319610402`.

The first credential attempt reached `RPTK` and was classified correctly as `auth`.

A retry with the verified Hotspot Security credential returned:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "config",
  "duration_ms": 289
}
```

That proves the public master accepted:

```text
RPTL device-ID login   PASS
RPTACK challenge       PASS
RPTK password response PASS
RPTACK authentication  PASS
RPTC zero-RF config    REJECTED
```

The authentication path is therefore real-network proven. Both live attempts left known-good revision 1 unchanged, created no rollback snapshot, and transmitted no DMR voice/data.

See [BrandMeister Live Test Notes](brandmeister-live-test-notes.md).

## BrandMeister/Homebrew probe

The Alpha1 tester performs:

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

## Current RPTC registration metadata

Homebrew `RPTC` is a fixed 302-byte packet. The current tester uses:

- committed station callsign;
- operator-supplied registration frequency for both RX and TX;
- informational power `01`;
- color code `01`;
- zero latitude/longitude/height until real station-location settings are added;
- `YWD-DMR software` location text;
- `Software DMR client` description;
- slot/mode marker `4` for simplex/DMO-style registration;
- YWD-DMR project/software identifiers.

The Homebrew callsign field is eight characters wide. If the stored generic station callsign cannot fit, the test returns `reason: config` rather than silently truncating it.

## Retry and timeout behavior

Each handshake stage is short and bounded. The setup tester sends each stage **once** and waits roughly 1.5 seconds for that stage's acknowledgement, while the HTTP handler imposes a 10-second overall deadline.

This is intentionally conservative. Homebrew `RPTACK` packets do not identify which handshake stage they acknowledge. Retrying a UDP stage could leave a delayed duplicate acknowledgement in the socket and create ambiguity during the next stage. For setup validation, a false timeout is safer than a false success; the operator can simply run the Test action again.

## Automated protocol tests

The tester is exercised against local UDP test masters. Coverage verifies:

- base-ID and ESSID hotspot-ID derivation;
- exact 8-byte `RPTL` framing;
- four-byte salt handling;
- `RPTK` SHA-256 construction from `salt || password`;
- fixed 302-byte `RPTC` framing;
- operator registration frequency copied into both RX/TX fields;
- informational power `01` and simplex/DMO marker `4`;
- no password in the RPTC packet;
- explicit `RPTCL` close after success;
- auth rejection mapping;
- bounded timeout behavior;
- missing/invalid registration-frequency rejection;
- over-width Homebrew callsign rejection.

HTTP tests additionally verify that `/network/test` requires committed identity, rejects invalid/cross-origin candidates before invoking the tester, passes the known-good identity plus registration frequency to the tester, does not echo the password, and does not advance the known-good revision.

## What the live test must not do

The setup tester is a credential/configuration probe, not the production network session. It must:

- never transmit DMR voice/data;
- never log or return the hotspot password;
- never persist a network candidate;
- never treat a socket open or local validation as successful connectivity;
- close the test session after the master accepts configuration;
- leave known-good configuration untouched on success **and** failure.

## Durable commit comes later

The current known-good schema stores only station identity. Network persistence will be added only after a complete real BrandMeister handshake succeeds on the installed Pi. That change requires an explicit schema/migration decision rather than silently changing schema 1.

## Current status

- [x] Backend-neutral network candidate model.
- [x] BrandMeister local validation and default port normalization.
- [x] Password-redacted normalized response.
- [x] Admin-protected `POST /api/v1/setup/network/validate`.
- [x] Installed Pi validation of protected network candidate validation.
- [x] Backend-neutral connectivity-test result/reason interface.
- [x] Real bounded BrandMeister/Homebrew tester implementation.
- [x] Admin-protected `POST /api/v1/setup/network/test`.
- [x] Installed Pi source/runtime validation of the live test endpoint and non-persistence rules.
- [x] Real BrandMeister device-ID login accepted.
- [x] Real BrandMeister Hotspot Security authentication accepted.
- [x] Zero-frequency RPTC rejection isolated to the configuration stage.
- [x] Explicit operator registration-frequency metadata added to the candidate/RPTC contract.
- [ ] Installed Pi validation of the registration-frequency RPTC update.
- [ ] Complete real BrandMeister `RPTC` acceptance / `ok` result.
- [ ] Known-good schema migration for tested network configuration.
- [ ] Protected network commit endpoint.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
