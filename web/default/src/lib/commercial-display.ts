import type { TFunction } from 'i18next'
import i18n from '@/i18n/config'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

export function getDisplayLocale(): string | undefined {
  return i18n.resolvedLanguage || i18n.language || undefined
}

export function getDefaultDisplayCurrency(locale = getDisplayLocale()): string {
  return locale?.toLowerCase().startsWith('zh') ? 'CNY' : 'USD'
}

function formatMoneyNumber(value: number): string {
  return new Intl.NumberFormat(getDisplayLocale(), {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function getConfiguredMoneySymbol(): string | null {
  const currency = useSystemConfigStore.getState().config.currency

  if (
    currency.customCurrencySymbol &&
    currency.customCurrencySymbol !== DEFAULT_CURRENCY_CONFIG.customCurrencySymbol
  ) {
    return currency.customCurrencySymbol
  }

  switch (currency.quotaDisplayType) {
    case 'CNY':
      return '¥'
    case 'CUSTOM':
      return currency.customCurrencySymbol || null
    case 'USD':
      return '$'
    case 'TOKENS':
    default:
      return null
  }
}

export function formatDisplayNumber(value?: number | null): string {
  if (value == null || Number.isNaN(Number(value))) return '-'
  return new Intl.NumberFormat(getDisplayLocale()).format(Number(value))
}

export function formatDisplayMoney(
  value?: number | null,
  currency?: string | null
): string {
  if (value == null || Number.isNaN(Number(value))) return '-'
  const configuredSymbol = getConfiguredMoneySymbol()
  if (configuredSymbol) {
    return `${configuredSymbol}${formatMoneyNumber(Number(value))}`
  }

  const resolvedCurrency =
    currency?.trim() || getDefaultDisplayCurrency(getDisplayLocale())

  try {
    return new Intl.NumberFormat(getDisplayLocale(), {
      style: 'currency',
      currency: resolvedCurrency,
      currencyDisplay: 'narrowSymbol',
      maximumFractionDigits: 2,
    }).format(Number(value))
  } catch {
    return `${formatDisplayNumber(value)} ${resolvedCurrency}`
  }
}

export function formatDisplayDateTime(value?: number | null): string {
  if (!value) return '-'
  return new Date(value * 1000).toLocaleString(getDisplayLocale())
}

const DISPLAY_LABELS: Record<string, string> = {
  active: 'Active',
  approved: 'Approved',
  cancelled: 'Canceled',
  canceled: 'Canceled',
  committed: 'Committed',
  company: 'Company',
  disabled: 'Disabled',
  draft: 'Draft',
  expired: 'Expired',
  failed: 'Failed',
  finished: 'Finished',
  ignored: 'Ignored',
  issued: 'Issued',
  paid: 'Paid',
  payment_order: 'Payment Order',
  pending: 'Pending',
  personal: 'Personal',
  processing: 'Processing',
  redeemed: 'Redeemed',
  rejected: 'Rejected',
  reserved: 'Reserved',
  settled: 'Settled',
  success: 'Success',
  subscription: 'Subscription voucher',
  token: 'Token voucher',
  unused: 'Unused',
  vat_normal: 'VAT Normal',
  vat_special: 'VAT Special',
}

export function displayEnumLabel(
  value: string | null | undefined,
  t: TFunction
): string {
  if (!value) return '-'
  const normalized = value.trim().toLowerCase()
  return t(DISPLAY_LABELS[normalized] ?? value)
}
