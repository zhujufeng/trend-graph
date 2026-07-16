# 修复内容生成与 GitHub 身份

## Goal

让“生成三平台内容包”在正常本地运行时真正调用已配置的 DeepSeek，并让本项目提交归属当前 GitHub 用户 `zhujufeng`。

## Background

- 已用真实登录请求稳定复现：`POST /api/radar/signals/12/content-packages` 返回 HTTP 500。
- 复现进程显式清空了 `DEEPSEEK_API_KEY`；`main` 将 `(*analyzer.Analyzer)(nil)` 传给接口类型参数，形成非 nil 的 typed-nil interface。`h.generator == nil` 未拦截，方法调用 panic，Gin 返回空 500。
- 用户点击时运行的正是该安全验收进程；即使 typed-nil 改为 503，内容仍无法生成，必须用 `.env` 中的 DeepSeek 配置启动可交互后端。
- 仓库没有 `user.name` / `user.email` 配置。最近三个提交使用自动身份 `zhujufeng@zhujufengdeMacBook-Air.local`，GitHub 无法关联贡献。
- 远端仓库属于公开账号 `zhujufeng`，GitHub 用户 ID 为 `99528815`，可使用 `99528815+zhujufeng@users.noreply.github.com`。

## Requirements

- 缺少内容模型时不得 panic；生成接口返回明确的 HTTP 503 JSON。
- 正常测试运行必须加载现有 `.env` DeepSeek 配置，使用户触发的内容生成可用。
- 不新增依赖。增加一个默认开启的 `BACKGROUND_JOBS_ENABLED` 开关，使本地能够只测试手动内容生成而不触发采集、批量分析和通知。
- 为本仓库配置 Git 作者名 `zhujufeng` 和 GitHub noreply 邮箱，保证未来提交可归属。
- 用户已明确批准重写已经推送的三个错误身份提交；执行时必须使用 force-with-lease，不能无保护强推。

## Diagnostic Hypotheses

1. 已证实：Go typed-nil interface 绕过 `h.generator == nil`，导致 500 panic；传入真正 nil 后应稳定返回 503。
2. 已证实：安全验收进程清空 DeepSeek key；恢复 `.env` 后 analyzer 应非 nil，生成路径应进入模型调用。
3. 次要可能：恢复模型后，DeepSeek 响应格式或模型名可能继续导致 502；需要真实生成请求验证，不能凭单元测试判断。
4. 已排除：前端吞掉后端错误。Axios 拦截器会显示后端 `error`，当前空 500 来自 panic 响应。

## Acceptance Criteria

- [x] 缺少 DeepSeek 配置时，真实或路由级回归检查得到 HTTP 503 和 `content model is not configured`，不再得到空 500。
- [x] 使用当前 `.env` 启动后端后，真实内容生成请求不再因模型未配置失败，并成功创建内容包；实测 HTTP 201，创建 `content_packages.id=1`。
- [x] `git config --local user.name` 和 `user.email` 对应 GitHub 账号 `zhujufeng`。
- [x] 最近三个本地提交已重写为 GitHub noreply 作者/提交者身份；待任务提交后使用 `--force-with-lease` 安全推送。
- [x] Go 测试、vet、前端相关测试和真实生成反馈环通过。

## Out of Scope

- 新的模型供应商、内容模板或自动发布功能。
- 重写三个提交之前的历史。
