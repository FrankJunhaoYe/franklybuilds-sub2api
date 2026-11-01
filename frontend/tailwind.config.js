import {
  primary,
  warmEffects,
  neutral,
  darkNeutral,
  teal,
  plum,
  success,
  warning,
  error,
  orange
} from './src/franklybuilds/theme.js'

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary,
        gray: neutral,
        accent: neutral,
        dark: darkNeutral,

        // Upstream markup uses stock hue utilities throughout. Remapping the
        // families here warms the whole application without touching that
        // markup, so upstream merges stay clean. See theme.js for the ramps.
        white: '#fffdf9',
        slate: neutral,
        zinc: neutral,
        stone: neutral,
        neutral,

        // Cool accents collapse onto one warm teal.
        blue: teal,
        sky: teal,
        cyan: teal,
        indigo: teal,
        teal,

        // Purple-family badges become plum, still distinct from coral.
        purple: plum,
        violet: plum,
        fuchsia: plum,
        pink: plum,
        rose: error,

        // Semantic families keep their meaning in warm form.
        green: success,
        emerald: success,
        lime: success,
        amber: warning,
        yellow: warning,
        orange,
        red: error
      },
      fontFamily: {
        sans: [
          'Anthropic Sans Text',
          'Frankly Noto Sans SC',
          'PingFang SC',
          'Microsoft YaHei',
          'Noto Sans CJK SC',
          'Noto Sans SC',
          'Source Han Sans SC',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      // Elevation is color-block first, shadow rare. The whole stock scale
      // collapses onto one faint warm shadow so upstream shadow-lg /
      // shadow-2xl markup stops stacking depth the system does not use.
      boxShadow: {
        none: 'none',
        sm: '0 1px 2px rgba(20, 20, 19, 0.04)',
        DEFAULT: '0 1px 3px rgba(20, 20, 19, 0.06)',
        md: '0 1px 3px rgba(20, 20, 19, 0.06)',
        lg: '0 1px 3px rgba(20, 20, 19, 0.08)',
        xl: '0 1px 3px rgba(20, 20, 19, 0.08)',
        '2xl': '0 2px 6px rgba(20, 20, 19, 0.10)',
        inner: 'inset 0 1px 2px rgba(20, 20, 19, 0.04)',
        glass: '0 1px 3px rgba(20, 20, 19, 0.06)',
        'glass-sm': '0 1px 2px rgba(20, 20, 19, 0.04)',
        glow: warmEffects.glow,
        'glow-lg': warmEffects.glowLarge,
        card: '0 1px 3px rgba(20, 20, 19, 0.04)',
        'card-hover': '0 1px 3px rgba(20, 20, 19, 0.08)',
        'inner-glow': 'none'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': warmEffects.gradient,
        'gradient-dark': 'linear-gradient(135deg, #1f1e1b 0%, #181715 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(250,249,245,0.10) 0%, rgba(250,249,245,0.05) 100%)',
        'mesh-gradient': warmEffects.mesh
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: warmEffects.glow },
          '100%': { boxShadow: warmEffects.glowLarge }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      // Radius is hierarchical in the reference: 8px controls, 12px cards,
      // 16px largest containers. Upstream uses rounded-lg for controls and
      // rounded-xl/2xl/3xl for panels, so the scale compresses onto those
      // three steps instead of running to 24px and beyond.
      borderRadius: {
        none: '0',
        sm: '4px',
        DEFAULT: '6px',
        md: '8px',
        lg: '8px',
        xl: '12px',
        '2xl': '12px',
        '3xl': '16px',
        '4xl': '16px',
        full: '9999px'
      }
    }
  },
  plugins: []
}
