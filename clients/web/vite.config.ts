import { createRequire } from "node:module";
import { dirname } from "node:path";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const locate = createRequire(import.meta.url).resolve;

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "EDUPILOT_");
  const backend = env.EDUPILOT_API_PROXY || "http://127.0.0.1:3000";
  return {
    base: "./",
    publicDir: dirname(dirname(locate("@edupilot/core/brand/web/assets/favicon.svg"))),
    resolve: { dedupe: ["react", "react-dom"] },
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
