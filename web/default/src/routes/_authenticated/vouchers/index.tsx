import { createFileRoute } from '@tanstack/react-router'
import { VoucherPortal } from '@/features/vouchers'

export const Route = createFileRoute('/_authenticated/vouchers/')({
  component: VoucherPortal,
})
