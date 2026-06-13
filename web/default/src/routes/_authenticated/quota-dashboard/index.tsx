import { createFileRoute } from '@tanstack/react-router'
import { QuotaDashboard } from '@/features/commercial-dashboards'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/quota-dashboard/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.FINANCE_CONSOLE),
  component: QuotaDashboard,
})
