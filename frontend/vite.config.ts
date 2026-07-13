// Vite 配置文件。
//
// Vite 是现代前端构建工具：开发时秒级热更新（HMR），生产时打包优化。
// 这个配置文件告诉 Vite：
//   1) 用 React 插件（处理 JSX/TSX）
//   2) 用 Tailwind v4 插件（处理 CSS）
//   3) 开发时把 /api 请求代理到后端 8080，避免跨域
//
// import 用 ES Module 语法（type: "module" 在 package.json 里设过）。

// defineConfig 让 IDE 有自动补全和类型检查
import { defineConfig } from 'vite'
// react() 插件让 Vite 能处理 .jsx/.tsx 文件
import react from '@vitejs/plugin-react'
// Tailwind v4 官方 Vite 插件，省去 PostCSS 配置
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  // 插件数组：Vite 会按顺序应用
  plugins: [react(), tailwindcss()],

  // 开发服务器配置
  server: {
    // 监听端口，5173 是 Vite 默认端口
    port: 5173,
    // 监听所有网卡，方便从局域网访问
    host: true,

    // 代理配置：让浏览器发出的请求转发给后端
    // 解决开发时的跨域问题（前端 5173，后端 8080）
    proxy: {
      // 任何浏览器以 /api 开头的请求会被代理到 http://localhost:8080
      '/api': {
        target: 'http://localhost:8080',
        // 修改 origin header 为 target，避免后端 CORS 校验失败
        changeOrigin: true,
      },
      // WebSocket 代理：/ws 转发到后端
      // ws: true 让 Vite 处理 WebSocket 协议升级
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})