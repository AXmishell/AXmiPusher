import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 管理员后台: 构建时用相对路径 base('./'), 前端路由与资源引用均相对当前前缀。
// 这样部署在任意 /{random}/ 前缀下都可用, 且支持运行期轮换 admin 路径(无需重新构建)。
// 开发时用独立端口, 通过 base 环境变量模拟生产路径。
export default defineConfig({
  plugins: [react()],
  base: process.env.MP_ADMIN_BASE || './',
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
