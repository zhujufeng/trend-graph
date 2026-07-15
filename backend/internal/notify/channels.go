// notify_channels.go 实现邮件 / 飞书 / 钉钉三个具体通知渠道。
//
// 三个渠道都实现 Notifier 接口（就是 notify.Notifier），
// 但因为它们是"群发"逻辑，需要一个汇总器按关键词发送，
// 所以再封装一层 MultiChannelNotifier 来扇出。
package notify

// 导入：
// - fmt/bytes/crypto/hmac/sha256: 钉钉签名计算
// - encoding/base64: 钉钉签名 Base64
// - encoding/json: 序列化 payload
// - net/http/net/smtp: 发 HTTP 和发邮件
// - strconv/time/strings: 杂项
import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// ===== 邮件通知 =====

// EmailNotifier 用 SMTP 发邮件
type EmailNotifier struct {
	host     string // SMTP 服务器
	port     int    // 端口
	user     string // 用户名（一般是邮箱地址）
	password string // 授权码（不是邮箱登录密码）
	from     string // 发件人
	to       string // 收件人（多个用逗号分隔）
}

// NewEmailNotifier 构造
// 用户名、密码、收件人都由调用方从配置传入
func NewEmailNotifier(host string, port int, user, password, from, to string) *EmailNotifier {
	return &EmailNotifier{host: host, port: port, user: user, password: password, from: from, to: to}
}

// Notify 发一封邮件。
// payload 必须是 string（邮件正文）或 fmt.Stringer，这里简化只接收 string
//
// 阶段 7 用弱类型 any，使用方传字符串即可。
// 真实项目会用类型枚举或专门 struct。
func (e *EmailNotifier) Notify(ctx context.Context, payload any) error {
	body, ok := payload.(string)
	if !ok {
		return fmt.Errorf("EmailNotifier: payload 必须是 string")
	}
	if e.user == "" || e.to == "" {
		return fmt.Errorf("EmailNotifier: 未配置 user/to")
	}

	// SMTP auth
	auth := smtp.PlainAuth("", e.user, e.password, e.host)
	addr := e.host + ":" + strconv.Itoa(e.port)
	// 邮件正文需要 header，简拼一段
	msg := []byte("To: " + e.to + "\r\n" +
		"From: " + e.from + "\r\n" +
		"Subject: trend-graph 监控通知\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	return smtp.SendMail(addr, auth, e.from, strings.Split(e.to, ","), msg)
}

// ===== 飞书通知 =====

// FeishuNotifier 飞书自定义机器人 Webhook
type FeishuNotifier struct {
	webhook string
	client  *http.Client
}

// NewFeishuNotifier 构造
func NewFeishuNotifier(webhook string) *FeishuNotifier {
	return &FeishuNotifier{webhook: webhook, client: &http.Client{Timeout: 10 * time.Second}}
}

// feishuPayload 飞书机器人消息体（文本消息最简版本）
type feishuPayload struct {
	MsgType string `json:"msg_type"` // "text"
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

type FeishuPost struct {
	Title    string
	Sections []FeishuSection
}

type FeishuSection struct {
	Text     string
	LinkText string
	LinkURL  string
}

type feishuPostPayload struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Post map[string]feishuPostLanguage `json:"post"`
	} `json:"content"`
}

type feishuPostLanguage struct {
	Title   string             `json:"title"`
	Content [][]feishuPostItem `json:"content"`
}

type feishuPostItem struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// Notify 发送飞书消息
func (f *FeishuNotifier) Notify(ctx context.Context, payload any) error {
	if f.webhook == "" {
		return fmt.Errorf("FeishuNotifier: webhook 未配置")
	}
	var body any
	switch value := payload.(type) {
	case string:
		textBody := feishuPayload{MsgType: "text"}
		textBody.Content.Text = value
		body = textBody
	case FeishuPost:
		language := feishuPostLanguage{Title: value.Title, Content: make([][]feishuPostItem, 0, len(value.Sections))}
		for _, section := range value.Sections {
			row := []feishuPostItem{{Tag: "text", Text: section.Text}}
			if section.LinkURL != "" {
				row = append(row, feishuPostItem{Tag: "a", Text: section.LinkText, Href: section.LinkURL})
			}
			language.Content = append(language.Content, row)
		}
		postBody := feishuPostPayload{MsgType: "post"}
		postBody.Content.Post = map[string]feishuPostLanguage{"zh_cn": language}
		body = postBody
	default:
		return fmt.Errorf("FeishuNotifier: payload 必须是 string 或 FeishuPost")
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("编码飞书消息失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", f.webhook, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("构造飞书请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := f.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("飞书 webhook 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

// ===== 钉钉通知 =====

// DingTalkNotifier 钉钉自定义机器人 Webhook
//
// 钉钉机器人需要签名（secret + timestamp + hmac-sha256）防伪造。
type DingTalkNotifier struct {
	webhook string // 机器人 Webhook URL（含 access_token）
	secret  string // 加签密钥（机器人在钉钉后台可见）
}

// NewDingTalkNotifier 构造
func NewDingTalkNotifier(webhook, secret string) *DingTalkNotifier {
	return &DingTalkNotifier{webhook: webhook, secret: secret}
}

// dingTalkPayload 钉钉文本消息体
type dingTalkPayload struct {
	MsgType string `json:"msgtype"` // "text"
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// Notify 发送钉钉消息
//
// 钉钉签名算法：
//
//	stringToSign = timestamp + "\n" + secret
//	sign = base64(hmac_sha256(stringToSign, secret))
//	url += &timestamp=xxx&sign=xxx
func (d *DingTalkNotifier) Notify(ctx context.Context, payload any) error {
	text, ok := payload.(string)
	if !ok {
		return fmt.Errorf("DingTalkNotifier: payload 必须是 string")
	}
	if d.webhook == "" {
		return fmt.Errorf("DingTalkNotifier: webhook 未配置")
	}

	// 拼最终 URL（含签名）
	url := d.webhook
	if d.secret != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		stringToSign := ts + "\n" + d.secret
		mac := hmac.New(sha256.New, []byte(d.secret))
		mac.Write([]byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		// URL 编码：base64 含 +/= 特殊字符
		url += "&timestamp=" + ts + "&sign=" + urlEncodeStr(sign)
	}

	body := dingTalkPayload{MsgType: "text"}
	body.Text.Content = text

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("钉钉 webhook 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

// urlEncodeStr 简单 URL 编码（钉钉签名里 base64 含 +/= 要转义）
func urlEncodeStr(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '+':
			b.WriteString("%2B")
		case c == '/':
			b.WriteString("%2F")
		case c == '=':
			b.WriteString("%3D")
		case c == '\n':
			b.WriteString("%0A")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ===== 多渠道扇出 =====

// MultiChannelNotifier 把多个渠道扇出：原生 Notifier 接口的"广播器"
type MultiChannelNotifier struct {
	channels []Notifier
}

// NewMultiChannelNotifier 拼装多个渠道
// 没有传任何 channel 时调用 Notify 直接返回 nil（不报错）
func NewMultiChannelNotifier(channels ...Notifier) *MultiChannelNotifier {
	return &MultiChannelNotifier{channels: channels}
}

// Notify 实现 Notifier 接口，发给所有 channel
// 单个 channel 失败不影响其他，错误聚合返回
func (m *MultiChannelNotifier) Notify(ctx context.Context, payload any) error {
	if len(m.channels) == 0 {
		return nil
	}
	var errs []string
	for _, ch := range m.channels {
		if err := ch.Notify(ctx, payload); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分通知渠道失败: %s", strings.Join(errs, "; "))
	}
	return nil
}
