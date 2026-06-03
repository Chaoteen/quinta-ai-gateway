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
  AdminConsoleApiResponse,
  AdminConsoleMutationPayload,
  ReadonlyListResponse,
  ReadonlyResource,
  ResourceRecordMap,
} from './types'

type ReadonlyResourceParams = {
  page?: number
  limit?: number
  q?: string
  tenant_id?: number
}

export async function getReadonlyResource<T extends ReadonlyResource>(
  resource: T,
  params: ReadonlyResourceParams = {}
): Promise<ReadonlyListResponse<ResourceRecordMap[T]>> {
  const res = await api.get(`/api/admin_console/${resource}`, {
    params: {
      page: params.page ?? 1,
      limit: params.limit ?? 50,
      q: params.q || undefined,
      tenant_id: params.tenant_id || undefined,
    },
  })
  return res.data.data
}

export async function createAdminConsoleResource<T extends ReadonlyResource>(
  resource: T,
  payload: AdminConsoleMutationPayload
): Promise<AdminConsoleApiResponse<ResourceRecordMap[T]>> {
  const res = await api.post(`/api/admin_console/${resource}`, payload)
  return res.data
}

export async function updateAdminConsoleResource<T extends ReadonlyResource>(
  resource: T,
  id: number,
  payload: AdminConsoleMutationPayload
): Promise<AdminConsoleApiResponse<ResourceRecordMap[T]>> {
  const res = await api.put(`/api/admin_console/${resource}/${id}`, payload)
  return res.data
}

export async function updateAdminConsoleResourceStatus<
  T extends ReadonlyResource,
>(
  resource: T,
  id: number,
  status: number
): Promise<AdminConsoleApiResponse<ResourceRecordMap[T]>> {
  const res = await api.patch(`/api/admin_console/${resource}/${id}/status`, {
    status,
  })
  return res.data
}
