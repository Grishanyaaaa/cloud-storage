import type { Config } from "tailwindcss";
import animate from "tailwindcss-animate";

export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: {
          0: "var(--bg-0)",
          1: "var(--bg-1)",
          2: "var(--bg-2)",
          3: "var(--bg-3)",
          4: "var(--bg-4)",
        },
        border: {
          1: "var(--border-1)",
          2: "var(--border-2)",
        },
        fg: {
          1: "var(--fg-1)",
          2: "var(--fg-2)",
          3: "var(--fg-3)",
        },
        accent: {
          1: "var(--accent-1)",
          "1h": "var(--accent-1-hover)",
          "1p": "var(--accent-1-pressed)",
          soft: "var(--accent-1-soft)",
        },
        danger: {
          DEFAULT: "var(--danger)",
          soft: "var(--danger-soft)",
        },
        success: "var(--success)",
        warning: "var(--warning)",
      },
      borderRadius: {
        sm: "var(--radius-1)",
        md: "var(--radius-1)",
        lg: "var(--radius-2)",
        xl: "var(--radius-3)",
      },
      fontFamily: {
        sans: [
          "Inter",
          "SF Pro Text",
          "system-ui",
          "-apple-system",
          "Segoe UI",
          "Roboto",
          "sans-serif",
        ],
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
    },
  },
  plugins: [animate],
} satisfies Config;
