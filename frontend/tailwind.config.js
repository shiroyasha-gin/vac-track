/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          '"IBM Plex Sans Arabic"',
          '"Noto Sans Arabic"',
          '"Noto Kufi Arabic"',
          '"Segoe UI"',
          'Tahoma',
          'Arial',
          'sans-serif',
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
      colors: {
        // A calm paper-and-ink scale. 950 is the card surface, 900 the page
        // ground, 100-600 ink from headline to fine print, and 700-800 lines
        // and raised surfaces. Keeping the utility names stable means every
        // screen shares one contrast system.
        slate: {
          50: '#eef0f5',
          100: '#252a3d',
          200: '#33384e',
          300: '#454a61',
          400: '#565b71',
          500: '#5f6479',
          600: '#6a6e83',
          700: '#d4d9e3',
          800: '#e8ebf1',
          900: '#f2f4f8',
          950: '#ffffff',
        },
        emerald: {
          100: '#145238',
          200: '#196546',
          300: '#1f7a54',
          400: '#2f9e6e',
          500: '#1d7a52',
          600: '#186b48',
          950: '#0b1f18',
        },
        rose: {
          100: '#741f2b',
          200: '#8a2432',
          300: '#a32c40',
          400: '#c43a4c',
          500: '#b4323f',
          600: '#9d2936',
          950: '#2b0b10',
        },
        amber: {
          100: '#6f4a00',
          200: '#7d5300',
          300: '#8a5a00',
          400: '#b06b12',
          500: '#9a5c0f',
          600: '#84500d',
          950: '#2a1a04',
        },
        sky: {
          100: '#124f5a',
          200: '#155e6b',
          300: '#176b7a',
          400: '#1f7f92',
          500: '#24718a',
          600: '#1c5f74',
          950: '#0a2026',
        },
      },
      boxShadow: {
        panel: '0 1px 2px rgba(37, 42, 61, 0.04), 0 1px 3px rgba(37, 42, 61, 0.06)',
        floating: '0 18px 50px -20px rgba(37, 42, 61, 0.28), 0 8px 20px -12px rgba(37, 42, 61, 0.14)',
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
