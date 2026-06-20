import { type ButtonHTMLAttributes } from 'react'

type Variant = 'primary' | 'ghost'

// 自研按钮：走全局 design token。
export default function Button({
  variant = 'primary',
  className = '',
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  const styles =
    variant === 'primary'
      ? 'bg-accent text-white hover:bg-accent-hover active:brightness-95'
      : 'border border-border text-ink hover:border-border-strong hover:bg-surface-2'
  return (
    <button
      {...rest}
      className={`w-full rounded-md px-4 py-3 font-medium transition disabled:opacity-50 ${styles} ${className}`}
    />
  )
}
