import { useState, type FormEvent } from 'react'
import { LockKeyhole, Sparkles } from 'lucide-react'

interface LoginPageProps {
  loading: boolean
  error: string
  onLogin: (password: string) => Promise<void> | void
}

export function LoginPage({ loading, error, onLogin }: LoginPageProps) {
  const [password, setPassword] = useState('')

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (password) await onLogin(password)
  }

  return (
    <main className="grid min-h-full place-items-center bg-base-bg px-5">
      <form className="w-full max-w-sm rounded-2xl border border-border bg-surface p-7 glow-border" onSubmit={submit}>
        <div className="flex items-center gap-2 text-accent">
          <Sparkles className="h-6 w-6" aria-hidden="true" />
          <span className="text-sm">trend-graph</span>
        </div>
        <h1 className="mt-5 text-2xl font-semibold">私人 AI 信号雷达</h1>
        <p className="mt-2 text-sm leading-6 text-text-secondary">聚合可实践的 AI 更新、Skill、Agent 和真实用户痛点。</p>
        <label className="mt-6 block text-sm" htmlFor="admin-password">
          管理员密码
        </label>
        <div className="mt-2 flex items-center rounded-lg border border-border bg-base-bg px-3 focus-within:border-accent">
          <LockKeyhole className="h-4 w-4 text-text-muted" aria-hidden="true" />
          <input
            id="admin-password"
            className="min-w-0 flex-1 bg-transparent px-3 py-2.5 outline-none"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
        </div>
        {error && <p className="mt-3 text-sm text-red-300">{error}</p>}
        <button className="mt-5 w-full rounded-lg bg-accent px-4 py-2.5 font-medium text-base-bg hover:bg-accent-hover disabled:opacity-50" type="submit" disabled={loading}>
          {loading ? '登录中…' : '进入雷达'}
        </button>
      </form>
    </main>
  )
}
