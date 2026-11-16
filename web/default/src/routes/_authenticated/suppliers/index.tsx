import { createFileRoute } from '@tanstack/react-router'
import { requireRoot } from '@/lib/route-guards'
import { PlatformSuppliers } from '@/features/platform-suppliers'

export const Route = createFileRoute('/_authenticated/suppliers/')({
  beforeLoad: requireRoot,
  component: PlatformSuppliers,
})
