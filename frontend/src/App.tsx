// App.tsx 是 React 应用的根组件。
//
// 也用函数组件（React 19 推荐）。
// 目前只渲染一个 HotListPage，阶段 6+ 会加路由（不同页面）。

import { HotListPage } from './pages/HotListPage'

// 这里的 export function 写法（命名导出），main.tsx 用具名导入 { App }。
// 也可以用 export default 让 main.tsx 用 default 导入，
// 这里为了一致性统一用具名导出。
export function App() {
  return <HotListPage />
}