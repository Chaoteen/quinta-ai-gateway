export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}

export type VoucherPage<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type VoucherBatch = {
  id: number
  batch_no: string
  name: string
  description?: string
  voucher_type: string
  quantity: number
  status: string
  tenant_id: number
  organization_id: number
  department_id: number
  distribution_channel_id: number
  created_by: number
  created_at: number
  updated_at: number
}

export type Voucher = {
  id: number
  batch_id: number
  voucher_code: string
  voucher_type: string
  quota_amount: number
  subscription_plan_id: number
  status: string
  activated_by: number
  activated_at: number
  expired_at: number
  created_at: number
  updated_at: number
}

export type VoucherRedemption = {
  id: number
  voucher_id: number
  voucher_code: string
  user_id: number
  tenant_id: number
  organization_id: number
  department_id: number
  distribution_channel_id: number
  redemption_type: string
  redemption_result: string
  created_at: number
}

export type VoucherListParams = {
  p?: number
  page_size?: number
  keyword?: string
  status?: string
  voucher_type?: string
  batch_id?: number
  start_time?: number
  end_time?: number
  user_id?: number
}
