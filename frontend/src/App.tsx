// App.tsx 是 React 应用的根组件。
//
// 阶段 8 引入"页面切换"：
//   - 用 useState 管理 currentPage
//   - 'list' 显示热点列表页
//   - 'graph' 显示关联图谱页
//
// 这里没用 react-router 是因为本项目只有两个页面，状态切换足够；
// 后续如果加更多页面再引入路由库。

import { useState } from 'react'
import { HotListPage } from './pages/HotListPage'
import { GraphPage } from './pages/GraphPage'

type Page = 'list' | 'graph'

export function App() {
  const [page, setPage] = useState<Page>('list')
  // 切换到图谱页时可选带一个 keyword
  const [graphKeyword, setGraphKeyword] = useState<string>('')

  // 从列表页跳到图谱页时传入当前搜索的关键词
  const goGraph = (keyword?: string) => {
    if (keyword) setGraphKeyword(keyword)
    setPage('graph')
  }

  if (page === 'graph') {
    return <GraphPage initialKeyword={graphKeyword} onBack={() => setPage('list')} />
  }
  return <HotListPage onNavigateGraph={goGraph} />
}