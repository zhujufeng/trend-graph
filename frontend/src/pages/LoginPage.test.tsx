import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { LoginPage } from './LoginPage'

describe('LoginPage', () => {
  it('renders the private radar password boundary', () => {
    const html = renderToStaticMarkup(<LoginPage loading={false} error="" onLogin={vi.fn()} />)

    expect(html).toContain('私人 AI 信号雷达')
    expect(html).toContain('管理员密码')
    expect(html).toContain('type="password"')
    expect(html).toContain('进入雷达')
  })
})
