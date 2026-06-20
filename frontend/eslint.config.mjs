import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import globals from 'globals'

// Flat config（ESLint 9）。单根配置统一 lint 整个 monorepo。
export default tseslint.config(
  {
    ignores: ['**/dist/**', '**/node_modules/**', '**/*.config.{js,mjs,cjs,ts}', '**/schema.d.ts'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: { globals: { ...globals.browser } },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      // 变量/全局未定义由 TypeScript 负责，关掉 eslint 的同名规则避免误报。
      'no-undef': 'off',
    },
  },
)
