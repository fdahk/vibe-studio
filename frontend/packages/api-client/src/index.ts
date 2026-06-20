import createClient from 'openapi-fetch'
import type { paths } from './schema'

// 从 backend/api/openapi/openapi.yaml 生成的类型安全客户端。
// baseUrl 用相对根 '/'，由 vite dev proxy 把 /api/* 转发到后端(:8888)。
// credentials:'include'：携带 refresh HttpOnly cookie（鉴权传输需要）。
export const api = createClient<paths>({ baseUrl: '/', credentials: 'include' })

export type { paths, components } from './schema'
