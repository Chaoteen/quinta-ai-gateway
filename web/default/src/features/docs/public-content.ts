import { APP_BASE_PATH, withBasePath } from '@/lib/base-path'
import aboutEn from '../../../../../docs/product/platform-introduction.en.md?raw'
import aboutZh from '../../../../../docs/product/platform-introduction.md?raw'
import apiEn from '../../../../../docs/user-guide/api_access.en.md?raw'
import apiZh from '../../../../../docs/user-guide/api_access.md?raw'

export function getPublicContent(
  kind: 'api' | 'about',
  language: string,
  origin: string
) {
  const chinese = language.toLowerCase().startsWith('zh')
  const content =
    kind === 'api' ? (chinese ? apiZh : apiEn) : chinese ? aboutZh : aboutEn
  const replacements: Record<string, string> = {
    __GATEWAY_BASE_URL__: `${origin}${APP_BASE_PATH}/v1`,
    __REGISTER_URL__: withBasePath('/sign-up'),
    __KEYS_URL__: withBasePath('/keys'),
    __MODELS_URL__: withBasePath('/pricing'),
    __BILLING_URL__: withBasePath('/billing'),
    __LOGS_URL__: withBasePath('/usage-logs'),
    __INVOICE_URL__: withBasePath('/invoices'),
    __API_GUIDE_URL__: withBasePath('/docs/api-access'),
  }
  return content.replace(/__[A-Z_]+__/g, (key) => replacements[key] ?? key)
}
