# Known-good Configuration Store

YWD-DMR does not treat every submitted settings form as active configuration. The daemon keeps one **known-good** configuration and one previous rollback snapshot so a bad candidate cannot casually destroy a working station setup.

## Where it lives

The daemon-owned state directory contains:

```text
/var/lib/ywd-dmr/known-good.json
/var/lib/ywd-dmr/known-good.previous.json
/var/lib/ywd-dmr/secrets/
```

The configuration snapshots are mode `0600`. The secret directory is mode `0700`, and revision-bound secret files inside it are mode `0600`. The state tree is owned by the non-root `ywd-dmr` service account.

These files are different from `/etc/ywd-dmr/ywd-dmr.env`, which contains service/bootstrap settings such as the frontend listener.

## Schema 1 is frozen as identity-only

Schema 1 remains exactly the already validated identity format:

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

A schema-1 document containing a `network` object is rejected. Network fields are **not** silently added to the old schema.

## Schema 2 adds tested non-secret network configuration

After the real BrandMeister login/auth/config handshake was accepted on the Pi 5, YWD-DMR introduced schema 2:

```json
{
  "schema": 2,
  "revision": 2,
  "identity": {
    "callsign": "N0CALL",
    "dmr_id": 1234567,
    "essid": 3
  },
  "network": {
    "backend": "brandmeister",
    "master_address": "3103.master.brandmeister.network",
    "master_port": 62031,
    "registration_frequency_hz": 446525000,
    "password_set": true
  }
}
```

The Hotspot Security password is intentionally **not present** in `known-good.json` or `known-good.previous.json`.

Schema 2 is created only by a successful tested network commit. It is not produced by `/network/validate` or the non-persisting `/network/test` diagnostic.

## Revision-bound network secrets

A schema-2 revision has a matching daemon-only secret file:

```text
/var/lib/ywd-dmr/secrets/network-2.json
```

The exact filename revision must match the active known-good revision. The secret file contains a small internal envelope and the Hotspot Security password. It is never served by an API, never copied into the WebUI result panel, and never written into normal configuration JSON.

Why bind the secret to the revision?

Because rollback must restore a **matching pair**:

```text
configuration revision 2
      +
network secret revision 2
```

If current revision 3 becomes corrupt and YWD-DMR recovers previous revision 2, it also loads the revision-2 secret. It cannot accidentally combine previous network settings with a newer password.

The store writes the new secret before publishing a schema-2 current snapshot. If a later configuration write fails, the new secret is only an orphan; no active snapshot points at it. Cleanup removes stale revision-bound network secret files after successful commits while preserving the current and rollback revisions.

## Commit rules

Identity-only transaction:

```text
candidate
  -> normalize/validate
  -> durable commit
```

Network transaction:

```text
candidate
  -> local validation
  -> REAL BrandMeister login/auth/config test
  -> durable commit of that exact normalized candidate
```

A candidate that merely has valid-looking fields is not known-good.

## One-request test-and-commit

The protected durable network operation is:

```text
POST /api/v1/setup/network/test-and-commit
```

YWD-DMR intentionally performs the live test and commit inside one request. The exact normalized `NetworkCandidate` remains in daemon memory from validation through BrandMeister testing and durable storage.

This avoids a browser-held reusable "test passed" token and avoids this dangerous sequence:

```text
test candidate A
change form to candidate B
commit B using proof from A
```

Instead:

```text
candidate A
  -> validate A
  -> test A
  -> if and only if A is accepted, commit A
```

If BrandMeister returns `auth`, `config`, `timeout`, `network`, or another normal test failure, the response reports `committed: false` and known-good state is untouched.

## Rotation

On a first identity commit:

```text
current revision 1 / schema 1
no previous snapshot
```

On the first successful network test-and-commit:

```text
current  -> revision 2 / schema 2
previous -> revision 1 / schema 1
secret   -> network-2.json
```

A later successful network update might produce:

```text
current  -> revision 3 / schema 2
previous -> revision 2 / schema 2
secrets  -> network-3.json + network-2.json
```

Only after the new secret and new current configuration are safely written does the new runtime setup state advance.

## Identity changes after network setup

An identity commit after schema 2 is active must not silently erase a known-good network.

The store therefore preserves the existing tested network metadata and copies/rebinds its secret to the new revision while applying the new validated identity. That new revision remains schema 2.

Changing identity may require the long-lived network runtime to reconnect later, but it does not reduce durable configuration back to identity-only by accident.

## Startup and recovery behavior

On daemon startup, YWD-DMR loads `known-good.json` and validates:

1. supported schema;
2. non-zero revision;
3. stored identity;
4. schema/network relationship;
5. non-secret network fields for schema 2;
6. matching revision-bound network secret for schema 2;
7. the combined stored network candidate including the real secret.

If current cannot be used, the store attempts `known-good.previous.json` and its matching secret. Successful fallback is reported as recovered.

If both snapshot files are absent, configuration state is `missing`.

If a snapshot exists but its JSON, schema, identity, network data, or required secret is unusable and no valid rollback exists, configuration state is `error`. A missing schema-2 secret is **not** misreported as a fresh empty appliance.

## Setup status

`GET /api/v1/setup/status` remains a coarse non-secret status endpoint.

Identity-only known-good state reports:

```text
stage: identity_complete
next_step: network
network_configured: false
```

After durable schema-2 network commit:

```text
stage: network_complete
next_step: audio
network_configured: true
```

The status endpoint does not return callsign, DMR ID, master, registration frequency, or password.

## Security boundary

The Hotspot Security credential is sensitive but must be recoverable by the daemon across restart for the future long-lived BrandMeister backend.

For Alpha1 the protection model is intentionally simple and honest:

- daemon runs as dedicated non-root `ywd-dmr`;
- secret directory mode `0700`;
- secret files mode `0600`;
- secret never returned by an API;
- secret never included in validation/test/commit result bodies;
- secret never placed in URL/query strings;
- WebUI clears the password field after live test or test-and-commit;
- normal known-good JSON contains only `password_set: true`;
- systemd restricts daemon write access to its state/log paths.

Encrypting a password with a key stored beside it would not meaningfully improve the appliance boundary. A future platform-backed secret facility may be added where hardware/OS support provides a real separate trust anchor.

Diagnostics/support bundles must continue to exclude revision-bound secret files unless an explicit future redacted export contract says otherwise.

## Existing real-machine proof

Before schema 2, the Pi 5 already proved:

```text
revision 1 identity commit
revision 2 identity commit
invalid candidate preservation
restart persistence
current corruption -> previous recovery
0600 current/previous ownership
```

The Pi 5 then proved the full non-persisting BrandMeister handshake, including acceptance of a YWD-DMR-owned numeric/date-style software identifier with the `MMDVM_DMO` compatibility profile.

The next installed-machine gate is schema-2 test-and-commit, restart, rotation, and recovery validation.

## Current implementation status

- [x] Schema 1 frozen identity-only.
- [x] Schema 2 non-secret network shape implemented.
- [x] Revision-bound restricted network secret store implemented.
- [x] Identity commit preserves an existing tested network.
- [x] Atomic current/previous configuration rotation retained.
- [x] Current/previous recovery validates matching network secret revision.
- [x] Protected one-request network test-and-commit endpoint implemented on `dev`.
- [x] WebUI control for explicit test-and-commit implemented on `dev`.
- [x] Unit/API tests added for schema 2, secret redaction, rotation/recovery, failed-test preservation, and same-origin protection.
- [ ] Installed Pi 5 proof of schema-2 network commit.
- [ ] Installed Pi 5 restart/load proof with revision-bound secret.
- [ ] Installed Pi 5 schema-2 rotation/recovery proof.
- [ ] Long-lived BrandMeister connection/reconnect state machine.
