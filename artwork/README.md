# YWD-DMR Artwork

This directory is for original project artwork and branding sources.

## Rule

Keep original/full-resolution artwork here unchanged. The application should not serve large source images directly to browsers.

Recommended source layout:

```text
artwork/
  source/
    ywd-dmr-logo-original.png
    ywd-dmr-banner-original.png
```

Web-ready derivatives belong under `web/assets/brand/` and may be resized/re-encoded as needed for fast loading on Raspberry Pi Zero-class hardware.

The initial branding uses a black/dark background with cyan/blue RF glow, silver/metallic structure, and small magenta accents. Frontend colors should be derived from that family so YWD-DMR remains visually related to YWD-Hotspot without being an identical interface.
