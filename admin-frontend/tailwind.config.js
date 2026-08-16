/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        bordeaux: {
          50: '#fbf1f3',
          100: '#f6e0e5',
          200: '#edc3cd',
          300: '#de9aab',
          400: '#c96681',
          500: '#ab3f60',
          600: '#8b1e3f',
          700: '#771a38',
          800: '#641932',
          900: '#56182e',
          950: '#380c1b',
        },
      },
    },
  },
  plugins: [],
}
