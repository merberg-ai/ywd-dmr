# Known-good Configuration Store

YWD-DMR does not treat every submitted settings form as active configuration. The daemon keeps one **known-good** configuration and one previous rollback snapshot so a bad candidate cannot casually destroy a working station setup.

## Where it lives

The development store uses files under the daemon-owned state directory, planned as:

```text
/var/lib/ywd-dmr/known-good.json
/var/lib/ywd-dmr/known-good.previous.json
```

These files are different from `/etc/ywd-dmr/ywd-dmr.env`. The `/etc` file contains service/bootstrap settings such as the frontend listener. Radio/network/audio configuration belongs to daemon-managed persistent state under `/var/lib/ywd-dmr/`.

The configuration snapshots are written with mode `0600`. The containing state directory remains restricted to the `ywd-dmr` service account.

## Current schema

Schema 1 currently contains only the normalized radio identity:

```json
{
  "schema": 1,
  "revision": 1,
  "identity": {
    "callsign": "N0CALL",
    "dmr_id": 1234567,
    "essid": 1
  }
}
```

`schema` and `revision` are daemon-owned. A setup client submits candidate values; it does not choose either field.

Future schema revisions will add network, audio, vocoder, and other station configuration without changing the rule that clients submit candidates and the daemon decides what may become known-good.

## Commit rules

A candidate follows this direction:

```text
client candidate
      |
      v
server-side validation
      |
      +-- invalid -> reject, current known-good remains untouched
      |
      v
normalize
      |
      v
save current as previous rollback snapshot
      |
      v
atomic replace of current known-good snapshot
```

The file implementation writes a temporary file in the same directory, sets restrictive permissions, writes and syncs the contents, then renames it over the destination and syncs the directory. Renaming within one filesystem gives us the atomic replacement property we need without a database dependency.

## Recovery behavior

Normal startup/load uses `known-good.json`.

If the current file is unreadable, malformed, has an unsupported schema, or contains invalid stored identity data, the store attempts to load `known-good.previous.json` instead. When that succeeds the caller is explicitly told that recovery came from the previous snapshot; the daemon must not silently pretend everything is normal.

If neither snapshot is usable, the store reports that there is no readable known-good configuration. Later setup-state code will turn that into a plain-language wizard/diagnostic state.

A recovery commit must not overwrite a valid previous snapshot with the corrupt current file.

## Why plain JSON first

The first configuration store deliberately uses only the Go standard library. It is tiny, easy to inspect/recover with normal Linux tools, cross-compiles cleanly for the Pi Zero/ARMv6 baseline, and does not add database startup or memory cost merely to store a small settings document.

This does **not** mean YWD-DMR will never use SQLite. Event history, Last Heard, account/session state, and other relational data may justify a database later. The known-good station configuration remains a separate transaction/recovery concern.

## Security rule

The store architecture is ready to contain protected fields later, but no BrandMeister password/token or administrator credential is being persisted in this slice.

When secrets are added:

- configuration read APIs must redact them;
- candidates may replace a secret but normal clients must not retrieve the stored value;
- filesystem ownership/mode remains part of the security boundary;
- commit APIs must be protected by daemon-side authorization.

## Current implementation status

Implemented on `dev`:

- schema/revision envelope;
- candidate identity validation and normalization before commit;
- atomic current-snapshot writes;
- one previous rollback snapshot;
- rejected candidates leave current known-good state unchanged;
- recovery load from a valid previous snapshot when current is corrupt;
- unsupported-schema rejection;
- unit tests using temporary directories, including file-mode and recovery checks.

The store is not yet exposed through a mutating HTTP API. That waits for setup-state and claim/auth boundaries so the current unauthenticated LAN-test dashboard cannot commit station configuration.
