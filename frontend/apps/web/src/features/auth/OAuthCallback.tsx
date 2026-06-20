import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../store/auth'

// 第三方登录回调：后端已种好 refresh cookie，App 启动的静默 refresh 换出 access；
// 这里只等 ready，再按是否登录跳转。
export default function OAuthCallback() {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const ready = useAuthStore((s) => s.ready)
  const accessToken = useAuthStore((s) => s.accessToken)

  useEffect(() => {
    if (ready) navigate(accessToken ? '/app' : '/', { replace: true })
  }, [ready, accessToken, navigate])

  return <p className="mx-auto mt-12 max-w-md px-6 text-ink-faint">{t('loading')}</p>
}
