import { defineConfig } from "vite";

// In development, Vite proxies all API calls through nginx (https://localhost).
// Nginx handles SSL termination and routes to the appropriate backend service.
// The FRONTEND_URL (verify redirect) also points to the same host.

export default defineConfig({
  server: {
    proxy: {
      // Identity routes → nginx → identity:8090
      "/login": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      "/register": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      "/verify": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      "/refresh": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      "/logout": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      // Lidar API → nginx → lidar:8091
      "/api": {
        target: "https://localhost",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
