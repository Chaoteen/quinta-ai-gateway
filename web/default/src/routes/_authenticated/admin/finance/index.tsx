import { createFileRoute } from '@tanstack/react-router'
import { FinanceConsole } from '@/features/finance-console'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/admin/finance/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.FINANCE_CONSOLE),
  component: FinanceConsole,
})
