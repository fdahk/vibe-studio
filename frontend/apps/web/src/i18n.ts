import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

// i18n（对齐官方：i18next）。资源内联在前端，默认中文，可切英文。
// ICU 复杂消息格式化（复数/性别）暂未引入，待出现此类文案时再加 i18next-icu。
const resources = {
  'zh-CN': {
    translation: {
      login_tab: '登录',
      register_tab: '注册',
      username: '用户名',
      password: '密码',
      processing: '处理中…',
      logged_in: '已登录',
      no_email: '无邮箱',
      logout: '退出登录',
      loading: '加载中…',
      session_expired: '登录状态已失效，请重新登录',
      error_login: '用户名或密码错误',
      error_register: '注册失败（用户名可能已存在）',
    },
  },
  en: {
    translation: {
      login_tab: 'Log in',
      register_tab: 'Sign up',
      username: 'Username',
      password: 'Password',
      processing: 'Processing…',
      logged_in: 'Logged in',
      no_email: 'no email',
      logout: 'Log out',
      loading: 'Loading…',
      session_expired: 'Session expired, please log in again',
      error_login: 'Incorrect username or password',
      error_register: 'Registration failed (the username may already exist)',
    },
  },
}

void i18n.use(initReactI18next).init({
  resources,
  lng: 'zh-CN',
  fallbackLng: 'zh-CN',
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
})

export default i18n
