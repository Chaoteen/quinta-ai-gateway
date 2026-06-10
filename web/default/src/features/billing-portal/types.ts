/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface BillingPortalPage<T> {
  page: number
  page_size: number
  total: number
  items: T[]
}

export interface BillingRankingItem {
  name: string
  quota_charged: number
  total_tokens: number
  request_count: number
}

export interface BillingSubscription {
  id: number
  tenant_id?: number
  organization_id?: number
  user_id: number
  plan_id: number
  status: string
  lifecycle_status: string
  source?: string
  start_time: number
  end_time: number
  token_quota_snapshot?: number
  request_quota_snapshot?: number
  created_at?: number
  updated_at?: number
}

export interface BillingSummary {
  balance_quota: number
  current_subscriptions: BillingSubscription[]
  total_recharge_amount: number
  total_recharge_currency: string
  total_consumption_amount: number
  consumption_currency: string
  total_tokens: number
  total_requests: number
  recent_30d_consumption: number
  recent_30d_tokens: number
  recent_30d_requests: number
  model_consumption_ranking: BillingRankingItem[]
  provider_consumption_ranking: BillingRankingItem[]
}

export interface PaymentOrder {
  id: number
  order_no: string
  user_id: number
  provider: string
  business_type: string
  business_id: number
  amount: number
  currency: string
  status: string
  subject?: string
  paid_at?: number
  created_at?: number
}

export interface UsageRecord {
  id: number
  user_id: number
  provider_name: string
  model_name: string
  request_id: string
  request_count: number
  total_tokens: number
  token_delta: number
  request_delta: number
  status: string
  occurred_at?: number
}

export interface BillingRecord {
  id: number
  user_id: number
  provider_name: string
  model_name: string
  request_id: string
  usage_record_id: number
  billing_status: string
  currency: string
  total_tokens: number
  request_count: number
  quota_charged: number
  created_at?: number
}

export interface BillingPortalParams {
  p?: number
  page_size?: number
  status?: string
  provider?: string
  model?: string
  payment_method?: string
  subscription?: string
}
