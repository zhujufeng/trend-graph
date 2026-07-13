// main.tsx 是前端项目的入口文件。
//
// 浏览器打开页面时，最先执行这段代码：
//   1) 找到 <div id="root" /> 节点
//   2) 把 React 应用挂载上去
//
// 用 React 19 的新 API：createRoot（旧版叫 ReactDOM.render）
// StrictMode 是开发期的"严格模式"，会双调用一些方法帮你发现潜在 bug。

// import 语法说明：
// import { xxx } from 'y' 表示从 y 模块导入具名导出
// import defaultX from 'y' 表示导入默认导出
// import './index.css' 没有变量名，是因为 CSS 文件本身有副作用（注册样式）

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { App } from './App'

// document.getElementById 返回 HTMLElement | null
// 末尾的 ! 是 TypeScript 的"非空断言"：
// 告诉编译器"我确定这个元素存在，不要当 null 处理"
// 这是 React 项目里非常常见的写法
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)