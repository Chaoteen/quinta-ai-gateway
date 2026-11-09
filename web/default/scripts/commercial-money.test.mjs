import { expect, mock, test } from 'bun:test'

let displayType = 'CNY'
mock.module('@/i18n/config', () => ({ default: { language: 'zh', resolvedLanguage: 'zh' } }))
mock.module('@/stores/system-config-store', () => ({
  DEFAULT_CURRENCY_CONFIG: { customCurrencySymbol: '' },
  useSystemConfigStore: {
    getState: () => ({ config: { currency: { quotaDisplayType: displayType, customCurrencySymbol: '' } } }),
  },
}))
const { formatDisplayMoney } = await import('../src/lib/commercial-display.ts')

test('explicit order currency takes precedence over a different configured symbol', () => {
  displayType = 'CNY'
  expect(formatDisplayMoney(10, 'USD')).toContain('$')
  expect(formatDisplayMoney(10, 'USD')).not.toContain('¥')
  displayType = 'USD'
  expect(formatDisplayMoney(10, 'CNY')).toContain('¥')
  expect(formatDisplayMoney(10, 'CNY')).not.toContain('$')
})

test('display preference is still used for values without a declared currency', () => {
  displayType = 'CNY'
  expect(formatDisplayMoney(10)).toContain('¥')
})
