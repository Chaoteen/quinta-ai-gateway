import { api } from '@/lib/api'

export type Provider = {
  id: string
  name: string
  docs: string
  partnership: string
  endpoint: string
  channel_type: number
  guidance: string
}
export type Supplier = {
  id: number
  provider: string
  name: string
  legal_entity: string
  contact: string
  mode: string
  status: string
  currency: string
  agreement_ref: string
  agreement_expires_at: number
  version: number
}
export type Account = {
  id: number
  supplier_id: number
  channel_id: number
  name: string
  region: string
  account_ref: string
  key_ref: string
  endpoint: string
  status: string
}
export type Price = {
  unit: string
  unit_price_micros: number
  id: number
  account_id: number
  model: string
  effective_at: number
  input_micros: number
  output_micros: number
  cache_read_micros: number
  cache_write_micros: number
}
export type Statement = {
  id: number
  account_id: number
  external_ref: string
  invoice_ref: string
  currency: string
  amount_minor: number
  paid_minor: number
  status: string
  start_at: number
  end_at: number
  platform_input: number
  platform_output: number
  platform_requests: number
  snapshot_at: number
  review_note: string
}
export type Payment = {
  id: number
  statement_id: number
  reference: string
  amount_minor: number
  created_at: number
}
export type Audit = {
  id: number
  actor_id: number
  action: string
  resource_id: number
  created_at: number
}
export type SupplierData = {
  revisions: (Supplier & {
    supplier_id: number
    actor_id: number
    created_at: number
  })[]
  providers: Provider[]
  suppliers: Supplier[]
  accounts: Account[]
  prices: Price[]
  statements: Statement[]
  payments: Payment[]
  audit: Audit[]
}
export async function getSuppliers(): Promise<SupplierData> {
  const response = await api.get('/api/platform/suppliers')
  if (!response.data.success) throw new Error(response.data.message)
  return response.data.data
}
export async function mutateSupplier(
  path: string,
  body: unknown,
  method: 'post' | 'patch' = 'post'
) {
  const response = await api[method]('/api/platform/suppliers' + path, body, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  if (!response.data.success) throw new Error(response.data.message)
  return response.data.data
}
