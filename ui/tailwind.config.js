module.exports = {
  darkMode: 'class',
  content: [
    './src/**/*.{svelte,ts,js}',
    './public/**/*.html',
  ],
  theme: {
    extend: {
      colors: {
        // Legacy alias — keep so any not-yet-renamed usages stay valid. Maps
        // to the brand primary CSS variable.
        'yellow-thunder': 'rgb(var(--brand-primary-rgb) / <alpha-value>)',
        brand: {
          primary: 'rgb(var(--brand-primary-rgb) / <alpha-value>)',
          accent:  'rgb(var(--brand-accent-rgb)  / <alpha-value>)',
          dark:    'rgb(var(--brand-dark-rgb)    / <alpha-value>)',
        },
      },
      fontFamily: {
        rajdhani: ['Rajdhani', 'Arial Narrow', 'sans-serif'],
      }
    }
  }
}