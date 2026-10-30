// FranklyBuilds palette. Keep these overrides separate from upstream components.
export const primary = {
  50: '#fcf5f1', 100: '#f7e9e1', 200: '#efd2c3', 300: '#e3b09b',
  400: '#d59075', 500: '#cc785c', 600: '#a9583e', 700: '#8d4632',
  800: '#713b2d', 900: '#5c3328', 950: '#321a14'
}

export const warmEffects = {
  glow: '0 0 20px rgba(169, 88, 62, 0.2)',
  glowLarge: '0 0 30px rgba(169, 88, 62, 0.3)',
  gradient: 'linear-gradient(135deg, #cc785c 0%, #a9583e 100%)',
  mesh: 'none'
}

// Warm neutral ramps keep upstream Tailwind utilities compatible with the theme.
export const neutral = {
  50: '#faf9f5', 100: '#f5f0e8', 200: '#e6dfd8', 300: '#d2c9be',
  400: '#8e8b82', 500: '#6c6a64', 600: '#595750', 700: '#3d3d3a',
  800: '#252523', 900: '#141413', 950: '#0e0e0d'
}
export const darkNeutral = {
  ...neutral, 600: '#514d47', 700: '#3b3833', 800: '#252320',
  900: '#1f1e1b', 950: '#181715'
}
