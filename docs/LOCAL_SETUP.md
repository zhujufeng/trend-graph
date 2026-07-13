# 本地开发环境搭建（阶段 0 配套）

## 一、必装软件清单

| 软件 | 版本要求 | 用途 | 安装指引 |
|---|---|---|---|
| Git | ≥ 2.40 | 版本控制 | https://git-scm.com/downloads |
| Go | ≥ 1.22 | 后端语言 | https://go.dev/dl/ |
| Node.js | ≥ 20 LTS | 前端运行时 | https://nodejs.org/ |
| PostgreSQL | ≥ 15 | 生产数据库 | https://www.postgresql.org/download/ |
| Docker | ≥ 24 | 后期部署用 | https://docs.docker.com/get-docker/ |
| Docker Compose | v2 内置 | 后期部署用 | 同上 |

> 阶段 0 / 1 / 3 / 4 只用到 Git + Go + Node。PostgreSQL 在阶段 2 才需要，Docker 在阶段 9 才需要，可以先不装。

## 二、检查安装是否成功

依次执行以下命令，能打印版本号就 OK：

```bash
git --version          # >= 2.40
go version            # >= 1.22
node --version        # >= v20
npm --version         # 自带，>= 10
psql --version        # 阶段 2 才要求
docker --version      # 阶段 9 才要求
docker compose version
```

## 三、Go 国内代理（强烈建议国内用户配置）

加速依赖下载：

```bash
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

## 四、npm 国内镜像（可选）

```bash
npm config set registry https://registry.npmmirror.com
```

## 五、克隆你的仓库到本地

假设你已经在 GitHub 创建了名为 `trend-graph` 的空仓库（不要勾选自动生成 README，因为我们已经有自己的）。

```bash
git clone https://github.com/<你的GitHub用户名>/trend-graph.git
cd trend-graph
```

然后把本项目骨架里的所有文件覆盖过去，再：

```bash
git add .
git commit -m "feat: stage 0 - project scaffold"
git push origin main
```

## 六、验证 Go 环境能跑通

```bash
cd backend
go mod tidy
go run ./cmd/server
# 应该看到: trend-graph backend 启动于 http://localhost:8080
```

## 七、常见问题

**Q：`go mod tidy` 报网络错误？**
A：执行了第五步的 GOPROXY 配置后再试。

**Q：8080 端口被占用？**
A：临时改端口：`PORT=9090 go run ./cmd/server`，后期我们会用配置文件统一管理。

**Q：用 Windows 怎么办？**
A：完全 OK，但 path 分隔符用 `\` 而不是 `/`。推荐用 WSL2，体验和 Linux 一致。