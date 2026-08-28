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

The retry again proved that the live test is non-persisting:

- known-good revision 1 remained byte-for-byte unchanged;
- no previous/rollback snapshot was created;
- cleanup restored fresh unclaimed/missing-config state;
- final daemon health passed.

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

## Second configuration hypothesis — package/profile identifier

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

Therefore the package/profile identifier alone is not the cause.

## Third configuration hypothesis — software/version identifier

Current upstream MMDVM-Host uses a date-style version string in the 40-byte software/version field. At the time of this probe the upstream `Version.h` value is:

```text
20260528
```

YWD-DMR's temporary connectivity tester now changes only this remaining identification field while keeping `MMDVM_DMO` as the package/profile:

```text
software ID: 20260528
package ID:  MMDVM_DMO
```

This is a narrow interoperability experiment in the short-lived tester only. It is not the product identity contract for the future long-lived YWD-DMR backend, and it does not mean `ywd-dmrd` owns or keys an MMDVM modem.

At this point the tested chain is:

```text
DNS/UDP path                      PASS
RPTL device-ID login              PASS
RPTACK salt receipt               PASS
RPTK wire construction            CROSS-CHECKED
RPTK authentication               PASS
RPTC packet length/layout         MATCHES 302-BYTE HOMEBREW FORMAT
RPTC with zero frequency          REJECTED
RPTC with 446525000 Hz RX/TX      REJECTED
RPTC on fresh ESSID 03            REJECTED
ESSID conflict hypothesis         UNLIKELY
MMDVM_DMO package-profile probe   REJECTED
MMDVM software/version ID probe   NEXT
network-test non-persistence      PASS
DMR voice/data transmission       NOT ATTEMPTED
network persistence               NOT ATTEMPTED
```

## Safety result

All public-network attempts remain non-persisting. They do not save the master, password, backend, registration metadata, or test result; do not advance durable network state; and do not transmit DMR voice/data.

The Alpha1 transaction rule remains:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

The durable network commit remains closed until the real setup test reaches `ok`.
