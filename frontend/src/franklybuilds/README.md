# FranklyBuilds theme

Theme customizations live here to reduce conflicts when merging Sub2API updates.

- `theme.js`: primary, warm gray and dark palettes consumed by Tailwind. Changing gray affects existing upstream utility classes throughout the application.
- `theme.css`: font faces, shared component surfaces, navigation, authentication and page-specific adaptations. Shared modal/menu rules intentionally apply outside layout wrappers because Vue teleports overlays to the document body.
- `fonts/`: Anthropic Text assets supplied for the project, plus Noto Sans SC variable WOFF2 subsets and its OFL license. Anthropic redistribution authorization still needs confirmation.

Noto subsets cover U+4E00–9FFF in three ranges and CJK punctuation, Extension A, compatibility forms, full-width forms and available supplementary ideographs in the fourth. Uncovered characters use the configured system fallback. Four subsets total approximately 8.1 MB, so Chinese-heavy pages may still download several MB. Do not preload all subsets.

## Updating the UI

Use shared classes (`card`, `input`, `btn`, `table`, `modal-content`) first. For upstream components with independent markup, add a narrow semantic hook rather than overriding every `bg-white` or replacing status/payment-provider colors. Keep error, success and disabled states recognizable.

After upstream updates, run type checking, the frontend suite and production build. Visually inspect login, both dashboards, operations, profile binding, a data table and a teleported dialog in light/dark mode and at mobile widths. Automated tests do not establish visual correctness or actual browser font loading.
