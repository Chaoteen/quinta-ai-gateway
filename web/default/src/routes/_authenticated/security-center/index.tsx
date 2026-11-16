import { createFileRoute } from '@tanstack/react-router'
import { requireRoot } from '@/lib/route-guards'
import { SecurityCenter } from '@/features/security-center'

export const Route = createFileRoute('/_authenticated/security-center/')({
  beforeLoad: requireRoot,
  component: SecurityCenter,
})
