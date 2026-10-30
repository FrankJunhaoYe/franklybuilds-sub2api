// FranklyBuilds palette. Keep these overrides separate from upstream components.
//
// Upstream markup carries thousands of stock Tailwind hue utilities (bg-white,
// text-blue-600, bg-purple-100, shadow-lg...). Rather than rewrite that markup —
// which would conflict on every upstream merge — every hue family is remapped
// here to a warm equivalent. Utilities keep their names and their meaning; only
// the resolved color changes, so the whole application lands on one palette.
//
// Ramps hold their stop semantics: 50-200 chip and tint backgrounds, 300-500
// fills and borders, 600-700 text on light surfaces, 800-950 dark-mode surfaces.
// Every 600 stop clears 4.5:1 against the cream canvas (#faf9f5).

export const primary = {
  50: '#fcf5f1', 100: '#f7e9e1', 200: '#efd2c3', 300: '#e3b09b',
  400: '#d59075', 500: '#cc785c', 600: '#a9583e', 700: '#8d4632',
  800: '#713b2d', 900: '#5c3328', 950: '#321a14'
}

export const warmEffects = {
  // Color-block first, shadow rare: depth comes from cream-vs-dark surfaces.
  glow: '0 1px 3px rgba(20, 20, 19, 0.06)',
  glowLarge: '0 1px 3px rgba(20, 20, 19, 0.08)',
  gradient: 'linear-gradient(135deg, #cc785c 0%, #cc785c 100%)',
  mesh: 'none'
}

// Warm neutral ramps keep upstream Tailwind utilities compatible with the theme.
//
// The 400 stop is the reference's muted-soft (#8e8b82), which it reserves for
// fine print. Upstream uses text-gray-400 ~2,450 times for captions, empty
// states and icon buttons, where #8e8b82 would sit at 3.2:1 on cream — so the
// stop is darkened to #706d65, which clears 4.5:1 on every light surface it
// lands on (card 5.09, canvas 4.90, tinted band 4.55). The lighter tone stays
// available as `muted-soft` for genuinely decorative text.
// Darkening 400 would have collapsed it into 500, so 500 and 600 shift down
// too: four stops that each clear 4.5:1 and still read as separate steps.
export const neutral = {
  50: '#faf9f5', 100: '#f5f0e8', 200: '#e6dfd8', 300: '#d2c9be',
  400: '#706d65', 500: '#5f5d56', 600: '#4c4a45', 700: '#3d3d3a',
  800: '#252523', 900: '#141413', 950: '#0e0e0d',
  'muted-soft': '#8e8b82'
}
// In dark mode the same stops run the other way: dark-400 and dark-500 carry
// secondary text ON dark surfaces, so they must be lighter than the surface,
// not darker. Both are set to the reference's on-dark-soft family and clear
// 4.5:1 against dark-950 through dark-800.
export const darkNeutral = {
  ...neutral, 400: '#a09d96', 500: '#8e8b82', 600: '#514d47',
  700: '#3b3833', 800: '#252320', 900: '#1f1e1b', 950: '#181715'
}

// Cool families become one warm teal. Anchored on the reference's accent-teal
// (#5db8a6), darkened through the text stops to stay legible on cream.
export const teal = {
  50: '#f0f7f5', 100: '#dceeea', 200: '#bcded6', 300: '#92c9bd',
  400: '#6bb3a4', 500: '#4ca292', 600: '#2d7868', 700: '#26614f',
  800: '#214e42', 900: '#1d4138', 950: '#0f2520'
}

// Purple and violet families become a muted plum — still clearly distinct from
// coral, so status and tier badges keep encoding different information.
export const plum = {
  50: '#faf1f5', 100: '#f3e2eb', 200: '#e7c6d6', 300: '#d6a1ba',
  400: '#c17c9c', 500: '#a85e80', 600: '#8c4765', 700: '#733b53',
  800: '#5f3245', 900: '#4f2b3a', 950: '#2d1620'
}

// Semantic ramps stay recognizable as success / warning / error while warming
// toward the reference's #5db872, #d4a017 and #c64545.
export const success = {
  50: '#f1f7f1', 100: '#dfeede', 200: '#c0dcbf', 300: '#97c498',
  400: '#6cab71', 500: '#4e9759', 600: '#357a46', 700: '#2b6239',
  800: '#254f30', 900: '#20422a', 950: '#102416'
}

export const warning = {
  50: '#fbf6ea', 100: '#f6ecd2', 200: '#ecd79f', 300: '#dfbb66',
  400: '#d4a017', 500: '#b9871a', 600: '#8a6314', 700: '#6f4f13',
  800: '#5b4114', 900: '#4c3614', 950: '#291c08'
}

export const error = {
  50: '#fcf2f1', 100: '#f8e2e0', 200: '#f0c4c0', 300: '#e29b96',
  400: '#d06f6b', 500: '#c64545', 600: '#b03a3a', 700: '#8e2f2f',
  800: '#742a2a', 900: '#5f2626', 950: '#331010'
}

// Amber's companion warm tone from the reference (#e8a55a) carries the orange
// family, a step warmer than warning so the two stay separable.
export const orange = {
  50: '#fdf4ec', 100: '#fae7d5', 200: '#f3cda8', 300: '#e8a55a',
  400: '#d98a3c', 500: '#c07328', 600: '#a35a1e', 700: '#83491c',
  800: '#6b3c1b', 900: '#58331a', 950: '#2f190b'
}
