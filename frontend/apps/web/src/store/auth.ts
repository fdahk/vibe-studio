import { create } from 'zustand'

interface AuthState {
  // access token 只放内存（不持久化）——防 XSS 持久窃取；刷新页靠 refresh cookie 静默恢复。
  accessToken: string | null
  // 启动时的静默 refresh 是否已完成（用于首屏 gate，避免闪登录页）。
  ready: boolean
  setAccessToken: (token: string | null) => void
  setReady: (ready: boolean) => void
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  ready: false,
  setAccessToken: (accessToken) => set({ accessToken }),
  setReady: (ready) => set({ ready }),
}))
