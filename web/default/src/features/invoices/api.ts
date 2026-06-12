import { api } from '@/lib/api'
import type {
  ApiResponse,
  InvoiceApplication,
  InvoiceFile,
  InvoiceListParams,
  InvoicePage,
  InvoiceProfile,
  PaymentOrder,
} from './types'

export async function createInvoiceProfile(input: Partial<InvoiceProfile>) {
  const res = await api.post<ApiResponse<InvoiceProfile>>('/api/invoices/profiles', input)
  return res.data
}

export async function getInvoiceProfiles(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceProfile>>>('/api/invoices/profiles', { params })
  return res.data
}

export async function disableInvoiceProfile(profileId: number) {
  const res = await api.post<ApiResponse<InvoiceProfile>>(`/api/invoices/profiles/${profileId}/disable`)
  return res.data
}

export async function createInvoiceApplication(input: {
  invoice_profile_id: number
  amount: number
  currency?: string
  invoice_type: string
  source_type: string
  source_id: number
}) {
  const res = await api.post<ApiResponse<InvoiceApplication>>('/api/invoices/applications', input)
  return res.data
}

export async function getInvoiceApplications(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceApplication>>>('/api/invoices/applications', { params })
  return res.data
}

export async function getInvoiceFiles(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceFile>>>('/api/invoices/files', { params })
  return res.data
}

export async function getPaidPaymentOrders(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<PaymentOrder>>>('/api/payment/orders', {
    params: { ...params, status: 'PAID' },
  })
  return res.data
}

export async function getAdminInvoiceProfiles(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceProfile>>>('/api/admin/invoices/profiles', { params })
  return res.data
}

export async function getAdminInvoiceApplications(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceApplication>>>('/api/admin/invoices/applications', { params })
  return res.data
}

export async function getAdminInvoiceFiles(params?: InvoiceListParams) {
  const res = await api.get<ApiResponse<InvoicePage<InvoiceFile>>>('/api/admin/invoices/files', { params })
  return res.data
}

export async function reviewInvoiceApplication(applicationId: number, input: { approved: boolean; review_note?: string }) {
  const res = await api.post<ApiResponse<InvoiceApplication>>(`/api/admin/invoices/applications/${applicationId}/review`, input)
  return res.data
}

export async function issueInvoice(applicationId: number, input: {
  invoice_no: string
  invoice_date?: number
  file_name: string
  file_url: string
  file_type: string
}) {
  const res = await api.post<ApiResponse<InvoiceApplication>>(`/api/admin/invoices/applications/${applicationId}/issue`, input)
  return res.data
}
