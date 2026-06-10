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
  BillingPortalPage,
  BillingPortalParams,
  BillingRecord,
  BillingSubscription,
  BillingSummary,
  PaymentOrder,
  UsageRecord,
} from './types'

export async function getBillingSummary(): Promise<
  ApiResponse<BillingSummary>
> {
  const res = await api.get('/api/billing/summary')
  return res.data
}

export async function getBillingPayments(
  params?: BillingPortalParams
): Promise<ApiResponse<BillingPortalPage<PaymentOrder>>> {
  const res = await api.get('/api/billing/payments', { params })
  return res.data
}

export async function getBillingUsages(
  params?: BillingPortalParams
): Promise<ApiResponse<BillingPortalPage<UsageRecord>>> {
  const res = await api.get('/api/billing/usages', { params })
  return res.data
}

export async function getBillingRecords(
  params?: BillingPortalParams
): Promise<ApiResponse<BillingPortalPage<BillingRecord>>> {
  const res = await api.get('/api/billing/records', { params })
  return res.data
}

export async function getBillingSubscriptions(
  params?: BillingPortalParams
): Promise<ApiResponse<BillingPortalPage<BillingSubscription>>> {
  const res = await api.get('/api/billing/subscriptions', { params })
  return res.data
}
