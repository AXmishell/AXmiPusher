import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 管理员后台: 生产构建时 base 设为 /{random}/ (管理员随机路径)。
// 开发时用独立端口, 通过 base 环境变量模拟生产路径。
export default defineConfig({
  plugins: [react()],
  base: process.env.MP_ADMIN_BASE || '/',
  server: {
    port: 5174,
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
