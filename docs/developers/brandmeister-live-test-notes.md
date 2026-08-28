# BrandMeister Live Test Notes

This page records the installed Raspberry Pi 5 exercise of the real, non-persisting BrandMeister/Homebrew setup probe.

## Checkpoint

Initial installed source checkpoint:

```text
4483403478b638c9bec58cd40c2b62f7a2054898
```

Initial installed release:

```text
dev-4483403-20260824203604
```

The later registration-frequency retest used the `dev` implementation at checkpoint `90d33f0f04ee516aa199b37e054806cd1b203e55` or its installed build. The exact installed release timestamp was not copied into the validation notes.

Listener:

```text
0.0.0.0:8990
```

Each full test exercise cleaned the box back to fresh/unclaimed state with no known-good configuration.

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

The Hotspot Security authentication path is therefore real-network proven.

## First configuration hypothesis — zero frequency

YWD-DMR's original test `RPTC` packet was structurally correct and 302 bytes long, but deliberately reported:

```text
RX frequency: 000000000
TX frequency: 000000000
Power:        00
Slots:        4 (simplex/DMO)
```

Current BrandMeister hotspot guidance expects valid RX/TX frequency metadata for Homebrew/MMDVM registration. YWD-DMR therefore added explicit request-only `registration_frequency_hz` metadata. The compatibility packet used the same supplied frequency in both RX and TX fields, informational power `01`, color code `01`, and slot/mode marker `4`.

## Third real BrandMeister contact — valid frequency still rejected

The focused registration-frequency infrastructure gate passed. The live BrandMeister result remained:

```text
ok=false
reason=config
```

The test used:

```text
446525000 Hz
```

This eliminates all-zero RX/TX frequency fields as the sole cause.

## Fourth real BrandMeister contact — fresh ESSID also rejected at config

Using the LAN Admin Test Console, the operator committed:

```text
KJ6YWD / 3196104 / ESSID 03
```

which derives Homebrew device ID `319610403`. The identity commit succeeded as revision 2.

A local validation attempt also caught a units typo: `14742000` is 14.742 MHz, not 147.420 MHz; the correct API value is `147420000`.

A separate live BrandMeister test with a locally valid candidate and the verified Hotspot Security credential again returned `reason: config`. This makes an already-connected `319610402` session an unlikely explanation: both `319610402` and `319610403` authenticated and then failed at the same RPTC stage.

## Package/profile identifier probe

The next compatibility build changed only the final Homebrew package/profile field:

```text
software ID: YWD-DMR
package ID:  MMDVM_DMO
```

`MMDVM_DMO` is the upstream simplex MMDVM profile used with slot marker `4`. The live test still returned:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "config",
  "message": "BrandMeister rejected the Homebrew registration metadata after successful authentication.",
  "duration_ms": 291
}
```

Therefore the package/profile identifier alone was not the cause.

## Upstream numeric software/version identifier probe

Current upstream MMDVM-Host uses a date-style version string in the 40-byte software/version field. At the time of this probe the upstream value was:

```text
20260528
```

The focused probe changed only this identification field while keeping `MMDVM_DMO` as the package/profile:

```text
software ID: 20260528
package ID:  MMDVM_DMO
```

The real BrandMeister result was:

```json
{
  "ok": true,
  "backend": "brandmeister",
  "reason": "ok",
  "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
  "duration_ms": 341
}
```

This was the first real-master end-to-end setup success for YWD-DMR.

## YWD-DMR-owned numeric software/version identifier — accepted

The final isolation probe deliberately stopped using the exact upstream MMDVM-Host version and changed only the software/version field to a YWD-DMR-owned date-style identifier:

```text
software ID: 20260827
package ID:  MMDVM_DMO
```

Everything else remained unchanged, including:

```text
station:    KJ6YWD / 3196104 / ESSID 03
master:     3103.master.brandmeister.network
port:       62031
frequency:  446525000 Hz RX/TX registration metadata
power:      01
color code: 01
slots/mode: 4
```

BrandMeister accepted the complete temporary setup handshake:

```json
{
  "ok": true,
  "backend": "brandmeister",
  "reason": "ok",
  "message": "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration.",
  "duration_ms": 294
}
```

This is the decisive compatibility result. YWD-DMR does **not** need to claim an exact MMDVMHost release version. The real master accepts a YWD-DMR-owned numeric/date-style value in the Homebrew software/version field when the remaining simplex compatibility metadata is valid.

The proven temporary setup chain is now:

```text
DNS/UDP path                       PASS
RPTL device-ID login               PASS
RPTACK challenge/salt              PASS
RPTK Hotspot Security              PASS
RPTACK authentication              PASS
RPTC 302-byte registration         PASS
YWD-owned numeric software ID      PASS
MMDVM_DMO compatibility profile    PASS as part of accepted packet
RPTCL close                        SENT
network-test non-persistence       PASS
DMR voice/data transmission        NOT ATTEMPTED
network persistence                NOT ATTEMPTED
```

The earlier failures isolate the important compatibility behavior:

- arbitrary text `YWD-DMR` in the software/version field was rejected;
- changing only the package field to `MMDVM_DMO` was not enough;
- a numeric/date-style software/version field was accepted;
- a YWD-DMR-owned numeric/date-style value was accepted.

The short-lived tester may therefore use a YWD-DMR-owned numeric compatibility identifier without impersonating another project's exact version.

## Next gate — tested durable network configuration

The protocol-discovery gate is complete. Further packet-field experimentation should stop unless future real-master evidence requires it.

The next transaction is:

```text
candidate
  -> local validation
  -> real BrandMeister login/auth/config test
  -> durable commit of that exact tested candidate
```

The preferred API shape is a single protected **test-and-commit** operation rather than a reusable browser proof token. This keeps the exact normalized candidate in daemon memory from validation through the real network test and durable commit, avoiding a time-of-check/time-of-use mismatch.

The existing `/api/v1/setup/network/test` remains useful as a non-persisting diagnostic operation.

## Safety result

All public-network attempts to date were non-persisting. They did not save the master, password, backend, registration metadata, or test result; did not advance durable network state; and did not transmit DMR voice/data.

The Alpha1 transaction rule remains:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```
