/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        surface: "#faf8ff",
        "surface-dim": "#d9d9e5",
        "surface-bright": "#faf8ff",
        "surface-container-lowest": "#ffffff",
        "surface-container-low": "#f3f3fe",
        "surface-container": "#f1f5f9",
        "surface-container-high": "#e7e7f3",
        "surface-container-highest": "#e1e2ed",
        "on-surface": "#191b23",
        "on-surface-variant": "#64748b",
        primary: "#004ac6",
        "on-primary": "#ffffff",
        "primary-container": "#2563eb",
        "on-primary-container": "#eeefff",
        secondary: "#505f76",
        "on-secondary": "#ffffff",
        tertiary: "#943700",
        "on-tertiary": "#ffffff",
        error: "#ba1a1a",
        "on-error": "#ffffff",
        "status-success": "#10b981",
        "status-warning": "#f59e0b",
        "status-danger": "#ef4444",
        "status-info": "#3b82f6",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "monospace"],
      },
    },
  },
  plugins: [],
}
