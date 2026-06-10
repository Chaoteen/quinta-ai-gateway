import { createFileRoute } from '@tanstack/react-router'
import { VoucherAdminPortal } from '@/features/vouchers'
import { RBAC_PERMISSION } from '@/lib/rbac'
import { requirePermission } from '@/lib/route-guards'

export const Route = createFileRoute('/_authenticated/admin/vouchers/')({
  beforeLoad: () => requirePermission(RBAC_PERMISSION.VOUCHERS),
  component: VoucherAdminPortal,
})
