import { useTranslation } from 'react-i18next'

export default function LangSwitch() {
  const { i18n } = useTranslation()
  const btn = (lng: string, label: string) => (
    <button
      onClick={() => void i18n.changeLanguage(lng)}
      disabled={i18n.resolvedLanguage === lng}
      className="text-sm text-ink-faint transition hover:text-ink-dim disabled:font-medium disabled:text-ink"
    >
      {label}
    </button>
  )
  return (
    <div className="flex items-center gap-3">
      {btn('zh-CN', '中文')}
      {btn('en', 'EN')}
    </div>
  )
}
