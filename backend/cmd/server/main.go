// 这是一个 Go 文件。Go 文件第一行永远是 package 声明。
// `package main` 表示这是一个可执行程序（而不是被别人 import 的库）。
// 程序入口 main.go 所在的目录约定为 package main。
package main

// import 用来引入其他包。这里只用 Go 标准库，没有第三方依赖。
// - fmt: 格式化输入输出（打印日志）
// - net/http: 标准库 HTTP 服务端
// - os: 读取环境变量
// - log: 日志输出
import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// main 是程序入口函数。Go 的程序从这里开始执行。
func main() {
	// 1. 打印欢迎信息，让用户知道服务起来了
	fmt.Println("============================================")
	fmt.Println("  trend-graph backend")
	fmt.Println("  AI 热点监控 + 关联图谱工具")
	fmt.Println("  技术栈: Go + TypeScript")
	fmt.Println("============================================")

	// 2. 从环境变量读端口，没设置就用默认 8080
	// os.Getenv 拿不到会返回空字符串 ""，所以用 if 判断
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 3. 注册一个最简单的路由: GET /health
	// http.HandleFunc 是标准库的写法: 路径 + 处理函数
	// 后面阶段 1 我们会换成 Gin 框架，写起来更简洁
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// w 用来写响应, r 是请求对象
		// w.Write 写字节切片 ([]byte)，所以字符串要转换
		_, _ = w.Write([]byte(`{"status":"ok","service":"trend-graph","version":"0.0.1"}`))
	})

	// 4. 启动 HTTP 服务
	// http.ListenAndServe 会阻塞当前 goroutine，直到服务停止
	// http.StatusOK 是 200 状态常量
	addr := ":" + port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Println("访问 http://localhost:" + port + "/health 查看健康检查")
	fmt.Println("按 Ctrl+C 退出")

	// 如果启动失败（比如端口被占用），会返回 error，log.Fatal 会打印并退出
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}