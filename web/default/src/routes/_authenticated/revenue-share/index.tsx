import { createFileRoute } from '@tanstack/react-router'
import { RevenueShareDashboard } from '@/features/commercial-dashboards'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/revenue-share/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.FINANCE_CONSOLE),
  component: RevenueShareDashboard,
})
