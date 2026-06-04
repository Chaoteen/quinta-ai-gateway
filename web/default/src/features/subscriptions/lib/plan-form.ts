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
import type { TFunction } from 'i18next'
import type { SubscriptionPlan, PlanPayload } from '../types'

export function getPlanFormSchema(t: TFunction) {
  return z.object({
    code: z.string().min(1, t('Please enter plan code')),
    name: z.string().min(1, t('Please enter plan name')),
    description: z.string().optional(),
    monthly_price: z.coerce.number().min(0, t('Please enter amount')),
    yearly_price: z.coerce.number().min(0, t('Please enter amount')),
    token_quota: z.coerce.number().min(0),
    request_quota: z.coerce.number().min(0),
    model_quota: z.string().optional(),
    status: z.enum(['enabled', 'disabled']),
  })
}

export type PlanFormValues = z.infer<ReturnType<typeof getPlanFormSchema>>

export const PLAN_FORM_DEFAULTS: PlanFormValues = {
  code: '',
  name: '',
  description: '',
  monthly_price: 0,
  yearly_price: 0,
  token_quota: 0,
  request_quota: 0,
  model_quota: '',
  status: 'enabled',
}

export function planToFormValues(plan: SubscriptionPlan): PlanFormValues {
  return {
    code: plan.code || '',
    name: plan.name || '',
    description: plan.description || '',
    monthly_price: Number(plan.monthly_price ?? 0),
    yearly_price: Number(plan.yearly_price || 0),
    token_quota: Number(plan.token_quota ?? 0),
    request_quota: Number(plan.request_quota || 0),
    model_quota: plan.model_quota || '',
    status: plan.status === 'disabled' ? 'disabled' : 'enabled',
  }
}

export function formValuesToPlanPayload(values: PlanFormValues): PlanPayload {
  return {
    plan: {
      code: values.code,
      name: values.name,
      description: values.description || '',
      monthly_price: Number(values.monthly_price || 0),
      yearly_price: Number(values.yearly_price || 0),
      token_quota: Number(values.token_quota || 0),
      request_quota: Number(values.request_quota || 0),
      model_quota: values.model_quota || '',
      status: values.status,
    },
  }
}
