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
import { api } from '@/lib/api'
import type {
  ApiResponse,
  PlanRecord,
  PlanPayload,
  UserSubscriptionRecord,
  SelfSubscriptionRecord,
  CreateUserSubscriptionRequest,
  SubscriptionPayResponse,
  SubscriptionPayRequest,
  SelfSubscriptionData,
  AdminListParams,
} from './types'

// ============================================================================
// Admin Plan Management
// ============================================================================

export async function getAdminPlans(
  params?: AdminListParams
): Promise<ApiResponse<PlanRecord[]>> {
  const res = await api.get('/api/subscription/admin/plans', { params })
  return res.data
}

export async function createPlan(
  data: PlanPayload
): Promise<ApiResponse<PlanRecord>> {
  const res = await api.post('/api/subscription/admin/plans', data)
  return res.data
}

export async function updatePlan(
  id: number,
  data: PlanPayload
): Promise<ApiResponse<PlanRecord>> {
  const res = await api.put(`/api/subscription/admin/plans/${id}`, data)
  return res.data
}

export async function patchPlanStatus(
  id: number,
  enabled: boolean
): Promise<ApiResponse> {
  const res = await api.patch(`/api/subscription/admin/plans/${id}`, {
    enabled,
    status: enabled ? 'enabled' : 'disabled',
  })
  return res.data
}

// ============================================================================
// Admin User Subscription Management
// ============================================================================

export async function getUserSubscriptions(
  userId: number
): Promise<ApiResponse<UserSubscriptionRecord[]>> {
  const res = await api.get(
    `/api/subscription/admin/users/${userId}/subscriptions`
  )
  return res.data
}

export async function getAdminUserSubscriptions(
  params?: AdminListParams
): Promise<ApiResponse<UserSubscriptionRecord[]>> {
  const res = await api.get('/api/subscription/admin/user-subscriptions', {
    params,
  })
  return res.data
}

export async function createUserSubscription(
  userId: number,
  data: CreateUserSubscriptionRequest
): Promise<ApiResponse<{ message?: string }>> {
  const res = await api.post(
    `/api/subscription/admin/users/${userId}/subscriptions`,
    data
  )
  return res.data
}

export async function invalidateUserSubscription(
  subId: number
): Promise<ApiResponse<{ message?: string }>> {
  const res = await api.patch(
    `/api/subscription/admin/user-subscriptions/${subId}/cancel`
  )
  return res.data
}

export async function suspendUserSubscription(
  subId: number
): Promise<ApiResponse<{ message?: string }>> {
  const res = await api.patch(
    `/api/subscription/admin/user-subscriptions/${subId}/suspend`
  )
  return res.data
}

export async function renewUserSubscription(
  subId: number
): Promise<ApiResponse<UserSubscriptionRecord>> {
  const res = await api.patch(
    `/api/subscription/admin/user-subscriptions/${subId}/renew`
  )
  return res.data
}

// ============================================================================
// User-facing Subscription Payment
// ============================================================================

export async function paySubscriptionStripe(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/stripe/pay', data)
  return res.data
}

export async function paySubscriptionCreem(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/creem/pay', data)
  return res.data
}

export async function paySubscriptionEpay(
  data: SubscriptionPayRequest & { payment_method: string }
): Promise<SubscriptionPayResponse & { url?: string }> {
  const res = await api.post('/api/subscription/epay/pay', data)
  return {
    ...res.data,
    url: res.data.url || (res as unknown as { url?: string }).url,
  }
}

// ============================================================================
// User Self Subscriptions
// ============================================================================

export async function getSelfSubscriptions(): Promise<
  ApiResponse<SelfSubscriptionRecord[]>
> {
  const res = await api.get('/api/subscription/self')
  return res.data
}

export async function getSelfSubscriptionFull(): Promise<
  ApiResponse<SelfSubscriptionData>
> {
  const res = await api.get('/api/subscription/self')
  return res.data
}

export async function getPublicPlans(): Promise<ApiResponse<PlanRecord[]>> {
  const res = await api.get('/api/subscription/plans')
  return res.data
}

export async function updateBillingPreference(
  preference: string
): Promise<ApiResponse<{ billing_preference?: string }>> {
  const res = await api.put('/api/subscription/self/preference', {
    billing_preference: preference,
  })
  return res.data
}

export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group')
  return res.data
}
