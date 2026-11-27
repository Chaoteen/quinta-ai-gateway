import { createFileRoute } from '@tanstack/react-router'
import { requireRoot } from '@/lib/route-guards'
import { RolePermissions } from '@/features/role-permissions'
export const Route = createFileRoute('/_authenticated/role-permissions/')({ beforeLoad: requireRoot, component: RolePermissions })
