// API 客户端：把所有后端调用集中在一个文件。
//
// 为什么不直接在组件里 fetch？
//   - 统一加 baseURL、超时、错误处理
//   - 改后端地址只改一处
//   - 接口列表一目了然，方便团队协作
//   - 单测时可以 mock 这一层

// axios 是最流行的 HTTP 客户端，支持拦截器、超时、取消等
import axios, { type AxiosInstance } from 'axios'

// 引入项目内的类型
import type {
  ApiResponse,
  CrawlMeta,
  ExpandResult,
  HotItem,
  ListMeta,
  ListParams,
} from '../types'

// 创建 axios 实例，复用配置
//
// 注意开发期间用 Vite 代理：
//   浏览器请求 /api/hots → Vite dev server → http://localhost:8080/api/hots
// 所以 baseURL 留空（同源），让 Vite 代理接管
const client: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 60000, // 60 秒超时（AI 分析可能慢）
  headers: {
    'Content-Type': 'application/json',
  },
})

// ====== 拦截器 ======
// 请求拦截器：在请求发出前可以加 token 等
client.interceptors.request.use(
  (config) => {
    // 比如未来加登录：config.headers.Authorization = `Bearer ${token}`
    return config
  },
  (error) => Promise.reject(error),
)

// 响应拦截器：统一拉出 .data 简化调用
client.interceptors.response.use(
  (response) => response.data, // 直接返回 data，省去每次 .data 的链式
  (error) => {
    // 把 axios 错误统一包装成更易读的字符串
    if (error.response) {
      // 后端给了非 2xx 响应
      const data = error.response.data
      const msg = data?.error || `HTTP ${error.response.status}`
      return Promise.reject(new Error(`${msg}${data?.detail ? ': ' + data.detail : ''}`))
    }
    if (error.request) {
      // 请求发了但没响应（网络/超时）
      return Promise.reject(new Error('网络错误或后端未响应'))
    }
    // 请求构造失败
    return Promise.reject(error)
  },
)

// ====== 接口实现 ======

// 健康检查
export const health = async (): Promise<{ status: string }> => {
  // 注意：用相对路径 'http://localhost:8080/health' 也行，
  // 但我们走代理，所以 /api 外的健康检查单独写绝对路径
  return axios.get('/health').then((r) => r.data)
}

// 列出信息源
export const listSources = async (): Promise<string[]> => {
  return client.get<unknown, ApiResponse<string[]>>('/sources').then((r) => r.data)
}

// 触发一次抓取
export const triggerCrawl = async (
  source: string,
  keyword: string,
  limit = 10,
  analyze = false,
): Promise<{ items: HotItem[]; meta: CrawlMeta }> => {
  const params: Record<string, string | number | boolean> = { source, limit }
  if (keyword) params.keyword = keyword
  if (analyze) params.analyze = true
  // client.get 的第二个参数是配置对象，params 用 querystring
  // 注意：因为响应拦截器已经返回 .data，这里返回的就是 ApiResponse
  const res = await client.post<unknown, ApiResponse<HotItem[]>>('/crawl', null, {
    params,
  })
  return { items: res.data, meta: res.meta as unknown as CrawlMeta }
}

// 从数据库读热点列表
export const listHots = async (params: ListParams): Promise<{ items: HotItem[]; meta: ListMeta }> => {
  const res = await client.get<unknown, ApiResponse<HotItem[]>>('/hots', { params })
  return { items: res.data, meta: res.meta as unknown as ListMeta }
}

// 取单条热点
export const getHot = async (id: number): Promise<HotItem> => {
  const res = await client.get<unknown, ApiResponse<HotItem>>(`/hots/${id}`)
  return res.data
}

// 查询扩展
export const expandKeyword = async (keyword: string): Promise<string[]> => {
  const res = await client.post<unknown, ApiResponse<ExpandResult>>('/expand', { keyword })
  return res.data
}

// 对已入库的热点做 AI 分析
export const analyzeHot = async (id: number, keyword: string): Promise<AnalysisResult> => {
  const params: Record<string, string> = {}
  if (keyword) params.keyword = keyword
  const res = await client.post<unknown, ApiResponse<AnalysisResult>>(`/analyze/${id}`, null, {
    params,
  })
  return res.data
}

// AI 分析结果类型
export interface AnalysisResult {
  id: number
  title: string
  summary: string
  relevance: number
  isAuthentic: boolean
  entities: string[]
  reason: string
}