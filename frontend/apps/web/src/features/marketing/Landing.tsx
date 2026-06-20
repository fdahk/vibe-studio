import { type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { APP_NAME } from '@vibe/shared'
import { useAuthStore } from '../../store/auth'
import LangSwitch from '../../components/LangSwitch'
import AuthPanel from '../auth/AuthPanel'
import { PROFILE, SKILLS, WORKS, pick, type Lang } from './data'

// 个人作品集落地页：Hero（左资料 / 右登录面板）+ 关于 / 经历 / 技能 / 联系。
export default function Landing() {
  const { i18n } = useTranslation()
  const lang: Lang = i18n.resolvedLanguage === 'en' ? 'en' : 'zh'
  const accessToken = useAuthStore((s) => s.accessToken)

  return (
    <div className="min-h-screen bg-bg text-ink">
      <header className="mx-auto flex max-w-6xl items-center justify-between px-6 py-5">
        <span className="font-semibold">{APP_NAME}</span>
        <LangSwitch />
      </header>

      <section className="mx-auto grid max-w-6xl grid-cols-1 gap-12 px-6 pt-14 lg:grid-cols-[1.2fr_1fr] lg:pt-24">
        <div>
          <div className="mb-6 inline-flex items-center gap-2 text-sm text-ink-dim">
            <span className="h-1.5 w-1.5 rounded-full bg-accent" />
            {pick(lang, PROFILE.eyebrow)}
          </div>
          <h1 className="text-5xl font-bold tracking-tight lg:text-6xl">
            {pick(lang, PROFILE.name)}
            <span className="ml-3 align-middle text-base font-normal tracking-[0.25em] text-ink-faint">
              {PROFILE.pinyin}
            </span>
          </h1>
          <p className="mt-5 text-2xl text-ink-dim">{pick(lang, PROFILE.slogan)}</p>
          <p className="mt-4 max-w-md leading-relaxed text-ink-dim">{pick(lang, PROFILE.intro)}</p>
          <div className="mt-8 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-ink-faint">
            <span>{pick(lang, PROFILE.school)}</span>
            <span>·</span>
            <span>{pick(lang, PROFILE.intent)}</span>
            <span>·</span>
            <span>{pick(lang, PROFILE.age)}</span>
          </div>
        </div>

        <div className="flex items-start">{accessToken ? <Entered /> : <AuthPanel />}</div>
      </section>

      <Section index="01" label={lang === 'zh' ? '关于' : 'About'}>
        <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4">
          <Cell
            k={lang === 'zh' ? '身份' : 'Identity'}
            v={`${pick(lang, PROFILE.name)} · ${pick(lang, PROFILE.age)}`}
            hint={pick(lang, PROFILE.school)}
          />
          <Cell
            k={lang === 'zh' ? '方向' : 'Track'}
            v={pick(lang, PROFILE.intent)}
            hint={lang === 'zh' ? '校招目标岗' : 'Targeting'}
          />
          <Cell
            k={lang === 'zh' ? '奖项' : 'Award'}
            v={pick(lang, PROFILE.award)}
            hint={lang === 'zh' ? '国家级' : 'National'}
          />
          <Cell
            k={lang === 'zh' ? '在线' : 'Online'}
            v={`@${PROFILE.github}`}
            hint="GitHub / 掘金"
          />
        </div>
      </Section>

      <Section index="02" label={lang === 'zh' ? '经历' : 'Experience'}>
        <div className="space-y-8">
          {WORKS.map((w, i) => (
            <article
              key={i}
              className="grid grid-cols-1 gap-3 border-l border-border pl-6 md:grid-cols-[200px_1fr]"
            >
              <div className="text-sm text-ink-faint">
                <div>{w.period}</div>
                <div>{pick(lang, w.location)}</div>
              </div>
              <div>
                <h3 className="font-medium">
                  {pick(lang, w.role)} <span className="text-ink-faint">—</span>{' '}
                  {pick(lang, w.company)}
                </h3>
                <ul className="mt-2 space-y-1 text-sm text-ink-dim">
                  {w.highlights[lang].map((h, j) => (
                    <li key={j}>· {h}</li>
                  ))}
                </ul>
                <div className="mt-3 flex flex-wrap gap-2">
                  {w.stack.map((s) => (
                    <span
                      key={s}
                      className="rounded border border-border px-2 py-0.5 text-xs text-ink-faint"
                    >
                      {s}
                    </span>
                  ))}
                </div>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section index="03" label={lang === 'zh' ? '技能' : 'Skills'}>
        <div className="space-y-4">
          {SKILLS.map((g) => (
            <div
              key={g.title.zh}
              className="flex flex-col gap-2 border-b border-border pb-4 md:flex-row md:items-center"
            >
              <div className="w-36 shrink-0 text-sm text-ink-faint">{pick(lang, g.title)}</div>
              <div className="flex flex-wrap gap-2">
                {g.items.map((s) => (
                  <span
                    key={s}
                    className="rounded border border-border px-2.5 py-1 text-xs text-ink-dim"
                  >
                    {s}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Section>

      <Section index="04" label={lang === 'zh' ? '联系' : 'Contact'}>
        <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4">
          <Contact
            k={lang === 'zh' ? '邮箱' : 'Email'}
            v={PROFILE.email}
            href={`mailto:${PROFILE.email}`}
          />
          <Contact
            k={lang === 'zh' ? '电话' : 'Phone'}
            v={PROFILE.phone}
            href={`tel:${PROFILE.phone.replace(/\s/g, '')}`}
          />
          <Contact
            k="GitHub"
            v={`@${PROFILE.github}`}
            href={`https://github.com/${PROFILE.github}`}
          />
          <Contact
            k={lang === 'zh' ? '掘金' : 'Juejin'}
            v={`@${PROFILE.juejin}`}
            href={`https://juejin.cn/user/${PROFILE.juejin}`}
          />
        </div>
        <footer className="mt-10 border-t border-border pt-6 text-xs text-ink-faint">
          © 2026 {pick(lang, PROFILE.name)} · Built with {APP_NAME}
        </footer>
      </Section>
    </div>
  )
}

function Entered() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  return (
    <div className="w-full rounded-lg border border-border bg-surface p-7 text-center">
      <p className="text-ink-dim">{t('logged_in')}</p>
      <button
        onClick={() => navigate('/app')}
        className="mt-4 w-full rounded-md bg-accent py-3 font-medium text-white transition hover:bg-accent-hover"
      >
        {t('enter')}
      </button>
    </div>
  )
}

function Section({
  index,
  label,
  children,
}: {
  index: string
  label: string
  children: ReactNode
}) {
  return (
    <section className="mx-auto max-w-6xl px-6 py-16">
      <header className="mb-8 flex items-center gap-4">
        <span className="text-sm tabular-nums text-ink-faint">{index}</span>
        <h2 className="text-lg font-medium">{label}</h2>
        <span className="h-px flex-1 bg-border" />
      </header>
      {children}
    </section>
  )
}

function Cell({ k, v, hint }: { k: string; v: string; hint: string }) {
  return (
    <div className="bg-surface p-5">
      <div className="text-xs text-ink-faint">{k}</div>
      <div className="mt-1 text-sm">{v}</div>
      <div className="mt-1 text-xs text-ink-faint">{hint}</div>
    </div>
  )
}

function Contact({ k, v, href }: { k: string; v: string; href: string }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="bg-surface p-5 transition hover:bg-surface-2"
    >
      <div className="text-xs text-ink-faint">{k}</div>
      <div className="mt-1 truncate text-sm">{v}</div>
    </a>
  )
}
