import type { Config } from "tailwindcss";
import plugin from "tailwindcss/plugin";

export default {
  content: ["./index.html", "./src/**/*.{vue,ts,tsx}"],
  theme: {
    extend: {
      colors: {
        accent: {
          DEFAULT: "#0f766e",
          hover: "#115e59",
          light: "#ecfdf5",
          border: "#99f6e4",
        },
        status: {
          success: { DEFAULT: "#10b981", bg: "#ecfdf5", border: "#a7f3d0" },
          error:   { DEFAULT: "#ef4444", bg: "#fef2f2", border: "#fecaca" },
          warning: { DEFAULT: "#f59e0b", bg: "#fffbeb", border: "#fde68a" },
          info:    { DEFAULT: "#0ea5e9", bg: "#eff6ff", border: "#bfdbfe" },
          neutral: { DEFAULT: "#64748b", bg: "#f8fafc", border: "#e2e8f0" },
        },
        icon: {
          teal:   "#0f766e",
          orange: "#ea580c",
          yellow: "#ca8a04",
          purple: "#0ea5a1",
          pink:   "#db2777",
          gray:   "#64748b",
        },
        surface: {
          card:       "rgba(255,255,255,0.9)",
          "card-hover": "rgba(248,250,252,0.96)",
          page:       "#f3f7fb",
          border:     "rgba(148,163,184,0.22)",
        },
        text: {
          primary:   "#0f172a",
          secondary: "#475569",
          muted:     "#64748b",
        },
        sidebar: {
          text:        "#94a3b8",
          "text-hover":"#e2e8f0",
          icon:        "#64748b",
          "icon-active":"#f8fafc",
          "item-active":"#f8fafc",
          accent:      "#2dd4bf",
        },
      },
      fontFamily: {
        base:    ["Manrope", "-apple-system", "BlinkMacSystemFont", "Segoe UI", "system-ui", "sans-serif"],
        display: ["Space Grotesk", "Manrope", "sans-serif"],
        mono:    ["JetBrains Mono", "Fira Code", "ui-monospace", "monospace"],
      },
      borderRadius: {
        sm:   "10px",
        md:   "14px",
        lg:   "20px",
        card: "20px",
        full: "9999px",
      },
      boxShadow: {
        card:         "0 18px 38px rgba(15,23,42,0.08), 0 6px 18px rgba(15,23,42,0.05)",
        "card-hover": "0 24px 48px rgba(15,23,42,0.12), 0 10px 26px rgba(15,23,42,0.08)",
        "btn-primary":      "0 12px 24px rgba(15,118,110,0.18)",
        "btn-primary-hover":"0 16px 28px rgba(15,118,110,0.24)",
        input:        "inset 0 1px 2px rgba(15,23,42,0.02)",
        "input-focus":"0 0 0 4px rgba(15,118,110,0.12), inset 0 1px 2px rgba(15,23,42,0.02)",
      },
      zIndex: {
        dropdown:       "100",
        sticky:         "200",
        fixed:          "300",
        "modal-backdrop":"400",
        modal:          "500",
        popover:        "600",
        tooltip:        "700",
      },
      transitionTimingFunction: {
        spring: "cubic-bezier(0.22, 1, 0.36, 1)",
        smooth: "cubic-bezier(0.4, 0, 0.2, 1)",
        fast:   "cubic-bezier(0.4, 0, 0.2, 1)",
      },
      maxWidth: {
        page: "1360px",
      },
      width: {
        sidebar: "272px",
      },
      spacing: {
        "xs": "4px",
        "sm": "8px",
        "md": "16px",
        "lg": "24px",
        "xl": "32px",
        "2xl": "48px",
      },
    },
  },
  plugins: [
    plugin(({ addComponents, addUtilities }) => {
      addComponents({
        ".glass": {
          "backdrop-filter": "blur(18px) saturate(140%)",
          "-webkit-backdrop-filter": "blur(18px) saturate(140%)",
          "background-clip": "padding-box",
        },
        ".btn-primary-gradient": {
          "background": "linear-gradient(135deg, #0f766e 0%, #14b8a6 100%)",
        },
        ".btn-primary-gradient-hover": {
          "background": "linear-gradient(135deg, #115e59 0%, #0f766e 100%)",
        },
        ".sidebar-bg": {
          "background": "linear-gradient(180deg, #09111d 0%, #0c1626 50%, #0a1220 100%)",
        },
      });
      addUtilities({
        ".text-balance": { "text-wrap": "balance" },
      });
    }),
  ],
} satisfies Config;
