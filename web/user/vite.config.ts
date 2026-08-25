import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 本地开发代理到后端 API(只代理 /api/v1, 避免误伤 /api-keys 等 SPA 路由)。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
  },
});
