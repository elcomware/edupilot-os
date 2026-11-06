import {fileURLToPath} from "node:url";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "EDUPILOT_");
  const backend = env.EDUPILOT_API_PROXY || "http://127.0.0.1:3000";
  return {
    base: "./",
    resolve: { alias: {"@resources":fileURLToPath(new URL("../../resources/js",import.meta.url))}, dedupe: ["react", "react-dom"] },
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      proxy: {
        "/api": { target: backend, changeOrigin: true },
        "^/login(?:\\?|$)": { target: backend, changeOrigin: true },
        "/assets/": { target: backend, changeOrigin: true },
      },
    },
  };
});
