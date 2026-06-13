import { createFileRoute } from '@tanstack/react-router'
import { BillingDashboard } from '@/features/commercial-dashboards'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/billing-dashboard/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.FINANCE_CONSOLE),
  component: BillingDashboard,
})
