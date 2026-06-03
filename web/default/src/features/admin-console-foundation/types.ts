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

export type ReadonlyResource =
  | 'tenants'
  | 'organizations'
  | 'departments'
  | 'distribution_channels'

export interface TenantRecord {
  id: number
  name: string
  status: number
  created_at: number
}

export interface OrganizationRecord {
  id: number
  name: string
  tenant_id: number
  status: number
  created_at: number
}

export interface DepartmentRecord {
  id: number
  name: string
  tenant_id: number
  organization_id: number
  status: number
  created_at: number
}

export interface DistributionChannelRecord {
  id: number
  name: string
  code: string
  tenant_id: number
  status: number
  created_at: number
}

export type ReadonlyRecord =
  | TenantRecord
  | OrganizationRecord
  | DepartmentRecord
  | DistributionChannelRecord

export interface ReadonlyListResponse<T extends ReadonlyRecord> {
  items: T[]
  total: number
  page: number
  limit: number
}

export type ResourceRecordMap = {
  tenants: TenantRecord
  organizations: OrganizationRecord
  departments: DepartmentRecord
  distribution_channels: DistributionChannelRecord
}

export type AdminConsoleMutationPayload = {
  name: string
  status?: number
  tenant_id?: number
  organization_id?: number
  code?: string
}

export interface AdminConsoleApiResponse<T = unknown> {
  success: boolean
  message: string
  data: T
}
