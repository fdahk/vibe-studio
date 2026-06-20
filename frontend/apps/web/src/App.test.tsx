import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'
import App from './App'
import { useAuthStore } from './store/auth'

// 屏蔽启动静默 refresh，避免无服务端时的网络调用。
vi.mock('./lib/auth', () => ({ bootstrapAuth: () => undefined }))

function renderApp(initialEntry: string) {
  useAuthStore.setState({ ready: true, accessToken: null })
  const qc = new QueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('落地页渲染个人信息与登录面板', () => {
  renderApp('/')
  expect(screen.getByRole('heading', { name: /涂将/ })).toBeInTheDocument()
  expect(screen.getByPlaceholderText('用户名')).toBeInTheDocument()
  expect(screen.getByText('使用 GitHub 登录')).toBeInTheDocument()
})

test('登录面板可切换到手机号 tab', () => {
  renderApp('/')
  fireEvent.click(screen.getByText('手机号登录'))
  expect(screen.getByPlaceholderText('请输入手机号')).toBeInTheDocument()
})
