# FranklyBuilds theme

Theme customizations live here to reduce conflicts when merging Sub2API updates.

- `theme.js`: every palette Tailwind consumes. Besides primary, warm gray and dark, this now remaps the stock hue families — `blue`/`sky`/`cyan`/`indigo` to one warm teal, `purple`/`violet`/`fuchsia`/`pink` to plum, and the semantic families to warm success/warning/error/orange ramps. Upstream markup uses those utility names thousands of times, so remapping here warms the whole application without editing that markup, and upstream merges stay clean. Changing any ramp affects existing utility classes application-wide.
- `theme.css`: font faces, shared component surfaces, navigation, authentication and page-specific adaptations. Shared modal/menu rules intentionally apply outside layout wrappers because Vue teleports overlays to the document body.

### Why the palette layer, not per-component overrides

Patching `bg-white` and cool-hue utilities selector by selector cannot keep up with the upstream surface area (424 `bg-white`, ~2,300 cool-hue utilities at the time of writing). Tokens cover all of it at once. Prefer adding a ramp or adjusting a stop over writing another narrow override.

### Contrast constraints baked into the ramps

Some stops deliberately depart from the reference, because the reference's marketing values fail WCAG AA in dense console use. Keep these when editing:

- Filled controls use the **600** coral (`#a9583e`, 5.06:1 with white) rather than the reference's `#cc785c` (3.28:1). `#cc785c` stays for focus rings and large coral areas, which carry no small text.
- `.btn-warning` uses amber **600**; amber 500 reaches only 3.21:1 behind white labels.
- `gray-400` is `#706d65`, not the reference's `muted-soft` `#8e8b82`. Upstream uses `text-gray-400` ~2,450 times for real content, where `#8e8b82` sits at 3.2:1 on cream. The lighter tone remains as `muted-soft` for decorative text. `gray-500`/`600` shifted down with it to keep four distinct steps.
- `darkNeutral` overrides 400/500 to **lighter** values (`#a09d96`, `#8e8b82`). In dark mode those stops carry secondary text *on* dark surfaces, so the light-mode darkening would invert their purpose and drop `dark:text-dark-400` to 3.2:1.
- `fonts/`: Anthropic Text assets supplied for the project, plus Noto Sans SC variable WOFF2 subsets and its OFL license. Anthropic redistribution authorization still needs confirmation.

Noto subsets cover U+4E00–9FFF in three ranges and CJK punctuation, Extension A, compatibility forms, full-width forms and available supplementary ideographs in the fourth. Uncovered characters use the configured system fallback. Four subsets total approximately 8.1 MB, so Chinese-heavy pages may still download several MB. Do not preload all subsets.

## Updating the UI

Use shared classes (`card`, `input`, `btn`, `table`, `modal-content`) first. For upstream components with independent markup, prefer a palette change in `theme.js`; fall back to a narrow semantic hook only when one component genuinely differs. Keep error, success and disabled states recognizable.

Design rules the system encodes, worth preserving:

- Depth is color-block first. The `boxShadow` scale collapses onto one faint warm shadow, so upstream `shadow-lg` / `shadow-2xl` cannot stack elevation the system does not use. Colored glow shadows (`shadow-primary-500/30` and similar) are removed, not re-tinted.
- Radius is hierarchical and capped: 8px controls, 12px cards, 16px largest containers.
- Coral is scarce. It belongs on primary actions and focus rings — not on active nav rows, which use a cream step with ink text.
- Display headings stay weight 400-500 with negative tracking; emphasis comes from size, never a bolder serif.
- No gradients, no `backdrop-blur`, no decorative blurred blobs.

Payment-provider brand colors (Stripe, Alipay, WeChat Pay, Airwallex) are intentionally left as-is — they identify third parties and are not ours to warm.

A caveat on the display face: `theme.css` previously named `Cormorant Garamond` and `EB Garamond`, but neither is bundled or loaded, so those headings always rendered Georgia. The stack now says Georgia explicitly. If the licensed Copernicus / Tiempos Headline or a self-hosted Cormorant is ever added, add the `@font-face` and prepend it to `.frankly-display`, `.frankly-auth-title`, `.frankly-home-title` and `.frankly-home-heading`.

After upstream updates, run type checking, the frontend suite and production build. Visually inspect login, both dashboards, operations, profile binding, a data table and a teleported dialog in light/dark mode and at mobile widths. Automated tests do not establish visual correctness or actual browser font loading.
