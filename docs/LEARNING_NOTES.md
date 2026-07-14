# Go + TypeScript 学习笔记（基于 trend-graph 项目）

> 把整个项目里用到的 Go 和 TS 知识点串起来，按主题分类。

---

## 一、Go 知识图谱

### 1. 基础语法

| 概念 | 用法 | 项目位置 |
|---|---|---|
| package + import | `package main` / `import ("fmt"; ...)` | 所有文件 |
| 函数 | `func add(a, b int) int {}` | 所有 |
| 多返回值 | `func f() (int, error)` | 几乎所有 |
| 错误处理 | `if err != nil { return err }` | 所有 |
| defer | `defer resp.Body.Close()` | HTTP 调用 |
| 指针 | `&x` 取地址、`*p` 解引用 | struct 字段 |
| 空接口 | `any` / `interface{}` | gin.H |

### 2. 复合类型

| 类型 | 写法 | 用途 |
|---|---|---|
| 数组/切片 | `[]int`、`[]HotItem` | 列表 |
| map | `map[string]bool` | 集合 |
| struct | `type HotItem struct {...}` | 数据载体 |
| interface | `type Crawler interface {...}` | 抽象/多态 |
| 指针接收者 | `func (c *Crawler) Fetch` | 修改 struct |

### 3. struct tag

```go
type HotItem struct {
    Title string `json:"title" gorm:"type:varchar(500);not null;index"`
}
```

- json tag：序列化时字段名
- gorm tag：ORM 建表约束
- 反引号包围，分号分隔多个约束

### 4. 并发原语

```go
go func() { ... }()           // goroutine
var wg sync.WaitGroup         // 等待组
wg.Add(1); defer wg.Done(); wg.Wait()

var mu sync.Mutex             // 互斥锁
mu.Lock(); mu.Unlock()

var rw sync.RWMutex           // 读写锁
rw.RLock(); rw.RUnlock()      // 读锁
rw.Lock(); rw.Unlock()        // 写锁

ch := make(chan int, 8)       // 缓冲 chan
ch <- 1                        // 写
v := <-ch                      // 读

select {                       // 多路复用
case v := <-ch1: ...
case ch2 <- 2: ...
default: ...                   // 非阻塞
}
```

### 5. context.Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()  // 必调

// 传给下游
http.NewRequestWithContext(ctx, "GET", url, nil)
aiClient.Chat(ctx, req)
```

ctx 是 Go 并发的"取消信号"载体，超时自动 cancel，下游 HTTP 请求立即结束。

### 6. interface 隐式实现

```go
type Crawler interface {
    Source() string
    Fetch(keyword string, limit int) ([]HotItem, error)
}

type HackerNewsCrawler struct{}
func (c *HackerNewsCrawler) Source() string { return "hn" }
func (c *HackerNewsCrawler) Fetch(...) (...) { ... }
// 自动满足 Crawler，无需声明 implements
```

### 7. 依赖注入

```go
handler := api.NewHandler(hotRepo, an, wsHub, crawlers...)
```

把依赖作为参数传入构造函数，不用全局变量。好处：
- 测试时可以传 mock
- 改实现只改一处
- 依赖关系一目了然

### 8. 项目分层（cmd / internal / pkg）

```
cmd/server/main.go        # 入口
internal/                  # 私有包（外部不能 import）
  api/                    # HTTP 层
  crawler/                # 业务层
  store/                  # 数据层
  ai/                     # 基础设施层
  types/                  # 共享类型
```

### 9. 错误处理模式

```go
// 1. 多返回值 + if err
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing X failed: %w", err)  // %w 包裹原错误
}

// 2. 哨兵错误
var ErrTwitterNoToken = fmt.Errorf("...")
if errors.Is(err, ErrTwitterNoToken) { ... }

// 3. panic 仅用于启动期不可恢复错误
if cfg.DatabaseURL == "" {
    panic("DATABASE_URL 未设置")
}
```

### 10. 测试

```go
// _test.go 结尾文件
func TestXxx(t *testing.T) {
    t.Logf("...")
    t.Errorf("...")
    t.Fatalf("...")  // 失败立即停止
}

func TestMain(m *testing.M) {
    // 测试包入口，加载 .env 等
    os.Exit(m.Run())
}

// 表驱动测试
func TestAdd(t *testing.T) {
    cases := []struct{ a, b, want int }{
        {1, 2, 3},
        {-1, 1, 0},
    }
    for _, c := range cases {
        if got := Add(c.a, c.b); got != c.want {
            t.Errorf("...")
        }
    }
}
```

---

## 二、TypeScript + React 知识图谱

### 1. TS 类型基础

```ts
// 基本类型
let n: number = 1
let s: string = "hi"
let arr: number[] = [1, 2]
let tuple: [string, number] = ["a", 1]

// interface（对象形状）
interface HotItem {
  id: number
  title: string
  relevance?: number  // 可选字段
  readonly createdAt: string  // 只读
}

// 泛型
function first<T>(arr: T[]): T { return arr[0] }
const items: HotItem[] = [...]

// 联合类型 + 字面量类型
type Page = 'list' | 'graph'
type Status = 'idle' | 'loading' | 'success' | 'error'

// 类型断言（慎用）
const el = document.getElementById('root') as HTMLElement
const el2 = document.getElementById('root')!  // 非空断言
```

### 2. React 函数组件

```tsx
interface Props { title: string; onClick?: () => void }

function MyComponent({ title, onClick }: Props) {
  return <h1 onClick={onClick}>{title}</h1>
}

// 默认导出 vs 命名导出
export function MyComponent() {}        // 命名
export default function MyComponent() {}  // 默认
```

### 3. Hooks

```tsx
// useState 状态
const [count, setCount] = useState(0)
const [items, setItems] = useState<HotItem[]>([])

// useEffect 副作用
useEffect(() => {
  fetchData()
  return () => cleanup()  // cleanup 函数
}, [dep1, dep2])  // 依赖数组

// useRef 持久引用（不触发 rerender）
const wsRef = useRef<WebSocket | null>(null)
wsRef.current = new WebSocket(url)

// useCallback 缓存函数
const handleClick = useCallback(() => {
  // ...
}, [dep])

// useMemo 缓存计算结果
const stats = useMemo(() => items.filter(...), [items])

// 自定义 Hook
function useWebSocket(opts: Options) {
  // 内部用其他 Hooks
  return { connected, ... }
}
```

### 4. 条件渲染 + 列表渲染

```tsx
// 条件
{loading && <Spinner />}
{error ? <Error /> : <Content />}
{status === 'success' && <Data />}

// 列表
{items.map(item => (
  <Card key={item.id} item={item} />
))}
// key 必填，最好用稳定 ID（不用 index）
```

### 5. 事件处理

```tsx
function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
  setValue(e.target.value)
}

function handleClick(e: React.MouseEvent) {
  // ...
}

<button onClick={handleClick}>...</button>
<input onChange={handleChange} />
```

### 6. 组件通信

```tsx
// 父→子：props
<Child title="hello" />

// 子→父：回调
function Child({ onSelect }: { onSelect: (id: number) => void }) {
  return <button onClick={() => onSelect(1)}>...</button>
}

// 兄弟：状态提升到共同父
function Parent() {
  const [selected, setSelected] = useState(null)
  return <>
    <A onSelect={setSelected} />
    <B item={selected} />
  </>
}
```

### 7. async/await

```tsx
async function fetchData() {
  try {
    const res = await fetch('/api/hots')
    const data = await res.json()
    setItems(data)
  } catch (e) {
    setError(String(e))
  }
}
```

### 8. axios 拦截器

```ts
const client = axios.create({ baseURL: '/api', timeout: 60000 })

client.interceptors.response.use(
  (response) => response.data,  // 拉出 .data
  (error) => {
    // 统一错误处理
    return Promise.reject(new Error(msg))
  }
)

// 业务调用直接拿 body
const res = await client.get<unknown, ApiResponse<T>>('/hots')
return res.data  // ApiResponse<T> 的 .data 字段
```

### 9. TailwindCSS v4

```css
@import "tailwindcss";

@theme {
  --color-accent: #06b6d4;
}
```

```tsx
<button className="bg-accent text-white px-4 py-2 rounded-md hover:bg-accent-hover transition">
  Click
</button>
```

### 10. React Flow

```tsx
import ReactFlow, { Background, Controls, MiniMap, useNodesState, useEdgesState } from 'reactflow'

const [nodes, setNodes, onNodesChange] = useNodesState([])
const [edges, setEdges, onEdgesChange] = useEdgesState([])

<ReactFlow
  nodes={nodes}
  edges={edges}
  onNodesChange={onNodesChange}
  onNodeClick={onNodeClick}
  nodeTypes={nodeTypes}  // 必须组件外定义
  fitView
>
  <Background />
  <Controls />
  <MiniMap />
</ReactFlow>
```

---

## 三、Go vs TypeScript 对照表

| 概念 | Go | TypeScript |
|---|---|---|
| 类型声明 | `type X struct {...}` | `interface X {...}` |
| 接口 | 隐式实现 | 显式 `implements`（实际不用写） |
| 错误处理 | `if err != nil` | `try/catch` |
| 并发 | goroutine + channel | Promise + async/await |
| 模块导出 | 大写 = public | `export` |
| 包管理 | go mod | npm + package.json |
| 测试 | `_test.go` + `go test` | `*.test.ts` + `vitest/jest` |
| 编译 | `go build` → 二进制 | `tsc` → JS |
| 类型推断 | 弱（要显式） | 强（可省略） |
| 指针 | 有 | 没有（值/引用类型） |

---

## 四、踩坑大全

### Go 踩坑

1. **unused import/variable**：编译期报错，必须删
2. **`context.Background` 是函数不是类型**：`_ = context.Background()` 不是 `_ = context.Background{}`
3. **闭包陷阱**：循环变量传给 goroutine 要复制
4. **map 并发不安全**：多 goroutine 写要加锁
5. **ON CONFLICT 列必须和唯一索引完全一致**
6. **AutoMigrate 不删旧索引/字段**：改 schema 要手动 DROP
7. **`go test` 工作目录是测试源码所在目录**：读 .env 要算路径

### TypeScript 踩坑

1. **`Cannot find module 'X'`**：装包后 tsc 缓存没刷新，重跑 build
2. **`'X' is declared but never read`**：删未用变量
3. **`nodeTypes` 必须组件外定义**：放组件内会无限重建
4. **`useEffect` 死循环**：依赖数组写错
5. **闭包陷阱**：用 useRef 持有最新 callback
6. **axios 响应拦截器改返回类型**：第二个泛型才是真实返回类型
7. **Vite 不代理 ws://**：单独加 `/ws` + `ws: true`

---

## 五、推荐学习资源

- Go Tour: https://tour.go.dev
- Effective Go: https://go.dev/doc/effective_go
- gorilla/websocket 官方 chat 示例
- React 官方文档（新版）：https://react.dev
- TypeScript 中文手册：https://www.typescriptlang.org/zh/docs
- TailwindCSS v4 文档：https://tailwindcss.com
- React Flow 文档：https://reactflow.dev/learn