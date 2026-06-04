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
import { z } from 'zod'

// ============================================================================
// Subscription Plan Schema & Types
// ============================================================================

export const subscriptionPlanSchema = z.object({
  id: z.number(),
  code: z.string().optional(),
  name: z.string().optional(),
  description: z.string().optional(),
  monthly_price: z.number().optional(),
  yearly_price: z.number().optional(),
  token_quota: z.number().optional(),
  request_quota: z.number().optional(),
  model_quota: z.string().optional(),
  status: z.string().optional(),
  title: z.string().optional(),
  subtitle: z.string().optional(),
  price_amount: z.number().optional(),
  currency: z.string().optional().default('USD'),
  duration_unit: z.enum(['year', 'month', 'day', 'hour', 'custom']).optional(),
  duration_value: z.number().optional(),
  custom_seconds: z.number().optional(),
  quota_reset_period: z
    .enum(['never', 'daily', 'weekly', 'monthly', 'custom'])
    .optional(),
  quota_reset_custom_seconds: z.number().optional(),
  enabled: z.boolean().optional(),
  max_purchase_per_user: z.number().optional(),
  total_amount: z.number().optional(),
  upgrade_group: z.string().optional(),
  stripe_price_id: z.string().optional(),
  creem_product_id: z.string().optional(),
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
})

export type SubscriptionPlan = z.infer<typeof subscriptionPlanSchema>

export interface PlanRecord {
  plan: SubscriptionPlan
}

// ============================================================================
// User Subscription Schema & Types
// ============================================================================

export const userSubscriptionSchema = z.object({
  id: z.number(),
  tenant_id: z.number().optional(),
  organization_id: z.number().optional(),
  department_id: z.number().optional(),
  distribution_channel_id: z.number().optional(),
  user_id: z.number(),
  plan_id: z.number(),
  plan_code: z.string().optional(),
  plan_name: z.string().optional(),
  lifecycle_status: z.string(),
  start_time: z.number(),
  end_time: z.number(),
  token_quota_snapshot: z.number().optional(),
  request_quota_snapshot: z.number().optional(),
  model_quota_snapshot: z.string().optional(),
  next_reset_time: z.number().optional(),
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
})

export type UserSubscription = z.infer<typeof userSubscriptionSchema>

export interface UserSubscriptionRecord {
  subscription: UserSubscription
}

export const selfSubscriptionSchema = z.object({
  plan_code: z.string().optional(),
  plan_name: z.string().optional(),
  lifecycle_status: z.string(),
  start_time: z.number(),
  end_time: z.number(),
  token_quota_snapshot: z.number().optional(),
  request_quota_snapshot: z.number().optional(),
  model_quota_snapshot: z.string().optional(),
  next_reset_time: z.number().optional(),
  token_quota: z.number().optional(),
  token_used: z.number().optional(),
  token_remaining: z.number().optional(),
})

export type SelfSubscription = z.infer<typeof selfSubscriptionSchema>

export interface SelfSubscriptionRecord {
  subscription: SelfSubscription
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface PlanPayload {
  plan: Partial<SubscriptionPlan>
}

export interface SubscriptionPayRequest {
  plan_id: number
  payment_method?: string
}

export interface SubscriptionPayResponse {
  success: boolean
  message?: string
  data?: {
    pay_link?: string
    checkout_url?: string
  }
  url?: string
}

export interface CreateUserSubscriptionRequest {
  plan_id: number
}

export interface AdminListParams {
  page?: number
  limit?: number
  status?: string
  q?: string
  user_id?: number
}

// ============================================================================
// Self Subscription Data (user-facing)
// ============================================================================

export interface SelfSubscriptionData {
  billing_preference: string
  subscriptions: SelfSubscriptionRecord[]
  all_subscriptions: SelfSubscriptionRecord[]
}

// ============================================================================
// Dialog Types
// ============================================================================

export type SubscriptionsDialogType = 'create' | 'update' | 'toggle-status'
