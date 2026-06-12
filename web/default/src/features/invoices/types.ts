export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}

export type InvoicePage<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type InvoiceListParams = {
  p?: number
  page_size?: number
  status?: string
  user_id?: number
  source_id?: number
  keyword?: string
}

export type InvoiceProfile = {
  id: number
  tenant_id: number
  user_id: number
  profile_type: string
  title: string
  tax_no: string
  bank_name: string
  bank_account: string
  company_address: string
  company_phone: string
  recipient_name: string
  recipient_phone: string
  recipient_email: string
  recipient_address: string
  is_default: boolean
  status: string
  created_at: number
}

export type InvoiceApplication = {
  id: number
  application_no: string
  tenant_id: number
  user_id: number
  invoice_profile_id: number
  amount: number
  currency: string
  invoice_type: string
  status: string
  source_type: string
  source_id: number
  reviewer_id: number
  reviewed_at: number
  review_note: string
  invoice_no: string
  invoice_date: number
  issued_at: number
  created_at: number
}

export type InvoiceFile = {
  id: number
  invoice_application_id: number
  file_name: string
  file_url: string
  file_type: string
  uploaded_by: number
  created_at: number
}

export type PaymentOrder = {
  id: number
  order_no: string
  amount: number
  currency: string
  status: string
  subject?: string
  paid_at?: number
  created_at?: number
}
