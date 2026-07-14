// store 包测试入口：加载 ../../.env 让集成测试能读到 DEEPSEEK_API_KEY 等环境变量
package store

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain 在所有测试运行前调一次
func TestMain(m *testing.M) {
	// go test 工作目录是测试源码所在目录: backend/internal/store/
	// 想读 backend/.env 要向上两级
	_ = godotenv.Load("../../.env")
	os.Exit(m.Run())
}