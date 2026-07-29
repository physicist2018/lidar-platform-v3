import { defineConfig } from "vite";

export default defineConfig({
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8091",
        changeOrigin: true,
      },
      "/login": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
      "/register": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
      "/verify": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
    },
  },
});
