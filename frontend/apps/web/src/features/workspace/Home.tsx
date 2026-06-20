import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { APP_NAME } from '@vibe/shared'
import { useAuthStore } from '../../store/auth'
import { useLogout, useMe } from '../../api/auth'
import LangSwitch from '../../components/LangSwitch'

// 登录后的工作台（占位；画布等业务后续）。
export default function Home() {
  const { t } = useTranslation()
  const accessToken = useAuthStore((s) => s.accessToken)
  const me = useMe()
  const logout = useLogout()

  // 未登录回落地页（refresh 失效时 access 也会被清，自动触发跳转）。
  if (!accessToken) return <Navigate to="/" replace />

  return (
    <div className="min-h-screen bg-bg text-ink">
      <header className="mx-auto flex max-w-5xl items-center justify-between px-6 py-5">
        <span className="font-semibold">{APP_NAME}</span>
        <LangSwitch />
      </header>
      <main className="mx-auto max-w-5xl px-6 py-16">
        {me.isPending ? (
          <p className="text-ink-faint">{t('loading')}</p>
        ) : me.data ? (
          <div className="rounded-lg border border-border bg-surface p-8">
            <p>
              {t('logged_in')}：<b>{me.data.username}</b>（{me.data.email || t('no_email')}）
            </p>
            <p className="mt-1 text-sm text-ink-faint">id: {me.data.id}</p>
            <button
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
              className="mt-6 rounded-md border border-border px-4 py-2 transition hover:border-border-strong hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t('logout')}
            </button>
          </div>
        ) : (
          <p className="text-danger">{t('session_expired')}</p>
        )}
      </main>
    </div>
  )
}
