import { api } from '@/lib/api'
import type {
  ApiResponse,
  Voucher,
  VoucherBatch,
  VoucherListParams,
  VoucherPage,
  VoucherRedemption,
} from './types'

export async function redeemVoucher(
  voucherCode: string
): Promise<ApiResponse<VoucherRedemption>> {
  const res = await api.post('/api/vouchers/redeem', {
    voucher_code: voucherCode,
  })
  return res.data
}

export async function getVoucherHistory(
  params?: VoucherListParams
): Promise<ApiResponse<VoucherPage<VoucherRedemption>>> {
  const res = await api.get('/api/vouchers/history', { params })
  return res.data
}

export async function createVoucherBatch(input: {
  name: string
  description?: string
  voucher_type: string
  status?: string
}): Promise<ApiResponse<VoucherBatch>> {
  const res = await api.post('/api/admin/vouchers/batches', input)
  return res.data
}

export async function generateVouchers(
  batchId: number,
  input: {
    quantity: number
    quota_amount?: number
    subscription_plan_id?: number
    expired_at?: number
  }
): Promise<ApiResponse<Voucher[]>> {
  const res = await api.post(`/api/admin/voucher-batches/${batchId}/generate`, input)
  return res.data
}

export async function getVoucherBatches(
  params?: VoucherListParams
): Promise<ApiResponse<VoucherPage<VoucherBatch>>> {
  const res = await api.get('/api/admin/vouchers/batches', { params })
  return res.data
}

export async function getVouchers(
  params?: VoucherListParams
): Promise<ApiResponse<VoucherPage<Voucher>>> {
  const res = await api.get('/api/admin/vouchers', { params })
  return res.data
}

export async function getVoucherRedemptions(
  params?: VoucherListParams
): Promise<ApiResponse<VoucherPage<VoucherRedemption>>> {
  const res = await api.get('/api/admin/vouchers/redemptions', { params })
  return res.data
}

export async function disableVoucher(
  voucherId: number
): Promise<ApiResponse<Voucher>> {
  const res = await api.post(`/api/admin/vouchers/${voucherId}/disable`)
  return res.data
}

export async function disableVoucherBatch(
  batchId: number
): Promise<ApiResponse<VoucherBatch>> {
  const res = await api.post(`/api/admin/voucher-batches/${batchId}/disable`)
  return res.data
}
