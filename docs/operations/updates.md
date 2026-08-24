# Updates and Rollback

Normal YWD-DMR updates should eventually be one click in the WebUI.

The updater is based on the appliance approach proven in YWD-Hotspot, simplified for YWD-DMR:

1. Check GitHub release metadata for the selected channel.
2. Download the release package over HTTPS.
3. Verify the published package checksum.
4. Create a protected pre-update backup.
5. Leave user/local vocoder plugins untouched.
6. Install the new release into a separate version directory.
7. Run configuration/database migrations against a rollback-safe copy.
8. Switch the `current` release atomically.
9. Start the new daemon.
10. Run application health checks and a post-update checkpoint.
11. Mark the release good only after the checkpoint passes.
12. Automatically restore the previous release/config if the new version fails.

Production devices should consume tagged GitHub releases rather than blindly running `git pull` against an active installation.

Planned channels are Stable, Beta, and Development. Stable should be the normal-user default once stable releases exist.

The updater must preserve `/var/lib/ywd-dmr/plugins/` and never overwrite a locally managed plugin just because the core was updated.
