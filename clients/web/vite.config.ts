import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "EDUPILOT_");
  return {
    base: "./",
    resolve: { dedupe: ["react", "react-dom"] },
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      proxy: {
        "/api": {
          target: env.EDUPILOT_API_PROXY || "http://127.0.0.1:3000",
          changeOrigin: true,
        },
      },
    },
  };
});
