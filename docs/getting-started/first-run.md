# First-run Setup

The production first-run wizard is planned but not complete yet. Phase 2 implementation has now started with the daemon-side station identity model and validation API.

The finished wizard will use large, plain-language steps:

1. Claim the new YWD-DMR installation using a one-time setup code.
2. Create the administrator account. There will be no default `admin/admin` password.
3. Enter callsign, DMR ID, and ESSID.
4. Select the DMR network and master server.
5. Test network credentials before saving them as active.
6. Choose and test microphone and speaker devices.
7. Detect or configure a vocoder backend, or continue safely in no-vocoder mode.
8. Run a complete station health check.
9. Finish setup and open the dashboard.

If a test fails, the wizard should explain what failed in normal language and let the user retry or continue when the failed component is optional.

## What works now on `dev`

The daemon can already normalize and validate the three station-identity fields that the future wizard will collect:

- **Callsign** — trimmed and converted to uppercase.
- **DMR ID** — the base numeric DMR ID, kept separate from a hotspot suffix.
- **ESSID** — a value from 0 through 99 for a network/device suffix when needed.

The development API endpoint is:

```text
POST /api/v1/setup/identity/validate
```

This is only a validation step. It does **not** save the values yet, does not create an administrator, and does not connect to BrandMeister.

That separation is intentional: YWD-DMR will add its durable known-good configuration store and one-time claim/authentication boundary before any setup endpoint is allowed to commit settings or secrets.

## Current safety note

The current LAN test dashboard is still unauthenticated. Keep it on a trusted local network only. Do not forward the YWD-DMR frontend port through a router or expose this development build directly to the public internet.

For the implementation order and security boundary, see [Setup and Security Phase](../developers/setup-security-phase.md).
