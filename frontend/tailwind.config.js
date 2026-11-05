/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          '"Noto Sans Arabic"',
          '"Noto Kufi Arabic"',
          '"Noto Naskh Arabic"',
          '"Segoe UI"',
          'Tahoma',
          'Arial',
          'sans-serif',
        ],
      },
      colors: {
        // Catppuccin Latte palette mapped onto the existing semantic utility
        // names so every screen shares one light theme.
        slate: {
          50: '#f7f8fa',
          100: '#4c4f69',
          200: '#4c4f69',
          300: '#5c5f77',
          400: '#6c6f85',
          500: '#7c7f93',
          600: '#8c8fa1',
          700: '#acb0be',
          800: '#ccd0da',
          900: '#e6e9ef',
          950: '#eff1f5',
        },
        emerald: {
          100: '#1f5e16',
          200: '#256b19',
          300: '#2d7d1f',
          400: '#40a02b',
          500: '#40a02b',
          950: '#11111b',
        },
        rose: {
          100: '#7d0a22',
          200: '#8f0b27',
          300: '#a30d2d',
          400: '#e64553',
          500: '#d20f39',
          950: '#11111b',
        },
        amber: {
          100: '#6f4a00',
          200: '#7d5300',
          300: '#8a5a00',
          400: '#df8e1d',
          500: '#df8e1d',
          950: '#11111b',
        },
        sky: {
          100: '#155e6b',
          200: '#176b7a',
          300: '#1a7f8f',
          400: '#04a5e5',
          500: '#209fb5',
          950: '#11111b',
        },
        violet: {
          100: '#5b21b6',
          200: '#6425c9',
          300: '#6c2bd9',
          400: '#7c3aed',
          500: '#8839ef',
          950: '#11111b',
        },
      },
      boxShadow: {
        panel: '0 1px 2px rgba(76, 79, 105, 0.08), 0 16px 48px -24px rgba(76, 79, 105, 0.28)',
      },
      keyframes: {
        'fade-in': {
          from: { opacity: '0', transform: 'translateY(4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
      },
      animation: {
        'fade-in': 'fade-in 140ms ease-out',
      },
    },
  },
  plugins: [],
};
