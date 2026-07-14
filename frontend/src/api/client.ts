// client.ts 单独导出配置好的 axios 实例。
//
// 这样各业务模块（hots、keywords 等）都能复用，
// 不用每个文件都自己 axios.create()。

import axios from 'axios'

// 开发期间 baseURL 留空 → /api 走 Vite 代理 → http://localhost:8080/api
// 生产部署时通过 nginx 反代也走 /api
const client = axios.create({
  baseURL: '/api',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 响应拦截器：拉出 .data，简化业务调用
client.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response) {
      const data = error.response.data
      const msg = data?.error || `HTTP ${error.response.status}`
      return Promise.reject(new Error(`${msg}${data?.detail ? ': ' + data.detail : ''}`))
    }
    if (error.request) {
      return Promise.reject(new Error('网络错误或后端未响应'))
    }
    return Promise.reject(error)
  },
)

export default client