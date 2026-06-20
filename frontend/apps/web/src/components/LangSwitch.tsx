import { useTranslation } from 'react-i18next'

export default function LangSwitch() {
  const { i18n } = useTranslation()
  return (
    <div className="mb-3 flex justify-end gap-2 text-sm">
      <button
        onClick={() => void i18n.changeLanguage('zh-CN')}
        disabled={i18n.resolvedLanguage === 'zh-CN'}
        className="text-blue-600 disabled:text-gray-400"
      >
        中文
      </button>
      <button
        onClick={() => void i18n.changeLanguage('en')}
        disabled={i18n.resolvedLanguage === 'en'}
        className="text-blue-600 disabled:text-gray-400"
      >
        EN
      </button>
    </div>
  )
}
