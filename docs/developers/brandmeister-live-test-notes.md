# BrandMeister Live Test Notes

This page records the first installed Raspberry Pi 5 exercise of the real, non-persisting BrandMeister/Homebrew setup probe.

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

## First real BrandMeister contact

The first public-network test used:

```text
master: 3103.master.brandmeister.network
port:   62031
station base DMR ID: 3196104
ESSID: 02
derived Homebrew device ID: 319610402
```

The Hotspot Security password was entered locally and was not printed or pasted into development notes.

Result:

```json
{
  "ok": false,
  "backend": "brandmeister",
  "reason": "auth",
  "message": "BrandMeister rejected the hotspot security password.",
  "duration_ms": 250
}
```

This result is important because it proves the temporary tester reached the real master and completed the initial `RPTL` login stage. BrandMeister accepted the derived hotspot/device ID sufficiently to return the login acknowledgement and salt. The failure occurred specifically after YWD-DMR sent the `RPTK` authentication response.

## Upstream protocol cross-check

After the live `auth` result, the YWD-DMR `RPTK` construction was compared against the current G4KLX DMRGateway implementation.

The upstream implementation:

1. copies the four salt bytes directly from `RPTACK` offset 6;
2. appends the Hotspot Security password bytes;
3. SHA-256 hashes exactly `salt || password`;
4. sends `RPTK` + four-byte Homebrew device ID + the 32-byte SHA-256 digest.

YWD-DMR performs the same byte sequence. The current evidence therefore does not indicate a YWD-DMR wire-format defect at the authentication stage.

BrandMeister documentation also distinguishes the Hotspot Security password from the SelfCare login password. The Hotspot Security value is case-sensitive and is associated with the base Radio ID; alias/ESSID hotspot IDs use the same base-ID hotspot credential.

## Interpretation

Current status is:

```text
DNS/UDP path             PASS
RPTL device-ID login     PASS
RPTACK salt receipt      PASS
RPTK packet construction cross-checked against upstream
RPTK authentication      REJECTED BY MASTER
RPTC software config     NOT REACHED
RPTCL success close      NOT REACHED
network persistence      NOT ATTEMPTED
```

The next operator test should use the exact BrandMeister **Hotspot Security** password configured for base DMR ID `3196104`. It should not use the BrandMeister/SelfCare account-login password. If the Hotspot Security password is uncertain, reset it in BrandMeister SelfCare and then enter the new value into the YWD-DMR test locally.

A successful retry should move the probe past `auth` to either:

- `ok` — login, authentication, and software endpoint configuration all accepted; or
- `config` — login and Hotspot Security authentication succeeded, but BrandMeister rejected YWD-DMR's current zero-RF software-endpoint `RPTC` representation.

Either result would advance the protocol validation significantly.

## Safety result

The live public-network attempt did **not** persist the master, password, backend, or test result. It did not increment the known-good revision or create a rollback snapshot. No DMR voice/data frame was transmitted.

This preserves the Alpha1 transaction rule:

```text
candidate -> local validation -> real connectivity/authentication test -> commit
```

The durable network commit remains closed.
