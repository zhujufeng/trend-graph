# 阶段 7：定时任务 + 多渠道通知

> 对应 commit：`feat: stage 7 - cron 定时任务 + 多渠道通知`

## 🎯 目标

- 关键词管理 API（CRUD）
- cron 定时任务按间隔自动抓取
- 邮件 / 飞书 / 钉钉 三种通知渠道
- 节流避免刷屏

## 📚 学到的概念

### 1. robfig/cron 库 + `@every` 语法

```go
c := cron.New()
c.AddFunc("@every 30m", func() {
    // 每 30 分钟跑一次
})
c.Start()
```

`@every Nm` 比 5 段 cron 表达式（`*/30 * * * *`）更不容易写错，适合"间隔重于定点"的监控场景。

### 2. cron 任务动态增删

```go
// 每分钟重载 keywords 表
ticker := time.NewTicker(time.Minute)
for range ticker.C {
    s.Reload()  // 增量增删 cron 任务
}
```

用户在前端加/删关键词，1 分钟内 cron 自动同步。

### 3. net/smtp 发邮件

```go
auth := smtp.PlainAuth("", user, password, host)
msg := []byte("To: ...\r\nFrom: ...\r\nSubject: ...\r\n\r\n" + body)
smtp.SendMail(addr, auth, from, to, msg)
```

- 密码是**授权码**不是登录密码（QQ/163 都要单独开 SMTP 拿授权码）
- 邮件正文必须有 RFC822 头（To/From/Subject）

### 4. Webhook 推送（飞书）

```go
body := feishuPayload{MsgType: "text"}
body.Content.Text = text
jsonBody, _ := json.Marshal(body)
http.Post(webhook, "application/json", bytes.NewReader(jsonBody))
```

飞书自定义机器人最简单：POST 一个 JSON 到 webhook URL。

### 5. 钉钉签名算法

```go
ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
stringToSign := ts + "\n" + secret
mac := hmac.New(sha256.New, []byte(secret))
mac.Write([]byte(stringToSign))
sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
url += "&timestamp=" + ts + "&sign=" + urlEncode(sign)
```

钉钉机器人在管理后台勾"加签"安全设置后，每次请求必须带 timestamp + HMAC-SHA256 签名，否则被拒绝。

### 6. HMAC-SHA256

```go
mac := hmac.New(sha256.New, key)
mac.Write([]byte(message))
result := mac.Sum(nil)  // 32 字节哈希
```

消息认证码：用密钥 + 消息生成签名，对方用同密钥验证消息没被篡改。

### 7. 扇出模式 + 单渠道失败不影响其他

```go
type MultiChannelNotifier struct {
    channels []Notifier
}

func (m *MultiChannelNotifier) Notify(...) error {
    var errs []string
    for _, ch := range m.channels {
        if err := ch.Notify(ctx, payload); err != nil {
            errs = append(errs, err.Error())  // 收集错误不中断
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("部分通知渠道失败: %s", strings.Join(errs, "; "))
    }
    return nil
}
```

### 8. 节流策略（避免刷屏）

```go
shouldNotify := false
for i := range dbItems {
    if dbItems[i].Relevance != nil && *dbItems[i].Relevance >= 0.5 {
        shouldNotify = true
        break
    }
}
if shouldNotify {
    notifier.Notify(...)
}
```

只在"高相关命中"才推送，否则 30 分钟一次的抓取会刷屏。

### 9. PATCH 部分更新

```ts
// 前端只传变化的字段
const body: Record<string, unknown> = {}
if (active !== undefined) body.active = active
if (intervalMin !== undefined) body.intervalMin = intervalMin
await client.patch(`/keywords/${id}`, body)
```

PATCH 语义：只更新传了的字段，没传的不动。比 PUT（全量替换）省带宽。

### 10. axios 响应拦截器和返回值类型

```ts
client.interceptors.response.use(
  (response) => response.data,  // 拦截器拉出 .data
  ...
)
// 所以 client.get() 返回的其实是后端 body 本身（含 {data, meta}）
const res = await client.get<unknown, ApiResponse<T>>('/keywords')
return res.data  // 再解一层拿业务 payload
```

第二个泛型参数才是真实返回类型，第一个是 axios response 类型。

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| cron 调度器 | `backend/internal/scheduler/scheduler.go` |
| 三种通知渠道 | `backend/internal/notify/channels.go` |
| Keyword CRUD | `backend/internal/store/keyword_repo.go` |
| Keyword API | `backend/internal/api/handler.go` |
| 配置项 | `backend/internal/config/config.go` |
| 前端面板 | `frontend/src/components/KeywordPanel.tsx` |
| 前端 API | `frontend/src/api/keywords.ts` |

## 🧪 测试

```bash
go test -v -run TestScheduler_RunKeywordJob ./internal/scheduler/
```

## 🐛 踩坑

1. **钉钉签名 URL 编码**：base64 含 `+/=`，URL 里要分别转义 `%2B %2F %3D`
2. **`store.HotItem` vs `types.HotItem`**：scheduler 里混用导致编译错，记得 `store.HotItem` 才有 `Relevance` 字段
3. **关键词唯一索引冲突**：测试重复跑同一个 word，要先 `db.Unscoped().Where("word=?", ...).Delete()` 硬删
4. **`Assignment mismatch`**：函数返回多值就别用单变量接，要 `a, b := f()` 不要 `a := f()`

## 📝 一句话总结

cron + 多渠道通知 = 让工具"自己跑、自己通知"，从被动查看变主动推送，这是工具类项目最关键的体验跃迁。