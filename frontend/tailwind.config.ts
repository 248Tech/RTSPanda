import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        base: '#0f1117',
        card: {
          DEFAULT: '#1a1d27',
          hover: '#22263a',
        },
        border: {
          DEFAULT: '#2e3150',
        },
        'text-primary': '#e2e8f0',
        'text-muted': '#64748b',
        'status-online': '#22c55e',
        'status-offline': '#ef4444',
        'status-connecting': '#f59e0b',
        accent: '#6366f1',
      },
    },
  },
  plugins: [],
} satisfies Config

