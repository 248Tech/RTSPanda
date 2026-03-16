import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
      colors: {
        base: '#09090b',
        surface: '#18181b',
        card: {
          DEFAULT: '#1c1c20',
          hover: '#27272a',
        },
        border: {
          DEFAULT: '#3f3f46',
          muted: '#2d2d32',
        },
        'text-primary': '#fafafa',
        'text-muted': '#a1a1aa',
        'text-subtle': '#52525b',
        'status-online': '#22c55e',
        'status-offline': '#ef4444',
        'status-connecting': '#f59e0b',
        accent: {
          DEFAULT: '#2563eb',
          hover: '#1d4ed8',
        },
      },
      boxShadow: {
        'glow-accent': '0 0 0 1px rgba(37,99,235,0.4), 0 0 16px rgba(37,99,235,0.15)',
        'modal': '0 25px 50px rgba(0,0,0,0.7), 0 0 0 1px rgba(255,255,255,0.05)',
      },
    },
  },
  plugins: [],
} satisfies Config
