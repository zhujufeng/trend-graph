# 阶段 4：React + TS 前端骨架 + 热点列表页

> 对应 commit：`feat: stage 4 - React + TS 前端骨架`

## 🎯 目标

- 用 Vite 起一个 React 19 + TypeScript 项目
- TailwindCSS v4 深色科技风
- 写热点列表页：信息源切换、AI 扩展、立即抓取、卡片网格

## 📚 学到的概念（TypeScript + React）

### 1. TS 接口与后端 JSON 对齐

```ts
export interface HotItem {
  id: number
  title: string
  url: string
  source: string
  relevance?: number  // ? = 可选字段
}
```

接口字段必须和后端 JSON 完全一致（驼峰命名），TS 编译期会检查类型。

### 2. useState 状态管理

```ts
const [items, setItems] = useState<HotItem[]>([])
const [loading, setLoading] = useState(false)
```

`useState<T>(initial)` 返回 `[当前值, 修改函数]`。修改时 React 自动重新渲染组件。

### 3. useEffect 副作用

```ts
useEffect(() => {
  listSources().then(setSources)
}, [])  // [] = 只在挂载时执行一次
```

依赖数组里放变量，变量变了会重新执行。空数组 = 只跑一次。

### 4. useCallback 缓存函数

```ts
const fetchList = useCallback(async () => {
  // ...
}, [activeSource, since, page])  // 依赖变了才重建函数
```

避免每次渲染都重建函数（性能优化 + 避免 useEffect 死循环）。

### 5. 自定义 Hook（虽然这阶段还没写）

约定：函数名以 `use` 开头。阶段 6 会写 `useWebSocket`。

### 6. axios 拦截器

```ts
client.interceptors.response.use(
  (response) => response.data,  // 拉出 .data
  (error) => Promise.reject(new Error(msg))  // 统一错误
)
```

业务调用 `client.get()` 直接拿到后端 body，省去每次 `.data` 链式。

### 7. Vite 代理解决跨域

```ts
// vite.config.ts
proxy: {
  '/api': { target: 'http://localhost:8080', changeOrigin: true }
}
```

开发时浏览器请求 `/api/hots` → Vite dev server → 后端 8080，无跨域问题。

### 8. TailwindCSS v4 配置

v4 不再需要 `tailwind.config.js`，所有配置在 CSS 里：

```css
@import "tailwindcss";

@theme {
  --color-accent: #06b6d4;
  --color-surface: #131826;
}
```

然后用 `bg-accent` `text-surface` 这种类名。

### 9. 条件渲染 + 列表渲染

```tsx
{loading && <Loader2 className="animate-spin" />}
{items.map((item) => <HotCard key={item.id} item={item} />)}
```

`&&` 短路做条件渲染。`.map` 必须给 `key` prop（React diff 用）。

### 10. 展示组件 vs 容器组件

- **展示组件**：`HotCard`/`SourceBadge` 只接收 props，不调 API，纯渲染
- **容器组件**：`HotListPage` 管状态、调 API、组合展示组件

这种分层让组件可复用、好测试。

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| 项目分层 | `frontend/src/{api,components,pages,types,hooks}/` |
| TS 类型 | `frontend/src/types/index.ts` |
| axios 客户端 | `frontend/src/api/index.ts` |
| 容器组件 | `frontend/src/pages/HotListPage.tsx` |
| 展示组件 | `frontend/src/components/HotCard.tsx` 等 |
| Tailwind 主题 | `frontend/src/index.css` |
| Vite 代理 | `frontend/vite.config.ts` |

## 🧪 验证

```bash
cd frontend
npm install
npm run build    # tsc -b && vite build
npm run dev      # 启动 dev server
# 浏览器 http://localhost:5173
```

## 🐛 踩坑

1. **`Cannot find module 'axios'`**：装包后 build 报错，可能是 node_modules 没装齐，删 `rm -rf node_modules package-lock.json` 重装
2. **`'X' is declared but never read`**：TS 严格 unused 检查，删掉未用变量
3. **Tailwind v4 不识别 `bg-surface`**：必须在 `@theme` 里定义 `--color-surface` 才能用

## 📝 一句话总结

React 函数组件 = useState 管状态 + useEffect 处理副作用 + 组件组合，TypeScript 编译期帮你抓类型错误。