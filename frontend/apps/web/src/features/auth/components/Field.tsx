import { type InputHTMLAttributes } from 'react'

// 自研输入框：走全局 design token，无第三方组件库。
export default function Field(props: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className="w-full rounded-md border border-border bg-surface-2 px-4 py-3 text-ink outline-none transition placeholder:text-ink-faint hover:border-border-strong focus:border-accent"
    />
  )
}
