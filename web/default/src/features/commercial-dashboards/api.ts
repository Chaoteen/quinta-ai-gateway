import { api } from '@/lib/api'
import type {
  ApiResponse,
  BillingPortalPage,
  BillingSummary,
  PaymentOrder,
  UsageRecord,
  BillingRecord,
} from '@/features/billing-portal/types'

export type BankTransferRecord = {
  id: number
  payment_order_id: number
  tenant_id: number
  user_id: number
  bank_account_name: string
  bank_account_no_masked: string
  transfer_amount: number
  transfer_time?: number
  proof_url?: string
  review_status: string
  reviewed_by?: number
  reviewed_at?: number
  review_note?: string
  created_at?: number
}

export type CommercialPage<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export async function getCommercialBillingSummary(): Promise<
  ApiResponse<BillingSummary>
> {
  const res = await api.get('/api/billing/summary')
  return res.data
}

export async function getCommercialUsageRecords(params?: {
  p?: number
  page_size?: number
}): Promise<ApiResponse<BillingPortalPage<UsageRecord>>> {
  const res = await api.get('/api/billing/usages', { params })
  return res.data
}

export async function getCommercialBillingRecords(params?: {
  p?: number
  page_size?: number
}): Promise<ApiResponse<BillingPortalPage<BillingRecord>>> {
  const res = await api.get('/api/billing/records', { params })
  return res.data
}

export async function getAdminPaymentOrders(params?: {
  p?: number
  page_size?: number
  status?: string
  provider?: string
}): Promise<ApiResponse<CommercialPage<PaymentOrder>>> {
  const res = await api.get('/api/admin/payment/orders', { params })
  return res.data
}

export async function getAdminBankTransfers(params?: {
  p?: number
  page_size?: number
  status?: string
}): Promise<ApiResponse<CommercialPage<BankTransferRecord>>> {
  const res = await api.get('/api/admin/payment/bank-transfers', { params })
  return res.data
}
