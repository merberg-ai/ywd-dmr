# Troubleshooting — Start Here

YWD-DMR is intended to explain failures in the WebUI before asking a user to run Linux commands.

The finished dashboard will provide **Diagnostics → Test My Setup** and **Download Support Bundle**.

A health test should check, in plain language:

- YWD-DMR core service
- network connectivity
- DMR network login
- vocoder status
- microphone and speaker
- API/database health
- disk space and time synchronization
- configuration validity

Support bundles must remove passwords, API keys, session credentials, and other secrets automatically.

## Foundation build

If the development server does not start, run:

```bash
go test ./...
go run ./cmd/ywd-dmrd
```

The normal development address is `http://127.0.0.1:8090/`.

If you changed `YWD_DMR_LISTEN`, use that address instead.
