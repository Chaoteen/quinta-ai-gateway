import { createFileRoute } from '@tanstack/react-router'
import { InvoiceAdminPortal } from '@/features/invoices'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/admin/invoices/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.INVOICES),
  component: InvoiceAdminPortal,
})
