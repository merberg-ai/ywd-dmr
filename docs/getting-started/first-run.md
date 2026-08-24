# First-run Setup

The production first-run wizard is planned but not implemented yet.

The wizard will use large, plain-language steps:

1. Claim the new YWD-DMR installation using a one-time setup code.
2. Create the administrator account. There will be no default `admin/admin` password.
3. Enter callsign, DMR ID, and ESSID.
4. Select the DMR network and master server.
5. Test network credentials before saving them as active.
6. Choose and test microphone and speaker devices.
7. Detect or configure a vocoder backend, or continue safely in no-vocoder mode.
8. Run a complete station health check.
9. Finish setup and open the dashboard.

If a test fails, the wizard should explain what failed in normal language and let the user retry or continue when the failed component is optional.
