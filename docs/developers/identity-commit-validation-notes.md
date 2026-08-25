# Protected Station-Identity Commit Validation Notes

## 2026-08-24 Raspberry Pi 5 partial installed-appliance run

Development checkpoint: `a682c1a3ced176538e2ea398969d1c98a2219d3a` on `dev`.

The first installed-appliance exercise of `POST /api/v1/setup/identity/commit` was **partially successful**. The implementation passed the source gate and the first durable protected commit. The second commit was not exercised because the validation command changed the HTTP `Host` header from `127.0.0.1:8990` to the LAN host while reusing a host-only session cookie issued to `127.0.0.1`. Curl therefore did not send that cookie on the second request, and the daemon correctly returned HTTP `401` with `{"error":"authentication required"}`.

This is a test-command cookie-host mismatch, not evidence that revision rotation or rollback recovery is broken.

## Confirmed in this run

- source gate passed;
- development release `dev-a682c1a-20260824184501` installed and was healthy;
- fresh setup state was unclaimed with missing configuration;
- one-time claim returned HTTP `201`;
- unauthenticated identity commit returned HTTP `401` and created no configuration file;
- authenticated cross-origin identity commit returned HTTP `403` and created no configuration file;
- an invalid Admin candidate returned HTTP `400` with callsign, DMR ID, and ESSID field errors and created no configuration file;
- the first valid Admin commit returned HTTP `200`, normalized `n0call` to `N0CALL`, and created revision `1`;
- `/var/lib/ywd-dmr/known-good.json` was mode `0600`, owned by `ywd-dmr:ywd-dmr`;
- no previous snapshot existed after the first commit, as expected;
- setup state advanced only after the durable commit to `identity_complete`, `next_step: network`, configuration `loaded`, revision `1`;
- daemon restart preserved revision `1` and cleared the old memory-only browser session;
- the durable administrator password still allowed a fresh login after restart;
- final cleanup returned the appliance to unclaimed/missing-config state with a fresh bootstrap claim code;
- final diagnostics reported the service active and the API healthy.

## Not proven by this run

Because the second valid commit was never authenticated, this run did **not** prove:

1. revision `1 -> 2` through the installed API;
2. creation of `known-good.previous.json` by the second commit;
3. current revision `2` containing the second identity while previous revision `1` retains the first identity;
4. invalid-candidate preservation when both current and previous snapshots exist;
5. normal restart loading revision `2`;
6. recovery from a deliberately corrupted current revision `2` using the API-created previous revision `1` snapshot;
7. the corresponding runtime `recovered` state and journal warning.

The later recovery attempt correctly produced `configuration_error` because no previous snapshot had ever been created.

## Validation-script correction

The corrected same-origin browser-style request must keep the same request host used when the cookie was issued. For the local test harness, use:

```text
URL/Host origin: http://127.0.0.1:8990
Origin:          http://127.0.0.1:8990
Sec-Fetch-Site: same-origin
```

Do not override `Host` to the LAN address while using the cookie jar created against `127.0.0.1`.

The original hash check also had a test-harness weakness: when `known-good.previous.json` was missing both the before/after `sha256sum` substitutions became empty strings, which could print a false PASS. The rerun must require both snapshot files to exist before comparing their hashes.

## Next validation

No reinstall is required unless code changes. The box was intentionally cleaned back to a fresh unclaimed/no-config state, so the corrected rerun can claim a test administrator, perform revision 1 and revision 2 commits using one consistent local origin, verify both snapshot files and hashes, restart, corrupt current revision 2, confirm recovery from previous revision 1, then clean up and run final health diagnostics.
