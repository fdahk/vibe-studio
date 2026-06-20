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
      error_sms_send: '验证码发送失败',
      error_sms_code: '验证码错误或已失效',
      or: '或',
      github_login: '使用 GitHub 登录',
      tab_account: '账号登录',
      tab_phone: '手机号登录',
      phone_ph: '请输入手机号',
      code_ph: '请输入验证码',
      send_code: '获取验证码',
      resend_in: '{{n}}s 后重发',
      to_register: '没有账号？注册',
      to_login: '已有账号？登录',
      enter: '进入工作台',
      agree: '登录即代表同意用户协议与隐私政策',
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
      error_sms_send: 'Failed to send code',
      error_sms_code: 'Wrong or expired code',
      or: 'or',
      github_login: 'Continue with GitHub',
      tab_account: 'Account',
      tab_phone: 'Phone',
      phone_ph: 'Phone number',
      code_ph: 'Verification code',
      send_code: 'Get code',
      resend_in: 'Resend in {{n}}s',
      to_register: 'No account? Sign up',
      to_login: 'Have an account? Log in',
      enter: 'Enter workspace',
      agree: 'By continuing you agree to the Terms & Privacy Policy',
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
