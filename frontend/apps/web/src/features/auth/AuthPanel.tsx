import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useLogin, useLoginByPhone, useRegister, useSendSmsCode } from '../../api/auth'
import Field from './components/Field'
import Button from './components/Button'

type Tab = 'account' | 'phone'

export default function AuthPanel() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('account')
  const onDone = () => navigate('/app', { replace: true })

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-7">
      <div className="mb-6 flex gap-6 text-sm">
        {(['account', 'phone'] as Tab[]).map((k) => (
          <button
            key={k}
            onClick={() => setTab(k)}
            className={
              tab === k
                ? 'border-b-2 border-accent pb-2 font-medium text-ink'
                : 'pb-2 text-ink-faint hover:text-ink-dim'
            }
          >
            {t(k === 'account' ? 'tab_account' : 'tab_phone')}
          </button>
        ))}
      </div>

      {tab === 'account' ? <AccountForm onDone={onDone} /> : <PhoneForm onDone={onDone} />}

      <div className="my-5 flex items-center gap-3 text-xs text-ink-faint">
        <span className="h-px flex-1 bg-border" />
        {t('or')}
        <span className="h-px flex-1 bg-border" />
      </div>
      <a
        href="/api/v1/auth/oauth/github"
        className="flex w-full items-center justify-center rounded-md border border-border px-4 py-3 text-ink transition hover:border-border-strong hover:bg-surface-2"
      >
        {t('github_login')}
      </a>

      <p className="mt-5 text-center text-xs text-ink-faint">{t('agree')}</p>
    </div>
  )
}

function AccountForm({ onDone }: { onDone: () => void }) {
  const { t } = useTranslation()
  const login = useLogin()
  const register = useRegister()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const pending = login.isPending || register.isPending
  const errKey = (login.error ?? register.error)?.message

  function submit(e: FormEvent) {
    e.preventDefault()
    const vars = { username, password }
    if (mode === 'login') login.mutate(vars, { onSuccess: onDone })
    else register.mutate(vars, { onSuccess: onDone })
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <Field
        value={username}
        onChange={(e) => setUsername(e.target.value)}
        placeholder={t('username')}
      />
      <Field
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder={t('password')}
      />
      {errKey && <p className="text-sm text-danger">{t(errKey)}</p>}
      <Button type="submit" disabled={pending || !username || !password}>
        {pending ? t('processing') : t(mode === 'login' ? 'login_tab' : 'register_tab')}
      </Button>
      <button
        type="button"
        onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
        className="w-full text-center text-xs text-ink-faint hover:text-ink-dim"
      >
        {t(mode === 'login' ? 'to_register' : 'to_login')}
      </button>
    </form>
  )
}

function PhoneForm({ onDone }: { onDone: () => void }) {
  const { t } = useTranslation()
  const send = useSendSmsCode()
  const loginByPhone = useLoginByPhone()
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [cooldown, setCooldown] = useState(0)

  // 倒计时（带 cleanup）。
  useEffect(() => {
    if (cooldown <= 0) return
    const id = setTimeout(() => setCooldown((n) => n - 1), 1000)
    return () => clearTimeout(id)
  }, [cooldown])

  const errKey = (send.error ?? loginByPhone.error)?.message

  function sendCode() {
    send.mutate(phone, { onSuccess: () => setCooldown(60) })
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    loginByPhone.mutate({ phone, code }, { onSuccess: onDone })
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <Field
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
        placeholder={t('phone_ph')}
        inputMode="numeric"
      />
      <div className="flex gap-2">
        <Field
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder={t('code_ph')}
          inputMode="numeric"
        />
        <button
          type="button"
          onClick={sendCode}
          disabled={cooldown > 0 || send.isPending || phone.length < 11}
          className="shrink-0 whitespace-nowrap rounded-md border border-border px-3 text-sm text-accent transition hover:border-border-strong hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {cooldown > 0 ? t('resend_in', { n: cooldown }) : t('send_code')}
        </button>
      </div>
      {errKey && <p className="text-sm text-danger">{t(errKey)}</p>}
      <Button type="submit" disabled={loginByPhone.isPending || !phone || !code}>
        {loginByPhone.isPending ? t('processing') : t('login_tab')}
      </Button>
    </form>
  )
}
