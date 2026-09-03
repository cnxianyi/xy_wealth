import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

function parseAllowedHosts(value: string | undefined): string[] {
  return (value ?? "")
    .split(",")
    .map((host) => host.trim())
    .filter(Boolean);
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "XY_WEALTH_WEB_");
  const allowedHosts = parseAllowedHosts(env.XY_WEALTH_WEB_ALLOWED_HOSTS);

  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: "0.0.0.0",
      ...(allowedHosts.length > 0 ? { allowedHosts } : {}),
      proxy: {
        "/api": "http://127.0.0.1:8088",
        "/health": "http://127.0.0.1:8088",
        "/openapi.yaml": "http://127.0.0.1:8088",
      },
    },
    preview: {
      host: "0.0.0.0",
      ...(allowedHosts.length > 0 ? { allowedHosts } : {}),
      proxy: {
        "/api": "http://127.0.0.1:8088",
        "/health": "http://127.0.0.1:8088",
        "/openapi.yaml": "http://127.0.0.1:8088",
      },
    },
  };
});
