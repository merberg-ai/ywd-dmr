# Protected Station-Identity Commit Validation Notes

## 2026-08-24 Raspberry Pi 5 installed-appliance validation

Implementation checkpoint: `a682c1a3ced176538e2ea398969d1c98a2219d3a` on `dev`.

The Admin-protected station-identity commit path is now **fully validated on the installed Raspberry Pi 5 appliance**.

The first run exposed a validation-script cookie-host mismatch rather than a daemon defect: a session cookie issued to `127.0.0.1` was later reused with a request whose Host had been changed to the LAN address, so curl correctly withheld the host-only cookie and the daemon returned HTTP `401`. The corrected rerun kept URL/Host and `Origin` on `http://127.0.0.1:8990` and completed the full rotation/recovery exercise.

## Complete results

The combined first run and corrected rerun proved:

1. source tests/build gate passed and release `dev-a682c1a-20260824184501` installed healthy;
2. a fresh appliance began `unclaimed` with configuration state `missing`;
3. the one-time test Admin claim returned HTTP `201`;
4. an unauthenticated identity commit returned HTTP `401` and created no configuration;
5. an authenticated cross-origin browser mutation returned HTTP `403` and created no configuration;
6. an invalid Admin candidate returned HTTP `400` with callsign, DMR-ID, and ESSID field errors and created no configuration;
7. the first valid Admin commit returned HTTP `200`, normalized the callsign to `N0CALL`, and created revision `1`;
8. `known-good.json` was mode `0600` and owned by `ywd-dmr:ywd-dmr`;
9. no previous snapshot existed after the first commit, as expected;
10. runtime setup state advanced only after durable commit to `identity_complete`, `next_step: network`, revision `1`;
11. a second authenticated same-origin commit returned HTTP `200`, normalized `k1abc` to `K1ABC`, and created revision `2`;
12. `known-good.previous.json` was created as mode `0600`, owner `ywd-dmr:ywd-dmr`;
13. current revision `2` contained `K1ABC / 7654321 / ESSID 2` while previous revision `1` retained `N0CALL / 1234567 / ESSID 1`;
14. an invalid candidate after revision `2` returned HTTP `400` and SHA-256 checks confirmed both current and previous snapshots remained byte-for-byte unchanged;
15. daemon restart loaded revision `2` normally and cleared the old memory-only browser session;
16. durable administrator credentials still allowed a fresh HTTP `200` login after restart;
17. after deliberately corrupting current revision `2`, daemon startup recovered the API-created previous revision `1` snapshot;
18. setup status reported `state: recovered`, `revision: 1`, `identity_configured: true`, `recovered: true`;
19. the service journal emitted `WARNING: known-good configuration recovered from previous snapshot (revision 1)`;
20. the previous snapshot itself remained valid revision `1` after recovery;
21. cleanup removed security/configuration test state, regenerated a fresh bootstrap claim code, and returned setup status to `unclaimed` / `missing`;
22. final `ywd-dmr diagnose` reported the service active and the API healthy.

## Snapshot rotation proven on real storage

The actual installed API produced this durable sequence:

```text
first Admin commit
  -> known-good.json revision 1

second Admin commit
  -> known-good.json          revision 2
  -> known-good.previous.json revision 1

corrupt current revision 2 + restart
  -> runtime recovers previous revision 1
  -> recovered state and journal warning are explicit
```

This proves the rollback snapshot used by startup recovery is not merely a unit-test fixture. It is the same previous snapshot created by a real authorized configuration update on the appliance.

## Validation-harness lesson

Cookie-authenticated same-origin tests must keep the cookie host, request Host, and `Origin` consistent. For the corrected local test:

```text
URL/Host: http://127.0.0.1:8990
Origin:   http://127.0.0.1:8990
```

The corrected harness also requires both snapshot files to exist before comparing hashes, preventing the earlier false-PASS possibility where two missing-file substitutions both became empty strings.

## Conclusion

The complete setup/configuration security chain is now proven on real hardware:

```text
one-time claim
  -> durable Admin credential
  -> memory-only authenticated session
  -> server-side Admin authorization
  -> browser same-origin mutation protection
  -> validated station-identity candidate
  -> atomic known-good commit
  -> previous-snapshot rotation
  -> restart persistence / explicit recovery
```

The next development phase can safely begin the BrandMeister/network configuration contract. Network settings must add a genuine connectivity-test step before durable commit; local field validation alone must never be treated as proof that a master address or hotspot password works.
