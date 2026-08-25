# Administrator Authentication Validation Notes

## 2026-08-24 Raspberry Pi 5 installed-appliance validation

Installed development release:

```text
dev-df681f9-20260824181622
```

The normal administrator login/logout/throttling slice is **fully validated on the installed Raspberry Pi 5 appliance**.

The source gate passed before installation:

```text
go test ./...
go vet ./...
git diff --check
./scripts/build.sh
```

The development installer then built, installed, started, and health-checked the `df681f9` release on the existing `0.0.0.0:8990` LAN-test listener.

## Real-machine results

A fresh temporary administrator was created through the already validated one-time claim path. The claim returned HTTP `201` in `0.531408s`, and its session authenticated correctly.

After daemon restart:

- durable claimed/admin state survived;
- the old memory-only claim session no longer authenticated;
- setup remained `claimed: true`, `stage: claimed`, `next_step: identity`.

Normal login then proved the intended persisted-password behavior:

```text
wrong username -> HTTP 401, 0.520663s
wrong password -> HTTP 401, 0.520534s
valid login    -> HTTP 200, 0.519940s
```

Both invalid-credential cases returned the identical generic body:

```json
{"error":"authentication failed"}
```

The close wrong-username/wrong-password timings confirm that the wrong-username path still performs password-KDF work rather than exposing a cheap username-existence timing oracle.

A successful login returned only non-secret session metadata and set a fresh `ywd_dmr_session` cookie with `HttpOnly` and `SameSite=Strict`. `GET /api/v1/auth/session` authenticated that cookie as administrator `testadmin`.

Logout returned HTTP `204`, expired the cookie, invalidated the server-side session, and the old token immediately reported `authenticated: false`.

## Throttle validation

The configured first login throttle is:

```text
5 failed logins from one direct client IP inside 5 minutes
-> block further login attempts from that IP for 60 seconds
```

The installed Pi produced:

```text
failure 1 -> 401, 0.519836s
failure 2 -> 401, 0.545728s
failure 3 -> 401, 0.520845s
failure 4 -> 401, 0.519354s
failure 5 -> 401, 0.519827s
failure 6 -> 429, 0.000410s
Retry-After: 59
```

The sixth request was rejected before PBKDF2 work, as intended while the client was blocked.

Restarting `ywd-dmrd.service` cleared the memory-only throttle. A correct password immediately logged in again with HTTP `200` in `0.517669s`, and the fresh post-restart session authenticated successfully.

## Cleanup and conclusion

Intentional test cleanup removed the temporary security state, returned setup to the unclaimed stage, regenerated a fresh bootstrap claim code, and left the service healthy. Final `ywd-dmr diagnose` reported the API responding on port 8990.

Installed-appliance validation of normal administrator authentication is complete. The next security slice is server-side role authorization plus browser mutation protection:

```text
Observer < Operator < Admin
```

Authenticated cookie-backed browser mutations must be same-origin/CSRF protected before configuration commits, network controls, or radio controls are exposed.

Raspberry Pi Zero / ARMv6 remains the minimum performance baseline. The current roughly 0.52-second PBKDF2 cost measured on Pi 5 is accepted for Alpha1 development, but the same work factor still needs performance validation on the Pi Zero before it is considered permanently tuned.
