import path from "node:path";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import viteCompression from "vite-plugin-compression";

export default defineConfig({
  css: {
    preprocessorOptions: {
      scss: {
        api: "modern-compiler",
      },
    },
  },
  plugins: [
    tailwindcss(),
    vue(),
    AutoImport({
      imports: [
        "vue",
        "vue-router",
        "pinia",
        {
          "@/composables": [
            "useMessage",
            "useConfirm",
            "useErrorHandler",
          ],
          "@/shared/utils": [
            "formatDateTime",
            "formatBytes",
            "formatDuration",
            "debounce",
            "throttle",
            "copyToClipboard",
          ],
          "@/store/command-palette": [
            "useCommandPaletteStore"
          ]
        },
      ],
      dts: "src/auto-imports.d.ts",
    }),
    Components({
      include: [/\.vue$/, /\.vue\?vue/],
      extensions: ["vue"],
      dirs: ["src/components"],
      dts: "src/components.d.ts",
    }),
    viteCompression({
      verbose: true,
      disable: false,
      threshold: 10240,
      algorithm: "gzip",
      ext: ".gz",
    }),
    viteCompression({
      verbose: true,
      disable: false,
      threshold: 10240,
      algorithm: "brotliCompress",
      ext: ".br",
    }),
  ],
  resolve: {
    alias: [
      { find: "@", replacement: path.resolve(__dirname, "./src") },
    ]
  },
  server: {
    port: 5173,
    proxy: {
      "/api/v2": "http://localhost:8080",
      "/healthz": "http://localhost:8080"
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          "vue-vendor": ["vue", "vue-router", "pinia"],
          "scroller": ["vue-virtual-scroller"]
        }
      }
    }
  }
});
