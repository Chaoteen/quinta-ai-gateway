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
import { t } from 'i18next'
import type { AuthUser } from '@/stores/auth-store'

export const ROLE = {
  GUEST: 0, // 后续如果需要用到这个角色那就再加，同语先留一下
  USER: 1,
  ADMIN: 10,
  SUPER_ADMIN: 100,
} as const

export type RoleValue = (typeof ROLE)[keyof typeof ROLE]

export const ROLE_KEY = {
  ROOT: 'root',
  TENANT_ADMIN: 'tenant_admin',
  ORGANIZATION_ADMIN: 'organization_admin',
  FINANCE: 'finance',
  OPS: 'ops',
  AUDITOR: 'auditor',
  USER: 'user',
} as const

export type RoleKey = (typeof ROLE_KEY)[keyof typeof ROLE_KEY]

const DEFAULT_ROLE = ROLE.GUEST

const ROLE_LABEL_KEYS: Record<RoleValue, string> = {
  [ROLE.SUPER_ADMIN]: 'Super Admin',
  [ROLE.ADMIN]: 'Admin',
  [ROLE.USER]: 'User',
  [ROLE.GUEST]: 'Guest',
}

export function getRoleLabelKey(role?: number): string {
  return ROLE_LABEL_KEYS[role as RoleValue] ?? ROLE_LABEL_KEYS[DEFAULT_ROLE]
}

export function getRoleLabel(role?: number): string {
  return t(getRoleLabelKey(role))
}

export function normalizeRoleKey(roleKey?: string | null): RoleKey {
  switch ((roleKey || '').trim().toLowerCase()) {
    case ROLE_KEY.ROOT:
      return ROLE_KEY.ROOT
    case ROLE_KEY.TENANT_ADMIN:
      return ROLE_KEY.TENANT_ADMIN
    case ROLE_KEY.ORGANIZATION_ADMIN:
      return ROLE_KEY.ORGANIZATION_ADMIN
    case ROLE_KEY.FINANCE:
      return ROLE_KEY.FINANCE
    case ROLE_KEY.OPS:
      return ROLE_KEY.OPS
    case ROLE_KEY.AUDITOR:
      return ROLE_KEY.AUDITOR
    default:
      return ROLE_KEY.USER
  }
}

export function getUserRoleKey(
  user?: Pick<AuthUser, 'role_key'> | null
): RoleKey {
  if (!user) return ROLE_KEY.USER
  return normalizeRoleKey(user.role_key)
}
