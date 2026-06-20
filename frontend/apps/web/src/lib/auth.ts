import { api } from '@vibe/api-client'
import { useAuthStore } from '../store/auth'

const REFRESH_URL = '/api/v1/auth/refresh'

let refreshing: Promise<boolean> | null = null

// tryRefresh 单航班：并发时只真正发一次 /refresh。
export function tryRefresh(): Promise<boolean> {
  if (!refreshing) {
    refreshing = doRefresh().finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

async function doRefresh(): Promise<boolean> {
  try {
    const res = await fetch(REFRESH_URL, { method: 'POST', credentials: 'include' })
    if (!res.ok) {
      useAuthStore.getState().setAccessToken(null)
      return false
    }
    const body = (await res.json()) as { data?: { access_token?: string } }
    const token = body.data?.access_token ?? null
    useAuthStore.getState().setAccessToken(token)
    return token !== null
  } catch {
    useAuthStore.getState().setAccessToken(null)
    return false
  }
}

// bootstrapAuth 启动时静默续期：有有效 refresh cookie 则恢复登录态，最后置 ready。
export async function bootstrapAuth(): Promise<void> {
  await tryRefresh()
  useAuthStore.getState().setReady(true)
}

// 注册一次：自动附带内存里的 access；受保护接口 401 时自动 refresh 后重试一次。
api.use({
  onRequest({ request }) {
    const token = useAuthStore.getState().accessToken
    if (token) request.headers.set('Authorization', `Bearer ${token}`)
    return request
  },
  async onResponse({ request, response }) {
    // 仅对非鉴权接口的 401 做静默续期重试（登录失败的 401 不该触发 refresh）。
    if (response.status !== 401 || request.url.includes('/api/v1/auth/')) {
      return response
    }
    const ok = await tryRefresh()
    if (!ok) return response
    const retry = new Request(request)
    const token = useAuthStore.getState().accessToken
    if (token) retry.headers.set('Authorization', `Bearer ${token}`)
    return fetch(retry)
  },
})
