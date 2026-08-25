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

The first claimed account remains an Admin. Persistent multi-user/device credential management is not introduced by this slice; the role model and middleware are being established before additional principal types exist.

## Server-side middleware

Protected handlers use reusable authorization middleware that:

1. requires a live `ywd_dmr_session` cookie;
2. validates the opaque token through the in-memory session manager;
3. compares the authenticated role against the handler's minimum required role;
4. returns HTTP `401` when authentication is absent/invalid;
5. returns HTTP `403` when an authenticated principal lacks the required role;
6. places the authenticated principal in request context for the protected handler.

The middleware is intentionally implemented before configuration mutation endpoints are opened. The protected known-good configuration commit API will be the first normal setup endpoint to depend on Admin authorization.

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

This slice adds the authorization machinery and browser mutation protection, but it does not yet expose configuration commits, BrandMeister controls, radio controls, or PTT.

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

Real installed-appliance validation is still required before this slice is marked complete.
