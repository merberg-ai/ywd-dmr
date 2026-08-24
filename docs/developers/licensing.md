# Licensing Status

YWD-DMR does **not yet have a final project license**.

This is intentional during the architecture/foundation stage. DMR networking implementations and vocoder-related projects use several different licenses, and YWD-DMR should not accidentally choose a license that conflicts with code we may later decide to reuse or adapt.

## Until a license is selected

- Do not copy source code from another DMR, vocoder, or radio project into YWD-DMR merely because the code is public on GitHub.
- Before adapting external code, identify its license and document the source/provenance.
- Prefer independently implemented interfaces and protocol documentation while licensing decisions are still open.
- A third-party vocoder plugin may have its own license because vocoders are deliberately out-of-process components, but plugin authors are responsible for their own licensing and legal obligations.

The previous `LICENSE` file in the initial repository foundation contained the Unlicense even though the README stated that no final license had been selected. That accidental declaration was removed from the `dev` branch so the repository matches the actual project decision.

A real open-source license should be selected before the first public release intended for broad reuse/contribution, and this page, the root README, contribution rules, and release documentation must be updated together when that decision is made.
