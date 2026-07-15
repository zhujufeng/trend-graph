// store 包测试入口：加载 ../../.env 让集成测试能读到 DEEPSEEK_API_KEY 等环境变量
package store

import (
	"os"
	"strings"
	"testing"
)

// TestMain 在所有测试运行前调一次
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置；跳过需要 PostgreSQL 的集成测试")
	}
	if !strings.Contains(dsn, "trend_graph_test") {
		t.Fatalf("TEST_DATABASE_URL 必须指向专用 trend_graph_test 数据库")
	}
	return dsn
}
