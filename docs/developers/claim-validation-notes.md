# One-time Claim Validation Notes

## 2026-08-24 Raspberry Pi 5 test

Development checkpoint: `4fcb3e239b1f5a90dc9cb3c5f418a9a3d5aea09b` on `dev`.

The claim/security slice built and installed successfully on the Raspberry Pi 5. The full Go suite, `go vet`, maintenance CLI regression test, and normal build passed. The installed appliance was healthy on `0.0.0.0:8990`.

Confirmed on the installed appliance:

- initial setup status reported `claimed: false`, `stage: unclaimed`, and `next_step: claim`;
- `/var/lib/ywd-dmr/claim-code` existed with mode `0600` and owner `ywd-dmr:ywd-dmr`;
- `sudo ywd-dmr claim-code` retrieved the local one-time code;
- an intentionally wrong claim code returned HTTP `403` with `{"error":"claim failed"}`;
- after the test cleanup, the daemon returned to the unclaimed state, regenerated a fresh bootstrap code, and final health diagnostics passed.

The successful-claim path was **not exercised** in this run. The shell expanded the `!` character in the temporary test password while constructing the double-quoted JSON request (`-bash: !\\: event not found`). `curl` therefore received malformed JSON and the daemon correctly returned HTTP `400` with `{"error":"invalid JSON request"}`. Because the real claim never occurred, there was no `security.json`, the claim code correctly remained present, setup remained unclaimed, no authenticated session existed, and a claim code was still available after restart.

This is a test-command quoting problem, not evidence of a claim implementation failure. Do not mark the installed-appliance claim checklist complete yet.

## Next validation

Repeat the claim test using a shell-safe temporary password such as `YwdDmr-Test-Only-2026-Alpha`, or disable Bash history expansion before building the request. The rerun still needs to prove:

1. wrong code returns HTTP `403`;
2. valid claim returns HTTP `201`;
3. `/var/lib/ywd-dmr/security.json` is created as mode `0600` and owned by `ywd-dmr:ywd-dmr`;
4. plaintext `claim-code` is removed after successful claim;
5. setup status becomes claimed with `next_step: identity`;
6. the returned HttpOnly session cookie authenticates `GET /api/v1/auth/session`;
7. reusing the one-time claim returns HTTP `409`;
8. claimed state survives daemon restart;
9. the memory-only browser session does not survive daemon restart;
10. no new claim code is generated on a claimed restart;
11. after intentional test cleanup, the appliance returns to a fresh unclaimed state with a new bootstrap code and healthy daemon.
