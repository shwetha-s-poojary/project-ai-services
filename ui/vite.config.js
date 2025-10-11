import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default ({ mode }) => {
  // Load environment variables based on the current mode
  const env = loadEnv(mode, process.cwd(), "");

  return defineConfig({
    plugins: [react()],
    server: {
      proxy: {
        "/backend": {
          target: env.VITE_BACKEND_SERVER_URL || "http://localhost:8000", // Backend server
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/backend/, '/api/v1'),
        },
      },
    },
  });
};
