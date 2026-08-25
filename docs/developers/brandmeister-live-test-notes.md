# BrandMeister Live Test Notes

This page records the installed Raspberry Pi 5 exercise of the real, non-persisting BrandMeister/Homebrew setup probe.

## Checkpoint

Installed source checkpoint:

```text
4483403478b638c9bec58cd40c2b62f7a2054898
```

Installed release:

```text
dev-4483403-20260824203604
```

Listener:

```text
0.0.0.0:8990
```

The test box began fresh/unclaimed with no known-good configuration.

## Local infrastructure result

The live test infrastructure passed its installed-machine safety gate:

- fresh one-time claim succeeded;
- `/api/v1/setup/network/test` before station identity returned HTTP `409` and did not run the network probe;
- real station identity `KJ6YWD / 3196104 / ESSID 02` committed as known-good revision 1;
- cross-origin live test returned HTTP `403` before UDP work;
- reserved `.invalid` master returned a structured `network` result rather than an HTTP/server failure;
- failed test left the revision-1 known-good file byte-for-byte unchanged;
- no `known-good.previous.json` rollback snapshot was created;
- setup remained `identity_complete / network / revision 1`;
- schema 1 remained identity-only;
- cleanup restored fresh unclaimed/missing-config state and regenerated the bootstrap claim code;
- final daemon health passed.

The `.invalid` test result was:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "network",
  "message": "Could not resolve or open a UDP path to the BrandMeister master.",
  "duration_ms": 336
}
```

## First real BrandMeister contact — authentication rejection

The public-network tests used:

```text
master: 3103.master.brandmeister.network
port:   62031
station base DMR ID: 3196104
ESSID: 02
derived Homebrew device ID: 319610402
```

The Hotspot Security password was entered locally and was not printed or pasted into development notes.

The first attempt returned:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "auth",
  "duration_ms": 250
}
```

That proved the temporary tester reached the real master, completed `RPTL`, and received the challenge salt. The failure occurred after `RPTK`.

## Authentication wire-format cross-check

The YWD-DMR `RPTK` construction was compared against the current G4KLX DMRGateway implementation.

Both implementations:

1. copy the four salt bytes directly from `RPTACK` offset 6;
2. append the Hotspot Security password bytes;
3. SHA-256 hash exactly `salt || password`;
4. send `RPTK` + four-byte Homebrew device ID + the 32-byte SHA-256 digest.

BrandMeister documentation also distinguishes Hotspot Security from the SelfCare account-login password. The Hotspot Security value is case-sensitive and is associated with the base Radio ID; alias/ESSID hotspot IDs use the same base-ID hotspot credential.

## Second real BrandMeister contact — authentication accepted

The operator retried with the verified Hotspot Security credential. Result:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "config",
  "message": "BrandMeister rejected the software-endpoint configuration.",
  "duration_ms": 289
}
```

This is a major protocol milestone. `reason: config` is only possible after BrandMeister has accepted:

```text
RPTL device-ID login       PASS
RPTACK challenge/salt      PASS
RPTK password response     PASS
RPTACK authentication      PASS
RPTC configuration         REJECTED
```

The Hotspot Security authentication path is therefore now real-network proven.

The retry again proved that the live test is non-persisting:

- known-good revision 1 remained byte-for-byte unchanged;
- no previous/rollback snapshot was created;
- cleanup restored fresh unclaimed/missing-config state;
- final daemon health passed.

## Configuration rejection analysis

YWD-DMR's original test `RPTC` packet was structurally correct and 302 bytes long, but deliberately reported:

```text
RX frequency: 000000000
TX frequency: 000000000
Power:        00
Slots:        4 (simplex/DMO)
```

Current BrandMeister hotspot guidance expects valid RX/TX frequency metadata for Homebrew/MMDVM registration. OpenSpot guidance specifically tells simplex users to enter the same valid UHF amateur frequency in both receive and transmit fields.

The next YWD-DMR slice therefore changes the network candidate to include:

```text
registration_frequency_hz
```

This is explicit Homebrew/BrandMeister registration metadata. It does **not** mean YWD-DMR has gained an RF transmitter or that the daemon will key RF.

For the compatibility `RPTC` packet YWD-DMR will:

- place the operator-supplied registration frequency in both RX and TX fields;
- use informational power `01` rather than zero;
- retain color code `01`;
- retain slot/mode marker `4` for simplex/DMO-style registration;
- retain zero location/height unless real station-location settings are added later;
- still send no `DMRD` voice/data during the setup test.

No frequency is silently invented by the daemon. The operator supplies the nominal registration frequency explicitly.

## Current interpretation

Current status is:

```text
DNS/UDP path                 PASS
RPTL device-ID login         PASS
RPTACK salt receipt          PASS
RPTK wire construction       CROSS-CHECKED
RPTK authentication          PASS
RPTC zero-frequency metadata REJECTED
network-test non-persistence PASS
DMR voice/data transmission  NOT ATTEMPTED
network persistence          NOT ATTEMPTED
```

The next installed-machine gate is a focused live handshake using a real registration frequency in the `RPTC` metadata. If BrandMeister returns `ok`, the complete login/auth/config probe is proven and network-schema/commit work may begin. If it still returns `config`, the remaining RPTC fields need to be narrowed further.

## Safety result

Neither public-network attempt persisted the master, password, backend, registration metadata, or test result. Neither incremented the known-good revision or created a rollback snapshot. No DMR voice/data frame was transmitted.

This preserves the Alpha1 transaction rule:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

The durable network commit remains closed.
