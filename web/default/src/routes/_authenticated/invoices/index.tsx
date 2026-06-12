import { createFileRoute } from '@tanstack/react-router'
import { InvoicePortal } from '@/features/invoices'

export const Route = createFileRoute('/_authenticated/invoices/')({
  component: InvoicePortal,
})
