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

Schema 1 currently contains only normalized radio identity:

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

BrandMeister/network configuration is **not** being silently added to schema 1. The network commit will make an explicit schema/migration decision after the real connectivity tester exists.

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

The file implementation writes a temporary file in the same directory, sets restrictive permissions, writes and syncs the contents, then renames it over the destination and syncs the directory.

On a first commit there is no previous snapshot yet. On a later successful commit, the prior current snapshot becomes `known-good.previous.json` before the new current snapshot is installed.

## Protected station-identity commit

Implemented and real-machine validated:

```text
POST /api/v1/setup/identity/commit
```

This endpoint requires Admin authentication and browser same-origin protection.

Identity has no external dependency to probe, so its transaction is:

```text
candidate -> normalize/validate -> durable commit
```

A successful first commit creates revision 1. Later successful commits increment the revision and rotate the old current snapshot to the previous file. Runtime setup state advances only after durable commit succeeds.

Invalid candidates, missing authentication, cross-origin browser requests, malformed JSON, or durable-write failures must not replace known-good state.

## Real Raspberry Pi rotation/recovery proof

The installed Raspberry Pi 5 API exercise proved the actual durable sequence:

```text
revision 1 commit
  -> current revision 1
  -> no previous file

revision 2 commit
  -> current revision 2
  -> previous revision 1

invalid candidate
  -> HTTP 400
  -> current and previous SHA-256 hashes unchanged

restart
  -> current revision 2 loaded normally

corrupt current revision 2 + restart
  -> previous revision 1 recovered
  -> setup status state=recovered
  -> explicit journal warning
```

Both real snapshot files were mode `0600` and owned by `ywd-dmr:ywd-dmr`.

This is important: startup recovery has now been proven using the **same previous snapshot created by a real authenticated API update**, not merely a hand-written fixture.

See [Protected Station-Identity Commit Validation Notes](identity-commit-validation-notes.md).

## Network configuration rule

Network settings such as BrandMeister require a fuller transaction because syntactically valid credentials may still fail in the real world:

```text
candidate
  -> local validation
  -> real connectivity/authentication test
  -> commit
```

The protected network-validation endpoint now exists, but it does not write this store. The future network test must prove the master/login/auth/config path before any network candidate can become known-good.

A failed network test must not increment the revision or alter current/previous snapshots.

See [DMR Network Backend and BrandMeister Setup Contract](network-backend-contract.md).

## Startup and recovery behavior

On daemon startup, YWD-DMR loads `known-good.json`.

If current is unreadable, malformed, has an unsupported schema, or contains invalid stored identity data, the store attempts `known-good.previous.json`. Successful fallback is explicitly reported as recovered; it is never silently treated as normal operation.

If neither snapshot is usable:

- both absent -> configuration state `missing`;
- one or both exist but neither is usable -> configuration state `error`.

`GET /api/v1/setup/status` reports only coarse health, whether identity is configured, and revision. It does not return the stored identity.

A recovery commit must not overwrite a valid previous snapshot with a corrupt current file.

## Why plain JSON first

The first store uses only the Go standard library. It is small, easy to inspect/recover with normal Linux tools, cross-compiles cleanly for the Pi Zero/ARMv6 baseline, and avoids a database dependency merely to store a tiny station configuration.

This does **not** mean YWD-DMR will never use SQLite. Event history, Last Heard, account/session state, and other relational data may justify it later. Known-good station configuration remains a separate transaction/recovery concern.

## Security rule

No BrandMeister password/token or administrator credential is persisted in this configuration document today.

When network secrets are added:

- configuration read/status APIs must redact them;
- candidates may replace a secret but normal clients must not retrieve it;
- filesystem ownership/mode remains part of the security boundary;
- commit APIs remain protected by daemon-side authorization;
- test results and logs must never expose the submitted password.

## Current implementation status

- [x] Schema/revision envelope.
- [x] Identity validation/normalization before commit.
- [x] Atomic current-snapshot writes.
- [x] Previous rollback snapshot.
- [x] Invalid-candidate preservation.
- [x] Recovery load from valid previous when current is corrupt.
- [x] Unsupported-schema rejection.
- [x] Daemon startup missing/loaded/recovered/error state.
- [x] Read-only setup status without stored identity disclosure.
- [x] Admin-protected identity commit through this same store.
- [x] Automated authorization/mutation/revision tests.
- [x] Installed Pi 5 revision `1 -> 2` and previous-snapshot rotation.
- [x] Installed Pi 5 invalid-candidate hash preservation.
- [x] Installed Pi 5 restart persistence.
- [x] Installed Pi 5 recovery from the API-created previous snapshot.
- [ ] Explicit schema/migration design for network configuration.
- [ ] BrandMeister network candidate real test before commit.
- [ ] Protected tested network commit through this same store.
