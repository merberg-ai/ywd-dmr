# DMR Network Backend and BrandMeister Setup Contract

YWD-DMR treats DMR networking as a daemon-owned backend rather than wiring BrandMeister details directly into the WebUI. BrandMeister is the first backend, but the API/runtime model remains backend-neutral.

## Safety transaction

Network setup uses:

```text
candidate
   -> local validation
   -> real backend connectivity/authentication/configuration test
   -> durable known-good commit of that exact candidate
```

A candidate that only passes local field validation is **not** known-good. A UDP socket opening is not success either. For BrandMeister, the temporary test succeeds only after the master accepts `RPTL`, `RPTK`, and `RPTC`.

The Pi 5 has now proven that complete setup handshake against the public BrandMeister master.

## Candidate shape

```json
{
  "backend": "brandmeister",
  "master_address": "3103.master.brandmeister.network",
  "master_port": 62031,
  "registration_frequency_hz": 446525000,
  "password": "your BrandMeister hotspot password"
}
```

Station callsign/base DMR ID/ESSID come from already committed known-good identity.

Fields:

- `backend` — currently only `brandmeister`.
- `master_address` — hostname/IP only; no URL, path, or embedded port.
- `master_port` — UDP port; `0` normalizes to `62031`.
- `registration_frequency_hz` — Homebrew registration metadata from `100000000..999999999` Hz. Simplex registration reports the same value as RX and TX. It does not key RF.
- `password` — BrandMeister Hotspot Security credential, separate from the SelfCare login password; non-empty, at most 20 characters, no control characters.

Passwords are accepted only as request input and daemon-internal secret state. They are never returned.

## Local validation

```text
POST /api/v1/setup/network/validate
```

Admin-only and browser same-origin protected. It performs no UDP I/O and no persistence.

A successful response replaces the password with `password_set`:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "3103.master.brandmeister.network",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  },
  "errors": []
}
```

## Non-persisting live diagnostic

```text
POST /api/v1/setup/network/test
```

This remains useful for protocol/credential troubleshooting. It validates locally, reads identity from known-good state, runs the bounded real Homebrew setup handshake, closes the temporary session, and returns a structured result.

It must not increment a revision, create/rotate configuration snapshots, save the master/password, or advance setup state.

Normal result reasons remain:

```text
ok
login
auth
config
timeout
network
unavailable
```

## Tested durable commit

The durable setup operation is:

```text
POST /api/v1/setup/network/test-and-commit
```

It uses the same candidate body.

The operation deliberately combines test and commit in one protected request:

```text
read candidate
  -> normalize/validate
  -> read known-good identity
  -> real BrandMeister test using that candidate
  -> if test is not ok: committed=false, stop
  -> if test is ok: persist that same normalized candidate
```

No browser proof token is issued. This prevents a tested candidate from being swapped for an untested candidate before commit.

Successful response example:

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
    "backend": "brandmeister",
    "reason": "ok",
    "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
    "duration_ms": 294
  }
}
```

The password is absent.

If the real test returns a normal failure, HTTP remains `200` with:

```json
{
  "committed": false,
  "test": {
    "ok": false,
    "backend": "brandmeister",
    "reason": "auth"
  }
}
```

Known-good state remains untouched in that case.

## Durable schema and secret storage

Schema 1 remains identity-only. A successful tested network commit creates schema 2 with non-secret network metadata and `password_set: true`.

The actual Hotspot Security password is stored separately under the daemon-owned state tree in a mode-`0600` revision-bound secret file. The containing secret directory is mode `0700`.

The configuration revision and secret revision must match. Rollback therefore restores matching settings and credentials rather than combining a previous master/frequency with a newer password.

See [Known-good Configuration Store](configuration-store.md).

## Proven BrandMeister/Homebrew handshake

The temporary tester performs:

```text
RPTL + device ID
  <- RPTACK + 4-byte salt

RPTK + device ID + SHA256(salt || password)
  <- RPTACK

RPTC + device ID + 294-byte configuration block
  <- RPTACK

RPTCL + device ID
```

No `DMRD` voice/data is sent by setup testing.

Device ID:

```text
ESSID 0     -> base DMR ID
ESSID 1..99 -> (base DMR ID * 100) + ESSID
```

## RPTC compatibility profile

Homebrew `RPTC` is fixed at 302 bytes. The accepted Alpha1 setup packet uses:

- committed station callsign;
- operator registration frequency in both RX/TX fields;
- informational power `01`;
- color code `01`;
- zero latitude/longitude/height until location setup is added;
- `YWD-DMR software` location text;
- `Software DMR client` description;
- slot/mode marker `4`;
- YWD-DMR project URL;
- a YWD-DMR-owned numeric/date-style software/version identifier;
- final package/profile identifier `MMDVM_DMO`.

`MMDVM_DMO` is used as the Homebrew simplex compatibility profile. It does **not** mean YWD-DMR owns or controls an MMDVM modem.

### Why the software field is numeric

Real-master isolation testing found:

```text
software ID YWD-DMR + package YWD-DMR       -> config rejected
software ID YWD-DMR + package MMDVM_DMO     -> config rejected
software ID 20260528 + package MMDVM_DMO    -> accepted
software ID 20260827 + package MMDVM_DMO    -> accepted
```

The final result is important: BrandMeister accepted a **YWD-DMR-owned** numeric/date-style software identifier. YWD-DMR therefore does not need to claim another project's exact release version.

The protocol field is compatibility metadata. Future releases can formalize how a suitable numeric YWD-DMR identifier is generated while keeping the product identity honest in normal UI/API/version reporting.

## Retry and timeout behavior

Each setup handshake stage sends once and waits roughly 1.5 seconds, with a 10-second HTTP-level deadline. Homebrew `RPTACK` does not identify the stage it acknowledges, so automatic UDP retries could allow delayed duplicate ACKs to create false success. For setup, a false timeout is safer; the user can simply run Test again.

## Automated coverage

The backend/tests cover:

- base-ID/ESSID device-ID derivation;
- exact `RPTL` framing;
- salt handling;
- SHA-256 `salt || password` RPTK construction;
- fixed 302-byte RPTC framing;
- frequency metadata;
- power/color/slot profile;
- numeric software ID and `MMDVM_DMO` profile;
- no password leakage into RPTC/results;
- explicit RPTCL close;
- auth/config/timeout classification;
- local-candidate rejection before UDP;
- Admin authorization and browser same-origin protection;
- failed test-and-commit leaves known-good state unchanged;
- successful test-and-commit creates schema 2 only after real-test success.

## Current status

- [x] Backend-neutral candidate/test result model.
- [x] Protected local validation.
- [x] Protected non-persisting live test.
- [x] Real BrandMeister RPTL accepted.
- [x] Real BrandMeister Hotspot Security authentication accepted.
- [x] Real RPTC configuration accepted.
- [x] YWD-owned numeric software/version identifier accepted.
- [x] Explicit schema-2 network persistence design implemented on `dev`.
- [x] Revision-bound network secret storage implemented on `dev`.
- [x] Protected one-request test-and-commit endpoint implemented on `dev`.
- [ ] Pi 5 installed validation of test-and-commit/restart/rollback.
- [ ] Long-lived BrandMeister backend.
- [ ] Connection/reconnect state machine.
- [ ] Talkgroup destination model.
