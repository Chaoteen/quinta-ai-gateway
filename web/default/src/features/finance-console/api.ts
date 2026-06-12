import { api } from '@/lib/api'
import type {
  ApiResponse,
  BillingRecord,
  FinanceListParams,
  FinanceMetricItem,
  FinancePage,
  FinanceSummary,
  FinanceTenantMetricItem,
  FinanceTopChannelItem,
  PaymentOrder,
  UserSubscription,
  VoucherRedemption,
} from './types'

export async function getFinanceSummary(
  params?: Pick<FinanceListParams, 'days'>
): Promise<ApiResponse<FinanceSummary>> {
  const res = await api.get('/api/admin/finance/summary', { params })
  return res.data
}

export async function getFinanceTopTenants(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<FinanceTenantMetricItem>>> {
  const res = await api.get('/api/admin/finance/top-tenants', { params })
  return res.data
}

export async function getFinanceTopModels(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<FinanceMetricItem>>> {
  const res = await api.get('/api/admin/finance/top-models', { params })
  return res.data
}

export async function getFinanceTopProviders(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<FinanceMetricItem>>> {
  const res = await api.get('/api/admin/finance/top-providers', { params })
  return res.data
}

export async function getFinanceTopChannels(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<FinanceTopChannelItem>>> {
  const res = await api.get('/api/admin/finance/top-channels', { params })
  return res.data
}

export async function getFinanceRecentPayments(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<PaymentOrder>>> {
  const res = await api.get('/api/admin/finance/recent-payments', { params })
  return res.data
}

export async function getFinanceRecentRedemptions(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<VoucherRedemption>>> {
  const res = await api.get('/api/admin/finance/recent-redemptions', { params })
  return res.data
}

export async function getFinanceRecentSubscriptions(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<UserSubscription>>> {
  const res = await api.get('/api/admin/finance/recent-subscriptions', { params })
  return res.data
}

export async function getFinanceRecentBilling(
  params?: FinanceListParams
): Promise<ApiResponse<FinancePage<BillingRecord>>> {
  const res = await api.get('/api/admin/finance/recent-billing', { params })
  return res.data
}
