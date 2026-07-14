# trend-graph 部署指南（VPS 直接部署，不用 Docker）

> 本文档适用场景：腾讯云 / 阿里云 / AWS 等 Linux VPS（Ubuntu 22.04+），不用 Docker，直接用 systemd + nginx 部署。

## 一、部署架构

```
            ┌──────────────────────────────────┐
            │       VPS（你的云服务器）          │
            │                                    │
   443/HTTP │  ┌─────────┐                       │
Internet ───┼─▶│  nginx  │                       │
            │  └────┬────┘                       │
            │       │ 静态文件 + 反代              │
            │       ├──→ /var/www/trend-graph  (前端 dist)
            │       └──→ 127.0.0.1:8080        (后端服务)
            │                                    │
            │  ┌──────────────────────┐         │
            │  │ systemd: trend-graph  │         │
            │  │   ↳ ./trend-graph     │         │
            │  └──────────────────────┘         │
            │                                    │
            │  ┌──────────────────────┐         │
            │  │ systemd: postgresql  │         │
            │  │   ↳ PostgreSQL 16    │         │
            │  └──────────────────────┘         │
            └──────────────────────────────────┘
```

- **nginx** 监听 80/443 端口，做反代 + 静态文件托管 + HTTPS 证书
- **trend-graph** 后端是 systemd 管理的常驻进程，监听 127.0.0.1:8080
- **PostgreSQL** 是数据库（前面已装好）

## 二、前置条件清单

| 项 | 要求 |
|---|---|
| OS | Ubuntu 22.04 / 24.04 |
| 域名 | 一个，已解析到 VPS 公网 IP（可选，没域名只能用 IP） |
| 软件包 | nginx、build-essential、Go、Node、PostgreSQL |
| 端口开放 | 80、443（如果用域名+HTTPS） |

## 三、初次部署：完整流程

### 3.1 系统准备

```bash
# 装基础工具
sudo apt update
sudo apt install -y build-essential nginx

# 验证 Go / Node 已装（前面阶段已装）
go version        # >= 1.22
node --version    # >= 20
```

### 3.2 编译后端

```bash
cd /opt/trend-graph/backend

# 设置国内代理
go env -w GOPROXY=https://goproxy.cn,direct

# 装依赖
go mod tidy

# 编译生产二进制
# CGO_ENABLED=0 让二进制静态，不依赖 libc，能跨机器拷贝
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /opt/trend-graph/bin/trend-graph ./cmd/server

# 验证能跑
/opt/trend-graph/bin/trend-graph
# Ctrl+C 退出
```

### 3.3 准备配置

```bash
# 创建运行目录
sudo mkdir -p /opt/trend-graph
sudo chown -R $USER:$USER /opt/trend-graph

# 配置 .env
cp /opt/trend-graph/backend/.env.example /opt/trend-graph/.env
nano /opt/trend-graph/.env
# 必填：
#   PORT=8080
#   DATABASE_URL=host=127.0.0.1 port=5432 user=tguser password=tgpass dbname=trend_graph sslmode=disable timezone=Asia/Shanghai
#   DEEPSEEK_API_KEY=sk-你的key
# 可选：
#   FEISHU_WEBHOOK=...
#   DINGTALK_WEBHOOK=...
#   SMTP_*...
```

### 3.4 构建 frontend

```bash
cd /opt/trend-graph/frontend
npm install --registry=https://registry.npmmirror.com
npm run build
# 产物在 dist/
ls dist/

# 把 dist 拷到 nginx 服务目录
sudo mkdir -p /var/www/trend-graph
sudo cp -r dist/* /var/www/trend-graph/
sudo chown -R www-data:www-data /var/www/trend-graph
```

### 3.5 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/trend-graph.service << 'EOF'
[Unit]
Description=trend-graph backend (AI 热点监控 + 关联图谱)
Documentation=https://github.com/zhujufeng/trend-graph
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/trend-graph
EnvironmentFile=/opt/trend-graph/.env
ExecStart=/opt/trend-graph/bin/trend-graph
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=trend-graph

[Install]
WantedBy=multi-user.target
EOF

# 重载 systemd
sudo systemctl daemon-reload
# 开机自启
sudo systemctl enable trend-graph
# 启动
sudo systemctl start trend-graph

# 看状态
sudo systemctl status trend-graph
# 看日志（实时跟踪）
sudo journalctl -u trend-graph -f
```

### 3.6 配置 nginx

```bash
sudo tee /etc/nginx/sites-available/trend-graph << 'EOF'
# HTTP 端口（如果有 HTTPS 会被自动重定向）
server {
    listen 80;
    server_name _;  # 替换成你的域名/IP

    # 前端静态文件
    root /var/www/trend-graph;
    index index.html;

    # SPA 路由兜底
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反代到后端
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 反代（关键：Upgrade/Connection 头）
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
    }

    # gzip 压缩
    gzip on;
    gzip_types text/plain application/javascript text/css application/json;
    gzip_min_length 1024;

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|svg|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# 启用站点
sudo ln -sf /etc/nginx/sites-available/trend-graph /etc/nginx/sites-enabled/trend-graph

# 删默认站点（避免冲突）
sudo rm -f /etc/nginx/sites-enabled/default

# 测试配置
sudo nginx -t

# 重载 nginx
sudo systemctl reload nginx
```

### 3.7 验证部署

```bash
# 健康检查
curl http://localhost/health
# 应返回 {"status":"ok","service":"trend-graph",...}

# 抓取测试
curl -X POST "http://localhost/api/crawl?source=hn&limit=3"
# 应返回热点数据

# 列表测试
curl "http://localhost/api/hots?source=hn&since=1h"

# 浏览器访问 http://你的VPS_IP/
# 应看到 trend-graph 界面
```

## 四、HTTPS（用 Caddy 自动签证书）

如果你有域名（强烈建议），用 Caddy 比 nginx + certbot 简单太多。

### 4.1 装 Caddy

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
```

### 4.2 关掉 nginx（让 Caddy 接管 80/443）

```bash
sudo systemctl stop nginx
sudo systemctl disable nginx
```

### 4.3 配置 Caddy

```bash
sudo tee /etc/caddy/Caddyfile << 'EOF'
你的域名.com {
    # 前端静态
    root * /var/www/trend-graph
    encode gzip
    try_files {path} /index.html
    file_server

    # API 反代
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }

    # WebSocket 反代
    handle /ws {
        reverse_proxy 127.0.0.1:8080
    }
}
EOF

# 启动 Caddy（会自动签 Let's Encrypt 证书）
sudo systemctl restart caddy
sudo systemctl enable caddy
sudo systemctl status caddy

# 浏览器访问 https://你的域名.com
# 第一次访问可能要等 10~30 秒签证书
```

## 五、日常运维命令

### 5.1 后端服务管理

```bash
# 查看状态
sudo systemctl status trend-graph

# 重启
sudo systemctl restart trend-graph

# 停止
sudo systemctl stop trend-graph

# 实时日志（跟踪）
sudo journalctl -u trend-graph -f

# 最近 100 行日志
sudo journalctl -u trend-graph -n 100

# 某时段的日志
sudo journalctl -u trend-graph --since "1 hour ago"
```

### 5.2 数据库

```bash
# 进 PostgreSQL
sudo -u postgres psql
\c trend_graph

# 看热点总数
SELECT count(*) FROM hot_items;

# 看关键词
SELECT id, word, active, interval_min FROM keywords;

# 看实体
SELECT name, kind, count FROM entities ORDER BY count DESC LIMIT 20;

# 看关系
SELECT type_from, id_from, relation, type_to, id_to, weight FROM entity_relations ORDER BY weight DESC LIMIT 20;

# 清空热点（测试用，慎用）
TRUNCATE hot_items;
```

### 5.3 nginx/Caddy

```bash
# nginx 重载（改了配置不重启服务）
sudo nginx -t && sudo systemctl reload nginx

# Caddy 重载
sudo systemctl reload caddy
```

## 六、版本更新流程

代码有更新（git pull 之后）要重新部署：

```bash
#!/bin/bash
# 部署脚本：保存为 /opt/trend-graph/deploy.sh
set -e

cd /opt/trend-graph

echo "==> 拉最新代码"
git pull

echo "==> 编译后端"
cd backend
CGO_ENABLED=0 go build -o /opt/trend-graph/bin/trend-graph ./cmd/server

echo "==> 构建前端"
cd ../frontend
npm run build

echo "==> 部署前端到 nginx"
sudo rm -rf /var/www/trend-graph/*
sudo cp -r dist/* /var/www/trend-graph/
sudo chown -R www-data:www-data /var/www/trend-graph

echo "==> 重启后端"
sudo systemctl restart trend-graph

echo "==> 完成"
sudo systemctl status trend-graph | head -5
```

赋执行权限：`chmod +x /opt/trend-graph/deploy.sh`，以后改完代码 `./deploy.sh` 一键搞定。

## 七、常见问题

### Q1：访问 IP 显示 502 Bad Gateway

后端没起来。检查：
```bash
sudo systemctl status trend-graph
sudo journalctl -u trend-graph -n 50
```
最常见原因：`.env` 配置错（DB 连不上 / API key 拼错）。

### Q2：WebSocket 连不上（前端显示"离线"）

nginx/Caddy 的 `/ws` 反代要带 Upgrade/Connection 头，参考上面配置。

### Q3：Caddy 拿不到证书

- 域名 DNS 必须已解析到 VPS
- 80/443 端口必须在云控制台安全组放行
- Caddy 服务跑着才能续签：`sudo systemctl status caddy`

### Q4：磁盘/内存吃紧

```bash
# 看磁盘
df -h
# 看内存
free -h
# 看进程
top -bn1 | head -20
```
PostgreSQL 默认占用较大，可在 `/etc/postgresql/16/main/postgresql.conf` 调小 `shared_buffers`。

### Q5：防火墙开端口

```bash
# UFW 默认未启用，云厂商安全组要放行 80/443
sudo ufw status
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

腾讯云控制台 → 安全组 → 入站规则 → 添加 TCP 80 和 443。