import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return;
          }
          if (id.includes("react-router") || id.includes("react-dom") || id.includes("\\react\\") || id.includes("/react/")) {
            return "react-vendor";
          }
          if (id.includes("antd") || id.includes("@ant-design")) {
            return "antd-vendor";
          }
          if (id.includes("axios") || id.includes("dayjs")) {
            return "shared-vendor";
          }
          return "vendor";
        },
      },
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
      "/swagger": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
  },
});
