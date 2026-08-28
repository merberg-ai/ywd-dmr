# LAN Admin Test Console

The development WebUI includes an **Admin Test Console** so repeated YWD-DMR setup and BrandMeister testing can be performed from a browser instead of long shell/curl blocks.

This is a development convenience, not a security bypass. The console uses the same `/api/v1/` endpoints, authentication, role checks, browser same-origin protection, validation rules, and non-persistence guarantees as any other client.

## Safety boundary

The current YWD-DMR frontend is a **LAN test build**.

Do not:

- router-forward the YWD-DMR listener;
- expose it directly to the public internet;
- treat the current HTTP listener as production remote access.

Future public access requires the planned HTTPS/WSS and trusted-proxy deployment contract.

The test console does **not** provide shell access, sudo access, arbitrary file operations, or an API that bypasses normal authorization.

## What the console can do

The console currently provides browser controls for:

1. reading daemon health and setup status;
2. claiming a fresh appliance with the one-time local claim code;
3. Admin login and logout;
4. committing station identity;
5. locally validating a BrandMeister network candidate;
6. running the real short-lived BrandMeister Homebrew test;
7. displaying the structured API result;
8. refreshing setup/session state.

The live BrandMeister test still follows the daemon transaction rule:

```text
candidate
  -> local validation
  -> real login/auth/config test
  -> no persistence yet
```

The browser does not gain a network-commit operation merely because it can run the test.

## Claiming a fresh development install

The one-time claim code is intentionally available only on the local machine. Retrieve it through SSH or a local terminal:

```bash
sudo ywd-dmr claim-code
```

Paste that code into the **Claim fresh appliance** section, choose the Admin username/password, and press **Claim & Sign In**.

The claim code is sent to the normal protected bootstrap endpoint and is not stored by the WebUI.

## Password handling

Password inputs are normal browser password fields.

The console:

- does not write passwords to Local Storage or Session Storage;
- does not put passwords in URLs;
- does not print submitted passwords into the result panel;
- clears the Admin password after claim/login attempts;
- clears the BrandMeister Hotspot Security password after a live network test;
- applies an additional client-side redaction pass before displaying structured responses.

The server remains authoritative. Existing server rules still ensure network-test responses do not echo the Hotspot Security password, challenge salt, or authentication digest.

## Station identity

The identity form calls:

```text
POST /api/v1/setup/identity/commit
```

It requires an authenticated Admin session. Callsign, base DMR ID, and ESSID are submitted through the same known-good configuration path already validated on the Raspberry Pi.

## BrandMeister candidate

The current development form includes:

```text
master hostname/IP
UDP port
registration frequency in Hz
Hotspot Security password
```

The registration frequency is Homebrew registration metadata. Entering a value does **not** enable an RF transmitter or cause YWD-DMR to key an attached MMDVM modem.

The field expects **Hz**, not MHz. Examples:

```text
147.420 MHz -> 147420000 Hz
446.525 MHz -> 446525000 Hz
```

A value such as `14742000` means 14.742 MHz and is rejected by the current BrandMeister candidate validator because it is outside YWD-DMR's supported Homebrew registration range.

**Validate Candidate** calls:

```text
POST /api/v1/setup/network/validate
```

This performs local normalization/validation only. A missing Hotspot Security password is also reported as a validation error; the password is not remembered from a previous operation.

**Run Live BM Test** calls:

```text
POST /api/v1/setup/network/test
```

The browser asks for confirmation before the real test. The daemon then performs the bounded temporary Homebrew setup handshake. The current tester sends no `DMRD` voice/data and does not persist the network candidate.

## Result panel

The result panel shows:

- the operation name;
- HTTP status;
- structured response from the daemon.

Typical live reasons remain:

```text
ok
login
auth
config
timeout
network
unavailable
```

This makes the browser console useful for the same protocol-debugging work previously performed with long curl scripts.

## What is intentionally missing

There is currently no browser button to wipe security state, delete known-good configuration, restart services, run shell commands, or perform an arbitrary appliance reset.

Those operations are more powerful than the normal Control API and should not be smuggled into the UI merely for convenience. If a development-only reset workflow is added later, it must have an explicit server-side contract, strong Admin checks, clear LAN/development gating, and documentation before it is exposed.

## Relationship to the future production WebUI

This console is useful scaffolding, but it is not the final first-run wizard. As Alpha 1 matures, proven pieces can be promoted into the normal guided UI while developer-only diagnostics remain clearly separated.

The important rule is that both use the same daemon-owned API and safety model rather than maintaining a special browser-only implementation.
