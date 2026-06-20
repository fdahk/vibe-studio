import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type components } from '@vibe/api-client'
import { useAuthStore } from '../store/auth'

type User = components['schemas']['User']

// 当前登录用户：有 access token 才查；Bearer 由 api-client 中间件自动附带，
// 过期(401) 时中间件会静默 refresh 后重试。
export function useMe() {
  const accessToken = useAuthStore((s) => s.accessToken)
  return useQuery({
    queryKey: ['me', accessToken],
    enabled: !!accessToken,
    queryFn: async (): Promise<User> => {
      const { data, error } = await api.GET('/api/v1/users/me')
      if (error || !data?.data) throw new Error('未登录或登录已失效')
      return data.data.user
    },
  })
}

export function useLogin() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  return useMutation({
    mutationFn: async (vars: { username: string; password: string }): Promise<string> => {
      const { data, error } = await api.POST('/api/v1/auth/login', { body: vars })
      if (error || !data?.data) throw new Error('error_login')
      return data.data.access_token
    },
    onSuccess: setAccessToken,
  })
}

export function useRegister() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  return useMutation({
    mutationFn: async (vars: {
      username: string
      password: string
      email?: string
    }): Promise<string> => {
      const { data, error } = await api.POST('/api/v1/auth/register', { body: vars })
      if (error || !data?.data) throw new Error('error_register')
      return data.data.access_token
    },
    onSuccess: setAccessToken,
  })
}

export function useSendSmsCode() {
  return useMutation({
    mutationFn: async (phone: string): Promise<void> => {
      const { error } = await api.POST('/api/v1/auth/sms/code', { body: { phone } })
      if (error) throw new Error('error_sms_send')
    },
  })
}

export function useLoginByPhone() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  return useMutation({
    mutationFn: async (vars: { phone: string; code: string }): Promise<string> => {
      const { data, error } = await api.POST('/api/v1/auth/login/phone', { body: vars })
      if (error || !data?.data) throw new Error('error_sms_code')
      return data.data.access_token
    },
    onSuccess: setAccessToken,
  })
}

export function useLogout() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (): Promise<void> => {
      await api.POST('/api/v1/auth/logout')
    },
    onSuccess: () => {
      setAccessToken(null)
      qc.clear()
    },
  })
}
