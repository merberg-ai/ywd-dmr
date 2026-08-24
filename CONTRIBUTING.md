# Contributing to YWD-DMR

YWD-DMR is being built as a small, dependable radio appliance rather than a collection of tightly coupled experiments.

## Ground rules

- Keep Pi Zero / ARMv6 performance in mind.
- Do not move authoritative call/PTT/security state into the WebUI.
- Use versioned APIs between components.
- Do not hardwire BrandMeister assumptions into generic core interfaces when a network-backend boundary is appropriate.
- Vocoders stay out-of-process.
- User/local vocoder plugins must remain independently installable and updatable.
- Never commit passwords, BrandMeister secrets/API keys, device credentials, or support bundles containing secrets.
- Do not copy external DMR/vocoder implementation code without first confirming license compatibility and documenting provenance.

## Documentation is part of the change

A code change that changes behavior without updating the corresponding `/docs/` page is incomplete.

## Before opening a pull request

```bash
go test ./...
go vet ./...
./scripts/build.sh
```

The CI build also cross-compiles the daemon for Linux ARMv6, our Raspberry Pi Zero baseline.
