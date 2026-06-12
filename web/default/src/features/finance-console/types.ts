export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}

export type FinancePage<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type FinanceListParams = {
  p?: number
  page_size?: number
  days?: number
}

export type FinanceRevenueSummary = {
  total_recharge_amount: number
  month_recharge_amount: number
  recent_30d_recharge: number
  payment_order_count: number
  paid_payment_order_count: number
  payment_success_rate: number
  currency: string
}

export type FinanceConsumptionSummary = {
  total_consumption_amount: number
  month_consumption_amount: number
  recent_30d_consumption: number
  total_requests: number
  total_tokens: number
  currency: string
}

export type FinanceActivitySummary = {
  active_tenant_count: number
  active_user_count: number
  active_subscription_count: number
  active_channel_count: number
}

export type FinanceProviderPayment = {
  provider: string
  amount: number
  orders: number
}

export type FinanceDailyAmount = {
  date: string
  amount: number
  orders: number
}

export type FinancePaymentDashboard = {
  days: number
  total_amount: number
  total_orders: number
  paid_orders: number
  success_rate: number
  provider_breakdown: FinanceProviderPayment[]
  daily_trend: FinanceDailyAmount[]
}

export type FinanceVoucherDashboard = {
  total_issued: number
  total_redeemed: number
  total_unused: number
  redemption_rate: number
  batch_count: number
  active_batch_count: number
}

export type FinanceTopChannelItem = {
  distribution_channel_id: number
  name: string
  gross_amount: number
  platform_amount: number
  record_count: number
}

export type FinanceRevenueShareSummary = {
  gross_amount: number
  platform_amount: number
  master_distributor_amount: number
  distributor_amount: number
  currency: string
  top_channels: FinanceTopChannelItem[]
}

export type FinanceTenantMetricItem = {
  tenant_id: number
  name: string
  amount: number
  count: number
}

export type FinanceMetricItem = {
  name: string
  amount: number
  request_count: number
  total_tokens: number
}

export type FinanceTenantDashboard = {
  recharge_ranking: FinanceTenantMetricItem[]
  consumption_ranking: FinanceTenantMetricItem[]
  balance_ranking: FinanceTenantMetricItem[]
  subscription_ranking: FinanceTenantMetricItem[]
}

export type FinanceSummary = {
  revenue: FinanceRevenueSummary
  consumption: FinanceConsumptionSummary
  activity: FinanceActivitySummary
  payment: FinancePaymentDashboard
  voucher: FinanceVoucherDashboard
  revenue_share: FinanceRevenueShareSummary
  tenant: FinanceTenantDashboard
}

export type PaymentOrder = {
  id: number
  order_no: string
  tenant_id: number
  user_id: number
  provider: string
  business_type: string
  amount: number
  currency: string
  status: string
  subject: string
  created_at: number
  paid_at: number
}

export type VoucherRedemption = {
  id: number
  voucher_code: string
  user_id: number
  tenant_id: number
  redemption_type: string
  redemption_result: string
  created_at: number
}

export type UserSubscription = {
  id: number
  tenant_id: number
  user_id: number
  plan_id: number
  status: string
  amount_total: number
  start_time: number
  end_time: number
  created_at: number
}

export type BillingRecord = {
  id: number
  tenant_id: number
  user_id: number
  provider_name: string
  model_name: string
  quota_charged: number
  request_count: number
  total_tokens: number
  billing_status: string
  created_at: number
}
