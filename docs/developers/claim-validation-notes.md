# One-time Claim Validation Notes

## 2026-08-24 Raspberry Pi 5 installed-appliance validation

Claim implementation checkpoint: `4fcb3e239b1f5a90dc9cb3c5f418a9a3d5aea09b` on `dev`.

Documentation checkpoint before the successful rerun: `a4b9c81d640ea8b7baf6a21df52ac178f5507565`.

The one-time installation claim path is now **fully validated on the installed Raspberry Pi 5 appliance**. The earlier partial run was invalidated only by interactive Bash history expansion of `!` in the temporary test password; the daemon had correctly rejected that malformed JSON request. The corrected rerun used `set +H` and the shell-safe temporary password `YwdDmr-Test-Only-2026-Alpha`.

## Successful rerun results

Starting state was the expected fresh unclaimed appliance:

```json
{
  "claimed": false,
  "stage": "unclaimed",
  "next_step": "claim",
  "configuration": {
    "state": "missing",
    "identity_configured": false,
    "recovered": false
  }
}
```

The installed claim-code file was confirmed as:

```text
mode=600 owner=ywd-dmr:ywd-dmr size=28
```

The corrected validation proved all required behaviors:

1. an intentionally wrong claim code returned HTTP `403` with the generic `{"error":"claim failed"}` response;
2. the real one-time claim returned HTTP `201`;
3. the claim created administrator `testadmin` with role `admin`;
4. the successful claim took `0.516708s` on the Raspberry Pi 5, providing the first real-hardware PBKDF2 timing measurement;
5. the response set `ywd_dmr_session` only as a cookie with `HttpOnly` and `SameSite=Strict`; the opaque token was not returned in JSON;
6. `/var/lib/ywd-dmr/security.json` was created with mode `0600` and owner `ywd-dmr:ywd-dmr`;
7. the plaintext `/var/lib/ywd-dmr/claim-code` was removed after durable claim;
8. setup status changed to `claimed: true`, `stage: claimed`, and `next_step: identity`;
9. `GET /api/v1/auth/session` with the new cookie reported `authenticated: true`, username `testadmin`, and role `admin`;
10. reusing the already-consumed claim request returned HTTP `409` with `{"error":"installation is already claimed"}`;
11. after `ywd-dmrd.service` restart, durable claimed state remained intact;
12. the old browser cookie no longer authenticated after restart, confirming sessions are memory-only;
13. the daemon did not regenerate or expose a claim code while the appliance remained durably claimed;
14. intentional test cleanup returned the box to the unclaimed state and generated a fresh bootstrap claim code;
15. final `ywd-dmr diagnose` reported the service active and the API healthy.

## PBKDF2 timing note

The successful claim's PBKDF2-HMAC-SHA256 work plus surrounding claim processing completed in approximately **517 ms** on the Raspberry Pi 5 at the current `310000` iteration setting. This is intentionally nontrivial but acceptable for infrequent administrator authentication. Raspberry Pi Zero / ARMv6 remains the minimum performance baseline, so normal-login validation should also watch authentication latency there before the work factor is considered permanently tuned.

## Validation conclusion

Installed-appliance validation of the one-time claim slice is complete. The claim implementation demonstrated the intended security contract on real hardware:

```text
local high-entropy code -> one durable claim -> persisted password verifier
                              |
                              +-> opaque memory-only browser session
```

A durable claim survives daemon restart. A browser session does not. The one-time plaintext bootstrap code is burned after successful durable claim and is not regenerated on a claimed appliance. Corrupt durable security state continues to fail closed by design.

The next security slice is normal administrator authentication after restart:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
```

That slice must verify the persisted PBKDF2 record, use generic invalid-credential responses, throttle repeated login failures, issue a fresh opaque HttpOnly session on success, invalidate the current session on logout, and retain the existing memory-only session/restart behavior.
