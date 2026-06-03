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
import type { AuthUser } from '@/stores/auth-store'
import { getUserRoleKey, ROLE_KEY, type RoleKey } from './roles'

export const RBAC_PERMISSION = {
  ADMIN_CONSOLE: 'adminConsole',
  ROOT_ADMIN: 'rootAdmin',
  CHANNELS: 'channels',
  MODELS: 'models',
  USERS: 'users',
  TOPUP: 'topup',
  REDEMPTION_CODES: 'redemptionCodes',
  SUBSCRIPTIONS: 'subscriptions',
  USAGE_LOGS: 'usageLogs',
  DASHBOARD_USERS: 'dashboardUsers',
  DASHBOARD_STATS: 'dashboardStats',
  SYSTEM_SETTINGS: 'systemSettings',
} as const

export type RbacPermission =
  (typeof RBAC_PERMISSION)[keyof typeof RBAC_PERMISSION]

type PermissionSubject = Pick<AuthUser, 'role_key'> | null | undefined

export const RBAC_PERMISSION_MATRIX: Record<
  RbacPermission,
  readonly RoleKey[]
> = {
  [RBAC_PERMISSION.ADMIN_CONSOLE]: [
    ROLE_KEY.TENANT_ADMIN,
    ROLE_KEY.ORGANIZATION_ADMIN,
    ROLE_KEY.FINANCE,
    ROLE_KEY.OPS,
    ROLE_KEY.AUDITOR,
  ],
  [RBAC_PERMISSION.ROOT_ADMIN]: [],
  [RBAC_PERMISSION.CHANNELS]: [ROLE_KEY.TENANT_ADMIN, ROLE_KEY.OPS],
  [RBAC_PERMISSION.MODELS]: [],
  [RBAC_PERMISSION.USERS]: [ROLE_KEY.TENANT_ADMIN, ROLE_KEY.ORGANIZATION_ADMIN],
  [RBAC_PERMISSION.TOPUP]: [ROLE_KEY.FINANCE],
  [RBAC_PERMISSION.REDEMPTION_CODES]: [ROLE_KEY.FINANCE],
  [RBAC_PERMISSION.SUBSCRIPTIONS]: [
    ROLE_KEY.TENANT_ADMIN,
    ROLE_KEY.ORGANIZATION_ADMIN,
    ROLE_KEY.FINANCE,
    ROLE_KEY.AUDITOR,
  ],
  [RBAC_PERMISSION.USAGE_LOGS]: [
    ROLE_KEY.TENANT_ADMIN,
    ROLE_KEY.ORGANIZATION_ADMIN,
    ROLE_KEY.OPS,
    ROLE_KEY.AUDITOR,
  ],
  [RBAC_PERMISSION.DASHBOARD_USERS]: [
    ROLE_KEY.TENANT_ADMIN,
    ROLE_KEY.ORGANIZATION_ADMIN,
    ROLE_KEY.FINANCE,
    ROLE_KEY.OPS,
    ROLE_KEY.AUDITOR,
  ],
  [RBAC_PERMISSION.DASHBOARD_STATS]: [ROLE_KEY.FINANCE, ROLE_KEY.AUDITOR],
  [RBAC_PERMISSION.SYSTEM_SETTINGS]: [],
}

export const ADMIN_SIDEBAR_PERMISSION_BY_URL: Record<string, RbacPermission> = {
  '/tenants': RBAC_PERMISSION.ROOT_ADMIN,
  '/organizations': RBAC_PERMISSION.ROOT_ADMIN,
  '/departments': RBAC_PERMISSION.ROOT_ADMIN,
  '/distribution-channels': RBAC_PERMISSION.ROOT_ADMIN,
  '/channels': RBAC_PERMISSION.CHANNELS,
  '/models/metadata': RBAC_PERMISSION.MODELS,
  '/users': RBAC_PERMISSION.USERS,
  '/topup': RBAC_PERMISSION.TOPUP,
  '/redemption-codes': RBAC_PERMISSION.REDEMPTION_CODES,
  '/subscriptions': RBAC_PERMISSION.SUBSCRIPTIONS,
  '/usage-logs/common': RBAC_PERMISSION.USAGE_LOGS,
  '/dashboard/models': RBAC_PERMISSION.DASHBOARD_STATS,
  '/system-settings/site': RBAC_PERMISSION.SYSTEM_SETTINGS,
}

export function hasPermission(
  user: PermissionSubject,
  permission: RbacPermission
): boolean {
  const roleKey = getUserRoleKey(user)
  if (roleKey === ROLE_KEY.ROOT) return true
  return RBAC_PERMISSION_MATRIX[permission].includes(roleKey)
}

export function isRootUser(user?: PermissionSubject): boolean {
  return hasPermission(user, RBAC_PERMISSION.ROOT_ADMIN)
}

export function isAdminConsoleUser(user?: PermissionSubject): boolean {
  return hasPermission(user, RBAC_PERMISSION.ADMIN_CONSOLE)
}
