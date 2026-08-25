# Authorization and Browser-Mutation Validation Notes

## 2026-08-24 Raspberry Pi 5 installed-appliance validation

Validated installed development release:

```text
dev-c485ca7-20260824182607
```

Source checkpoint:

```text
c485ca715d1d8a83db02092c836ba4b3f8cfc9f1
```

The role hierarchy, reusable authorization middleware tests, and browser mutation-origin protection all passed the source gate and the browser-origin behavior was exercised against the real installed daemon on the Raspberry Pi 5.

## Source gate

The following all passed:

```text
bash -n scripts/install.sh
go test ./...
go test ./internal/security -run TestRoleAllowsHierarchy -v
go test ./internal/httpapi -run 'TestRequireRole|TestBrowserMutationProtection|TestNewServerRejectsCrossOriginMutation' -v
go vet ./...
git diff --check
./scripts/build.sh
```

Role tests confirmed the intended hierarchy:

```text
Observer < Operator < Admin
```

including fail-closed behavior for unknown actual or minimum roles.

Authorization middleware tests confirmed that a missing or invalid session is rejected and an Admin session can enter an Admin-protected handler with its authenticated principal attached to request context.

## Installed runtime results

The development installer preserved the configured LAN listener on `0.0.0.0:8990`, installed the new release, and emitted the corrected security warning:

```text
Authentication exists, but production HTTPS/authorization hardening is incomplete.
Do not forward port 8990 from your router and do not expose this service to the public internet.
```

The installed daemon remained healthy before and after the validation.

The runtime browser-mutation sequence produced exactly:

```text
direct API client                    200
same-origin browser-style mutation  200
cross-origin Origin header          403
cross-site Sec-Fetch-Site           403
same-site but different Origin      403
read-only cross-origin GET          200
```

The rejected mutations returned:

```json
{"error":"same-origin request required"}
```

This proves the browser protection does not accidentally break ordinary direct API clients, allows same-origin WebUI requests, blocks cross-origin browser mutation attempts, and does not apply mutation restrictions to read-only GET requests.

The test appliance remained unclaimed throughout this runtime exercise. `/var/lib/ywd-dmr/claim-code` remained present, proving these requests did not disturb bootstrap security state.

Final `ywd-dmr diagnose` reported the service active and API healthy.

## Validation conclusion

The authorization/origin foundation is validated on the installed Raspberry Pi 5 appliance.

The next configuration slice can therefore use the reusable middleware rather than implementing endpoint-specific security. The first protected configuration mutation is station identity:

```text
POST /api/v1/setup/identity/commit
```

It is Admin-only, browser same-origin protected, and commits through the existing known-good configuration store. Identity has no external network dependency, so its transaction is candidate -> normalize/validate -> durable commit. Network configuration such as BrandMeister will later add a real test step before commit.
