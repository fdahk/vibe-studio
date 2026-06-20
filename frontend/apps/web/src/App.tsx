import { APP_NAME } from '@vibe/shared'
import LangSwitch from './components/LangSwitch'

export default function App() {
  return (
    <div className="mx-auto mt-12 max-w-md px-4">
      <LangSwitch />
      <h1 className="mb-4 text-2xl font-bold">{APP_NAME}</h1>
      <p className="text-gray-500">脚手架就绪，业务页面建设中…</p>
    </div>
  )
}
