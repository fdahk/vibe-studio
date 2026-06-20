// 跨包共享的类型/工具（web 及未来 JS 子应用复用）。
// 原生 iOS/Android 不复用这里的代码——它们经 OpenAPI 契约各自生成客户端。

export const APP_NAME = 'Vibe Studio'

/** 统一响应信封（与后端 pkg/response 的 {code,msg,data} 对应）。 */
export interface ApiEnvelope<T> {
  code: number
  msg: string
  data?: T
}
