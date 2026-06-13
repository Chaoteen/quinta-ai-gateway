import { createFileRoute } from '@tanstack/react-router'
import { UsageAnalyticsDashboard } from '@/features/commercial-dashboards'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/usage-analytics/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.USAGE_LOGS),
  component: UsageAnalyticsDashboard,
})
