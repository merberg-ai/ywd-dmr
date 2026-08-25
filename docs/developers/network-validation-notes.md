# BrandMeister Candidate Validation Notes

## 2026-08-24 Raspberry Pi 5 installed-appliance run

Development checkpoint: `1edb37f6f7719eb3c2dc888f2541bb40989eb439` on `dev`.

Installed release:

```text
dev-1edb37f-20260824201417
```

The protected BrandMeister **local candidate-validation** slice passed its real-machine gate. This run intentionally used the reserved `.invalid` namespace and did not perform a BrandMeister/Homebrew connection. The purpose was to prove authorization, normalization, password redaction, strict JSON/method handling, and the rule that local validation cannot change known-good state.

## Source gate

The Raspberry Pi 5 passed:

- `gofmt` check for the network/config/API files;
- focused network configuration tests;
- focused network API tests;
- `go test ./...`;
- `go vet ./...`;
- `git diff --check`;
- normal-user `./scripts/build.sh`.

The installed daemon remained healthy on the existing LAN-test listener `0.0.0.0:8990`.

## Runtime results

A fresh appliance was claimed with a temporary Admin account and station identity was committed as revision 1. Setup then correctly reported:

```text
stage: identity_complete
next_step: network
configuration revision: 1
```

The SHA-256 of `known-good.json` was captured before network validation.

### Authorization

An unauthenticated request to:

```text
POST /api/v1/setup/network/validate
```

returned HTTP `401` with `authentication required`.

An authenticated cross-origin request returned HTTP `403` with:

```json
{"error":"same-origin request required"}
```

No submitted password appeared in either response.

### Valid candidate normalization

The Admin request intentionally used mixed case, surrounding whitespace, a trailing hostname dot, and master port `0`:

```text
backend:        " BrandMeister "
master_address: " BM3103.INVALID. "
master_port:    0
password:       supplied test value
```

The installed API returned HTTP `200` and normalized the non-secret fields to:

```json
{
  "valid": true,
  "normalized": {
    "backend": "brandmeister",
    "master_address": "bm3103.invalid",
    "master_port": 62031,
    "password_set": true
  },
  "errors": []
}
```

The submitted password was not echoed anywhere in the response.

### Invalid candidate

A candidate containing an unsupported backend, URL/path instead of a master hostname, port `70000`, and an empty password returned HTTP `200` with `valid: false` and field errors for all four fields:

```text
backend
master_address
master_port
password
```

This is intentional: local field validation is a normal result rather than a transport failure.

### Strict request contract

- unknown JSON field -> HTTP `400`;
- GET against the POST-only route -> HTTP `405`;
- `Allow: POST` was present.

## Known-good preservation

The network-validation request did not change the SHA-256 of `known-good.json` and did not create `known-good.previous.json`.

Schema 1 remained exactly:

```text
schema
revision
identity
```

No network fields and no password were persisted. Setup remained `identity_complete / network / revision 1` throughout validation.

## Cleanup and health

The temporary Admin/security state and identity snapshot were removed after the run. The daemon restarted as a fresh unclaimed appliance, generated a new bootstrap claim code, and final diagnostics reported the service active and API healthy.

The overall result was:

```text
PASS: protected BrandMeister candidate validation
```

## Next gate

The next slice is a **real but non-persisting** BrandMeister Homebrew test. It must perform a bounded temporary `RPTL -> RPTK -> RPTC` handshake, classify login/auth/config/timeout/network failures, explicitly close the temporary session, transmit no `DMRD` traffic, return no password/challenge material, and leave known-good state untouched regardless of success or failure.
