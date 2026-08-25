# Known-good Configuration Store

YWD-DMR does not treat every submitted settings form as active configuration. The daemon keeps one **known-good** configuration and one previous rollback snapshot so a bad candidate cannot casually destroy a working station setup.

## Where it lives

The development store uses files under the daemon-owned state directory:

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

## Commit rules

A candidate follows this direction:

```text
client candidate
      |
      v
server-side validation / normalization
      |
      +-- invalid -> reject, current known-good remains untouched
      |
      v
save current as previous rollback snapshot
      |
      v
atomic replace of current known-good snapshot
```

The file implementation writes a temporary file in the same directory, sets restrictive permissions, writes and syncs the contents, then renames it over the destination and syncs the directory. Renaming within one filesystem gives us the atomic replacement property we need without a database dependency.

On a first commit there is no previous snapshot yet. On a later successful commit, the prior current snapshot becomes `known-good.previous.json` before the new current snapshot is installed.

## Protected HTTP commit path

The first real HTTP consumer of the store is now implemented on `dev`:

```text
POST /api/v1/setup/identity/commit
```

This endpoint requires an Admin session and is covered by the global browser same-origin mutation rule. It accepts only the identity candidate fields; clients cannot supply `schema` or `revision`.

Identity has no external service to probe, so this transaction is:

```text
candidate -> normalize/validate -> durable commit
```

A successful first commit creates revision 1. A later successful commit increments the revision and rotates the old current snapshot to the previous file. Runtime setup state advances only after the durable commit succeeds.

Invalid candidates, missing authentication, insufficient role, cross-origin browser requests, malformed JSON, or durable-write failures must not replace known-good state.

Network settings such as BrandMeister will later use the fuller pattern:

```text
candidate -> validate -> connectivity test -> commit
```

## Startup and recovery behavior

On daemon startup, YWD-DMR loads `known-good.json` from the state directory.

If the current file is unreadable, malformed, has an unsupported schema, or contains invalid stored identity data, the store attempts to load `known-good.previous.json` instead. When that succeeds the daemon records a recovered configuration state and logs a warning; recovery is not silently treated as normal operation.

If neither snapshot is usable, the daemon distinguishes two cases:

- both snapshots absent -> configuration state is `missing`;
- one or both snapshots exist but no usable configuration can be loaded -> configuration state is `error`.

The read-only `GET /api/v1/setup/status` endpoint reports only coarse health/state information, whether identity is configured, and the revision when available. It does not return the stored identity itself.

A recovery commit must not overwrite a valid previous snapshot with a corrupt current file.

## Why plain JSON first

The first configuration store deliberately uses only the Go standard library. It is tiny, easy to inspect/recover with normal Linux tools, cross-compiles cleanly for the Pi Zero/ARMv6 baseline, and does not add database startup or memory cost merely to store a small settings document.

This does **not** mean YWD-DMR will never use SQLite. Event history, Last Heard, account/session state, and other relational data may justify a database later. The known-good station configuration remains a separate transaction/recovery concern.

## Security rule

The store architecture is ready to contain protected fields later, but no BrandMeister password/token or administrator credential is persisted in this configuration document today.

When secrets are added:

- configuration read APIs must redact them;
- candidates may replace a secret but normal clients must not retrieve the stored value;
- filesystem ownership/mode remains part of the security boundary;
- commit APIs must remain protected by daemon-side authorization.

## Current implementation status

Implemented on `dev`:

- schema/revision envelope;
- candidate identity validation and normalization before commit;
- atomic current-snapshot writes;
- one previous rollback snapshot;
- rejected candidates leave current known-good state unchanged;
- recovery load from a valid previous snapshot when current is corrupt;
- unsupported-schema rejection;
- unit tests using temporary directories, including file-mode and recovery checks;
- daemon startup loading from the configured state directory;
- explicit missing/loaded/recovered/error setup state;
- read-only setup-status reporting without returning stored identity fields;
- Admin-protected `POST /api/v1/setup/identity/commit` wired to this same store;
- API tests for unauthorized rejection, normalized persistence, invalid-candidate preservation, revision increment, runtime setup-state advancement, and cross-origin rejection before mutation.

The core store and startup/recovery behavior were already exercised on the Raspberry Pi 5. Installed validation of the new protected HTTP commit path is the active gate.
