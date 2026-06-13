import { createFileRoute } from '@tanstack/react-router'
import { PaymentCenter } from '@/features/commercial-dashboards'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/payment-center/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.FINANCE_CONSOLE),
  component: PaymentCenter,
})
