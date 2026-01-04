/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#fdf4ff',
          100: '#fae8ff',
          200: '#f5d0fe',
          300: '#f0abfc',
          400: '#e879f9',
          500: '#d946ef',
          600: '#c026d3',
          700: '#a21caf',
          800: '#86198f',
          900: '#701a75',
        },
        cinema: {
          dark: '#0f0f1a',
          darker: '#08080f',
          accent: '#e11d48',
          gold: '#fbbf24',
          screen: '#1e293b',
        }
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'glow': 'glow 2s ease-in-out infinite alternate',
      },
      keyframes: {
        glow: {
          '0%': { boxShadow: '0 0 5px #e11d48, 0 0 10px #e11d48' },
          '100%': { boxShadow: '0 0 20px #e11d48, 0 0 30px #e11d48' },
        }
      }
    },
  },
  plugins: [],
}
