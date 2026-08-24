# Frontend Branding and Asset Rules

YWD-DMR uses the same general visual family as YWD-Hotspot: dark radio-console surfaces, cyan/blue RF glow, silver/metallic structure, and restrained magenta highlights.

The goal is a modern radio appliance UI, not a generic admin template.

## Source versus runtime assets

Full-resolution source artwork belongs under `artwork/source/`. The WebUI should use smaller WebP derivatives under `web/assets/brand/`.

Initial target derivatives:

| Asset | Target size | Purpose |
| --- | ---: | --- |
| `logo.webp` | about 384 × 384 | About/setup/dashboard feature use |
| `logo-small.webp` | about 128 × 128 | compact header/mobile/icon-like use |
| `banner.webp` | about 1024 px wide | desktop/tablet header/hero use |

These are targets, not a rigid protocol. If the source artwork changes, choose dimensions that preserve clarity while keeping downloads small.

## Performance rule

Branding must not become a Pi Zero tax. Avoid serving multi-megabyte PNG/JPEG originals in normal dashboard pages. Prefer WebP/AVIF where browser support and build simplicity make sense, lazy-load noncritical artwork, and do not decode oversized images just to render them at a small CSS size.

## Responsive use

- Phones should prefer the compact logo or a cropped/contained banner rather than forcing a wide banner to overflow.
- Desktop/tablet layouts may use the full banner.
- Artwork is decorative branding; navigation and status must remain understandable if an image fails to load.
- Important RX/TX/error state must never be conveyed only by color or artwork.

## Accessibility

The UI must support large text, keyboard use, useful alternative text, reduced motion, and high-contrast operation. Branding should enhance the radio feel without making controls harder to read.

## Documentation rule

Material changes to the branding, asset paths, generated sizes, or frontend visual system must update this page and any affected user-facing screenshots/help pages in the same change.
