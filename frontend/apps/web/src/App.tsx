import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from './store/auth'
import { bootstrapAuth } from './lib/auth'
import Landing from './features/marketing/Landing'
import Home from './features/workspace/Home'
import OAuthCallback from './features/auth/OAuthCallback'

export default function App() {
  const { t } = useTranslation()
  const ready = useAuthStore((s) => s.ready)

  // 首屏静默续期：拿 refresh cookie 换 access，完成前 gate 住路由，避免闪。
  useEffect(() => {
    void bootstrapAuth()
  }, [])

  if (!ready) {
    return <p className="bg-bg p-6 text-ink-faint">{t('loading')}</p>
  }

  return (
    <Routes>
      <Route path="/" element={<Landing />} />
      <Route path="/app" element={<Home />} />
      <Route path="/oauth/callback" element={<OAuthCallback />} />
    </Routes>
  )
}
