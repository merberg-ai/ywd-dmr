# Authorization and Browser Mutation Protection

YWD-DMR uses daemon-owned authorization. The WebUI, future Android client, and CLI are clients of the same server-side rules and cannot grant themselves privileges.

## Role hierarchy

The first authorization model is intentionally small:

```text
Observer < Operator < Admin
```

Role meaning:

- **Observer** — may read protected operational/status information that is safe for an authenticated viewer. Observer cannot change configuration, control network/radio state, or transmit.
- **Operator** — includes Observer access and may perform approved day-to-day operational controls. Exact controls are added explicitly as their APIs are implemented. PTT will later require the Operator role or higher plus the separate renewable TX lease and timeout safety model.
- **Admin** — includes Operator and Observer access and may perform protected configuration/security/appliance administration operations that are explicitly exposed.

Unknown role strings fail closed. There is no implicit superuser role above Admin.

The first claimed account remains an Admin. Persistent multi-user/device credential management is not introduced by this slice; the role model and middleware are established before additional principal types exist.

## Server-side middleware

Protected handlers use reusable authorization middleware that:

1. requires a live `ywd_dmr_session` cookie;
2. validates the opaque token through the in-memory session manager;
3. compares the authenticated role against the handler's minimum required role;
4. returns HTTP `401` when authentication is absent/invalid;
5. returns HTTP `403` when an authenticated principal lacks the required role;
6. places the authenticated principal in request context for the protected handler.

The first normal setup mutation to depend on this middleware is the Admin-only station-identity commit endpoint:

```text
POST /api/v1/setup/identity/commit
```

## Browser mutation protection

Cookie-authenticated browser mutations need protection beyond authentication itself. YWD-DMR therefore applies a browser-origin check to state-changing HTTP methods:

```text
POST
PUT
PATCH
DELETE
```

For browser-originated requests:

- an `Origin` header, when present, must match the request scheme and Host exactly;
- `Sec-Fetch-Site`, when present, must be `same-origin` or `none`;
- `cross-site` and `same-site` browser mutation contexts are rejected with HTTP `403` and a generic same-origin-required response.

This deliberately treats sibling subdomains as different origins even though browsers may classify them as the same site.

Direct API clients such as curl do not normally send `Origin` or `Sec-Fetch-Site`, so they remain usable during development. Future Android/device credentials will not need to imitate browser headers.

The server does **not** trust `X-Forwarded-For`, `X-Forwarded-Proto`, or other proxy headers yet. As a result, production HTTPS reverse-proxy deployment is still not defined by this slice. Trusted-proxy configuration and HTTPS/WSS rules must be explicit before remote/public deployment is supported.

## Current boundary

The authorization/origin foundation is now real-machine validated. The first Admin-protected configuration commit endpoint is implemented on `dev`, but BrandMeister controls, radio controls, and PTT remain absent.

Station identity uses the existing known-good configuration transaction rather than a separate setup file. Identity has no external service to probe, so its path is:

```text
candidate -> normalize/validate -> durable commit
```

Network settings such as BrandMeister will later add a real connectivity-test step before commit.

The development LAN interface remains trusted-LAN-only. Do not router-forward it or expose it directly to the public internet.

## Validation status

Automated tests cover:

- the complete Observer/Operator/Admin hierarchy;
- unknown-role fail-closed behavior;
- missing/invalid-session authorization rejection;
- successful Admin authorization and principal propagation;
- direct-client mutation compatibility;
- same-origin browser mutation acceptance;
- cross-origin and same-site/different-origin mutation rejection;
- read-only GET behavior;
- end-to-end server middleware wiring before an existing mutation endpoint.

Installed Raspberry Pi 5 validation also passed. The runtime sequence was exactly:

```text
direct API mutation        200
same-origin browser        200
cross-origin Origin        403
cross-site fetch metadata  403
same-site/different-origin 403
cross-origin read-only GET 200
```

The unclaimed bootstrap code remained intact and final health passed. See [Authorization and Browser-Mutation Validation Notes](authorization-validation-notes.md) for the exact real-machine results.

The next validation target is the Admin-only station-identity commit path, including durable revision changes, invalid-candidate protection, restart persistence, and previous-snapshot rollback behavior.
