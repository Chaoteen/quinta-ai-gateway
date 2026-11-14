import { APP_BASE_PATH, withBasePath } from '@/lib/base-path'
import aboutEn from '../../../../../docs/product/platform-introduction.en.md?raw'
import aboutZh from '../../../../../docs/product/platform-introduction.md?raw'
import guide1 from '../../../../../docs/user-guide/account_and_api_key.md?raw'
import apiEn from '../../../../../docs/user-guide/api_access.en.md?raw'
import apiZh from '../../../../../docs/user-guide/api_access.md?raw'
import guide6 from '../../../../../docs/user-guide/faq.md?raw'
import guide5 from '../../../../../docs/user-guide/invoice.md?raw'
import guide0 from '../../../../../docs/user-guide/quick_start.md?raw'
import guide2 from '../../../../../docs/user-guide/quota_and_billing.md?raw'
import guide3 from '../../../../../docs/user-guide/subscription.md?raw'
import guide4 from '../../../../../docs/user-guide/voucher.md?raw'

export function getPublicContent(
  kind: 'api' | 'about',
  language: string,
  origin: string
) {
  const chinese = language.toLowerCase().startsWith('zh')
  const content =
    kind === 'api' ? (chinese ? apiZh : apiEn) : chinese ? aboutZh : aboutEn
  return replacePublicLinks(content, origin)
}

function replacePublicLinks(content: string, origin: string) {
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

const guideMarkdown: Record<string, string> = {
  'quick-start': guide0,
  'account-and-api-key': guide1,
  'quota-and-billing': guide2,
  subscription: guide3,
  voucher: guide4,
  invoice: guide5,
  faq: guide6,
}

export function getPublicGuideContent(
  slug: string,
  language: string,
  origin: string
): string | undefined {
  if (slug === 'api-access') return getPublicContent('api', language, origin)
  const content = guideMarkdown[slug]
  return content ? replacePublicLinks(content, origin) : undefined
}
