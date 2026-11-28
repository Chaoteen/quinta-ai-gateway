import { createFileRoute } from '@tanstack/react-router'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'
import { RolePermissions } from '@/features/role-permissions'

export const Route = createFileRoute('/_authenticated/role-permissions/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.ROLE_MANAGEMENT),
  component: RolePermissions,
})
