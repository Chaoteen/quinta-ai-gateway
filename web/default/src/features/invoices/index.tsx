import { useState, type ReactNode } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FileText, ReceiptText, RefreshCw, Send, Upload } from 'lucide-react'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  displayEnumLabel,
  formatDisplayDateTime,
  formatDisplayMoney,
} from '@/lib/commercial-display'
import {
  createInvoiceApplication,
  createInvoiceProfile,
  disableInvoiceProfile,
  getAdminInvoiceApplications,
  getAdminInvoiceFiles,
  getAdminInvoiceProfiles,
  getInvoiceApplications,
  getInvoiceFiles,
  getInvoiceProfiles,
  getPaidPaymentOrders,
  issueInvoice,
  reviewInvoiceApplication,
} from './api'
import type {
  InvoiceApplication,
  InvoiceFile,
  InvoicePage,
  InvoiceProfile,
} from './types'

const PAGE_SIZE = 10

type UserTab = 'profiles' | 'applications' | 'files'
type AdminTab = 'applications' | 'profiles' | 'files'

function statusVariant(
  status?: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status || '').toLowerCase()
  if (['active', 'approved', 'issued', 'paid'].includes(normalized))
    return 'default'
  if (['pending'].includes(normalized)) return 'secondary'
  if (['rejected', 'disabled', 'canceled', 'cancelled'].includes(normalized))
    return 'destructive'
  return 'outline'
}

function StatusBadge({ status }: { status?: string }) {
  const { t } = useTranslation()
  return (
    <Badge variant={statusVariant(status)}>
      {displayEnumLabel(status, t)}
    </Badge>
  )
}

function EmptyRows({ colSpan }: { colSpan: number }) {
  const { t } = useTranslation()
  return (
    <TableRow>
      <TableCell colSpan={colSpan} className='h-24 text-center text-muted-foreground'>
        {t('No data')}
      </TableCell>
    </TableRow>
  )
}

function Pager({
  page,
  pageData,
  onPageChange,
}: {
  page: number
  pageData?: InvoicePage<unknown>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const maxPage = Math.max(1, Math.ceil((pageData?.total ?? 0) / PAGE_SIZE))
  return (
    <div className='flex items-center justify-end gap-2 pt-3 text-sm text-muted-foreground'>
      <span>
        {t('Page')} {page} / {maxPage}
      </span>
      <Button
        type='button'
        variant='outline'
        size='sm'
        disabled={page <= 1}
        onClick={() => onPageChange(Math.max(1, page - 1))}
      >
        {t('Previous')}
      </Button>
      <Button
        type='button'
        variant='outline'
        size='sm'
        disabled={page >= maxPage}
        onClick={() => onPageChange(page + 1)}
      >
        {t('Next')}
      </Button>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className='grid gap-1 text-sm'>
      <span className='text-muted-foreground'>{label}</span>
      {children}
    </label>
  )
}

export function InvoicePortal() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<UserTab>('profiles')
  const [pages, setPages] = useState<Record<UserTab, number>>({
    profiles: 1,
    applications: 1,
    files: 1,
  })
  const [profileForm, setProfileForm] = useState({
    profile_type: 'COMPANY',
    title: '',
    tax_no: '',
    recipient_email: '',
    is_default: true,
  })
  const [applicationForm, setApplicationForm] = useState({
    invoice_profile_id: '',
    source_id: '',
    amount: '',
    invoice_type: 'VAT_NORMAL',
  })

  const profilesQuery = useQuery({
    queryKey: ['invoices', 'profiles', pages.profiles],
    queryFn: () => getInvoiceProfiles({ p: pages.profiles, page_size: PAGE_SIZE }),
  })
  const applicationsQuery = useQuery({
    queryKey: ['invoices', 'applications', pages.applications],
    queryFn: () =>
      getInvoiceApplications({ p: pages.applications, page_size: PAGE_SIZE }),
  })
  const filesQuery = useQuery({
    queryKey: ['invoices', 'files', pages.files],
    queryFn: () => getInvoiceFiles({ p: pages.files, page_size: PAGE_SIZE }),
  })
  const paymentsQuery = useQuery({
    queryKey: ['invoices', 'paid-payments'],
    queryFn: () => getPaidPaymentOrders({ p: 1, page_size: 100 }),
  })

  const refreshAll = () => {
    void profilesQuery.refetch()
    void applicationsQuery.refetch()
    void filesQuery.refetch()
    void paymentsQuery.refetch()
  }

  const createProfileMutation = useMutation({
    mutationFn: () => createInvoiceProfile(profileForm),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setProfileForm({
        profile_type: 'COMPANY',
        title: '',
        tax_no: '',
        recipient_email: '',
        is_default: true,
      })
      toast.success(t('Created successfully'))
      void profilesQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const createApplicationMutation = useMutation({
    mutationFn: () =>
      createInvoiceApplication({
        invoice_profile_id: Number(applicationForm.invoice_profile_id),
        source_id: Number(applicationForm.source_id),
        amount: Number(applicationForm.amount),
        invoice_type: applicationForm.invoice_type,
        source_type: 'PAYMENT_ORDER',
      }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setApplicationForm({
        invoice_profile_id: '',
        source_id: '',
        amount: '',
        invoice_type: 'VAT_NORMAL',
      })
      toast.success(t('Submitted successfully'))
      void applicationsQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const disableMutation = useMutation({
    mutationFn: disableInvoiceProfile,
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      toast.success(t('Disabled successfully'))
      void profilesQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const setPage = (key: UserTab, page: number) =>
    setPages((current) => ({ ...current, [key]: page }))

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Invoice')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button type='button' variant='outline' size='sm' onClick={refreshAll}>
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as UserTab)}
          className='space-y-4'
        >
          <TabsList className='w-full justify-start overflow-x-auto'>
            <TabsTrigger value='profiles'>
              <ReceiptText className='size-4' />
              {t('Invoice Profiles')}
            </TabsTrigger>
            <TabsTrigger value='applications'>
              <Send className='size-4' />
              {t('Invoice Applications')}
            </TabsTrigger>
            <TabsTrigger value='files'>
              <FileText className='size-4' />
              {t('Invoice Files')}
            </TabsTrigger>
          </TabsList>

          <TabsContent value='profiles'>
            <div className='grid gap-4 xl:grid-cols-[360px_1fr]'>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Create invoice profile')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <form
                    className='space-y-3'
                    onSubmit={(event) => {
                      event.preventDefault()
                      createProfileMutation.mutate()
                    }}
                  >
                    <Field label={t('Profile Type')}>
                      <select
                        className='h-9 rounded-md border bg-background px-3 text-sm'
                        value={profileForm.profile_type}
                        onChange={(event) =>
                          setProfileForm((current) => ({
                            ...current,
                            profile_type: event.target.value,
                          }))
                        }
                      >
                        <option value='COMPANY'>{t('Company')}</option>
                        <option value='PERSONAL'>{t('Personal')}</option>
                      </select>
                    </Field>
                    <Input
                      value={profileForm.title}
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          title: event.target.value,
                        }))
                      }
                      placeholder={t('Invoice title')}
                    />
                    <Input
                      value={profileForm.tax_no}
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          tax_no: event.target.value,
                        }))
                      }
                      placeholder={t('Tax number')}
                    />
                    <Input
                      value={profileForm.recipient_email}
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          recipient_email: event.target.value,
                        }))
                      }
                      placeholder={t('Recipient email')}
                    />
                    <label className='flex items-center gap-2 text-sm'>
                      <input
                        type='checkbox'
                        checked={profileForm.is_default}
                        onChange={(event) =>
                          setProfileForm((current) => ({
                            ...current,
                            is_default: event.target.checked,
                          }))
                        }
                      />
                      {t('Set as default')}
                    </label>
                    <Button type='submit' className='w-full'>
                      {t('Create')}
                    </Button>
                  </form>
                </CardContent>
              </Card>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Invoice Profiles')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <ProfileTable
                    items={profilesQuery.data?.data.items ?? []}
                    onDisable={(id) => disableMutation.mutate(id)}
                  />
                  <Pager
                    page={pages.profiles}
                    pageData={profilesQuery.data?.data}
                    onPageChange={(page) => setPage('profiles', page)}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value='applications'>
            <div className='grid gap-4 xl:grid-cols-[360px_1fr]'>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Submit invoice application')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <form
                    className='space-y-3'
                    onSubmit={(event) => {
                      event.preventDefault()
                      createApplicationMutation.mutate()
                    }}
                  >
                    <Field label={t('Payment Order')}>
                      <select
                        className='h-9 rounded-md border bg-background px-3 text-sm'
                        value={applicationForm.source_id}
                        onChange={(event) =>
                          setApplicationForm((current) => ({
                            ...current,
                            source_id: event.target.value,
                          }))
                        }
                      >
                        <option value=''>{t('Select payment order')}</option>
                        {(paymentsQuery.data?.data.items ?? []).map((order) => (
                          <option key={order.id} value={order.id}>
                            {order.order_no} - {formatDisplayMoney(order.amount, order.currency)}
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Field label={t('Invoice Profile')}>
                      <select
                        className='h-9 rounded-md border bg-background px-3 text-sm'
                        value={applicationForm.invoice_profile_id}
                        onChange={(event) =>
                          setApplicationForm((current) => ({
                            ...current,
                            invoice_profile_id: event.target.value,
                          }))
                        }
                      >
                        <option value=''>{t('Select invoice profile')}</option>
                        {(profilesQuery.data?.data.items ?? []).map((profile) => (
                          <option key={profile.id} value={profile.id}>
                            {profile.title}
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Input
                      type='number'
                      min='0'
                      step='0.01'
                      value={applicationForm.amount}
                      onChange={(event) =>
                        setApplicationForm((current) => ({
                          ...current,
                          amount: event.target.value,
                        }))
                      }
                      placeholder={t('Invoice amount')}
                    />
                    <Field label={t('Invoice Type')}>
                      <select
                        className='h-9 rounded-md border bg-background px-3 text-sm'
                        value={applicationForm.invoice_type}
                        onChange={(event) =>
                          setApplicationForm((current) => ({
                            ...current,
                            invoice_type: event.target.value,
                          }))
                        }
                      >
                        <option value='VAT_NORMAL'>{t('VAT Normal')}</option>
                        <option value='VAT_SPECIAL'>{t('VAT Special')}</option>
                      </select>
                    </Field>
                    <Button type='submit' className='w-full'>
                      {t('Submit')}
                    </Button>
                  </form>
                </CardContent>
              </Card>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Invoice Applications')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <ApplicationTable items={applicationsQuery.data?.data.items ?? []} />
                  <Pager
                    page={pages.applications}
                    pageData={applicationsQuery.data?.data}
                    onPageChange={(page) => setPage('applications', page)}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value='files'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='text-base'>{t('Invoice Files')}</CardTitle>
              </CardHeader>
              <CardContent>
                <FileTable items={filesQuery.data?.data.items ?? []} />
                <Pager
                  page={pages.files}
                  pageData={filesQuery.data?.data}
                  onPageChange={(page) => setPage('files', page)}
                />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function InvoiceAdminPortal() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<AdminTab>('applications')
  const [pages, setPages] = useState<Record<AdminTab, number>>({
    applications: 1,
    profiles: 1,
    files: 1,
  })
  const [status, setStatus] = useState('')
  const [reviewNote, setReviewNote] = useState('')
  const [issueForm, setIssueForm] = useState({
    application_id: '',
    invoice_no: '',
    invoice_date: '',
    file_name: '',
    file_url: '',
    file_type: 'PDF',
  })

  const applicationsQuery = useQuery({
    queryKey: ['admin-invoices', 'applications', pages.applications, status],
    queryFn: () =>
      getAdminInvoiceApplications({
        p: pages.applications,
        page_size: PAGE_SIZE,
        status: status || undefined,
      }),
  })
  const profilesQuery = useQuery({
    queryKey: ['admin-invoices', 'profiles', pages.profiles],
    queryFn: () =>
      getAdminInvoiceProfiles({ p: pages.profiles, page_size: PAGE_SIZE }),
  })
  const filesQuery = useQuery({
    queryKey: ['admin-invoices', 'files', pages.files],
    queryFn: () => getAdminInvoiceFiles({ p: pages.files, page_size: PAGE_SIZE }),
  })

  const refreshAll = () => {
    void applicationsQuery.refetch()
    void profilesQuery.refetch()
    void filesQuery.refetch()
  }

  const reviewMutation = useMutation({
    mutationFn: ({ id, approved }: { id: number; approved: boolean }) =>
      reviewInvoiceApplication(id, { approved, review_note: reviewNote }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setReviewNote('')
      toast.success(t('Updated successfully'))
      void applicationsQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const issueMutation = useMutation({
    mutationFn: () =>
      issueInvoice(Number(issueForm.application_id), {
        invoice_no: issueForm.invoice_no,
        invoice_date: issueForm.invoice_date
          ? Math.floor(new Date(issueForm.invoice_date).getTime() / 1000)
          : undefined,
        file_name: issueForm.file_name,
        file_url: issueForm.file_url,
        file_type: issueForm.file_type,
      }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setIssueForm({
        application_id: '',
        invoice_no: '',
        invoice_date: '',
        file_name: '',
        file_url: '',
        file_type: 'PDF',
      })
      toast.success(t('Issued successfully'))
      void applicationsQuery.refetch()
      void filesQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const setPage = (key: AdminTab, page: number) =>
    setPages((current) => ({ ...current, [key]: page }))

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Invoice Management')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button type='button' variant='outline' size='sm' onClick={refreshAll}>
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as AdminTab)}
          className='space-y-4'
        >
          <TabsList className='w-full justify-start overflow-x-auto'>
            <TabsTrigger value='applications'>
              <ReceiptText className='size-4' />
              {t('Applications')}
            </TabsTrigger>
            <TabsTrigger value='profiles'>
              <FileText className='size-4' />
              {t('Profiles')}
            </TabsTrigger>
            <TabsTrigger value='files'>
              <Upload className='size-4' />
              {t('Files')}
            </TabsTrigger>
          </TabsList>

          <TabsContent value='applications'>
            <div className='grid gap-4 xl:grid-cols-[360px_1fr]'>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Manual invoice registration')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <form
                    className='space-y-3'
                    onSubmit={(event) => {
                      event.preventDefault()
                      issueMutation.mutate()
                    }}
                  >
                    <Input
                      value={issueForm.application_id}
                      onChange={(event) =>
                        setIssueForm((current) => ({
                          ...current,
                          application_id: event.target.value,
                        }))
                      }
                      placeholder={t('Application ID')}
                    />
                    <Input
                      value={issueForm.invoice_no}
                      onChange={(event) =>
                        setIssueForm((current) => ({
                          ...current,
                          invoice_no: event.target.value,
                        }))
                      }
                      placeholder={t('Invoice number')}
                    />
                    <Input
                      type='date'
                      value={issueForm.invoice_date}
                      onChange={(event) =>
                        setIssueForm((current) => ({
                          ...current,
                          invoice_date: event.target.value,
                        }))
                      }
                    />
                    <Input
                      value={issueForm.file_name}
                      onChange={(event) =>
                        setIssueForm((current) => ({
                          ...current,
                          file_name: event.target.value,
                        }))
                      }
                      placeholder={t('File name')}
                    />
                    <Input
                      value={issueForm.file_url}
                      onChange={(event) =>
                        setIssueForm((current) => ({
                          ...current,
                          file_url: event.target.value,
                        }))
                      }
                      placeholder={t('File URL')}
                    />
                    <Button type='submit' className='w-full'>
                      {t('Register issued invoice')}
                    </Button>
                  </form>
                </CardContent>
              </Card>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='text-base'>{t('Applications')}</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className='mb-3 flex flex-wrap gap-2'>
                    <select
                      value={status}
                      onChange={(event) => setStatus(event.target.value)}
                      className='h-9 rounded-md border bg-background px-3 text-sm'
                    >
                      <option value=''>{t('All statuses')}</option>
                      <option value='PENDING'>{t('Pending')}</option>
                      <option value='APPROVED'>{t('Approved')}</option>
                      <option value='REJECTED'>{t('Rejected')}</option>
                      <option value='ISSUED'>{t('Issued')}</option>
                    </select>
                    <Input
                      value={reviewNote}
                      onChange={(event) => setReviewNote(event.target.value)}
                      placeholder={t('Review note')}
                      className='max-w-64'
                    />
                  </div>
                  <ApplicationTable
                    items={applicationsQuery.data?.data.items ?? []}
                    actions={(item) =>
                      item.status === 'PENDING' && (
                        <div className='flex gap-2'>
                          <Button
                            type='button'
                            size='sm'
                            onClick={() =>
                              reviewMutation.mutate({ id: item.id, approved: true })
                            }
                          >
                            {t('Approve')}
                          </Button>
                          <Button
                            type='button'
                            size='sm'
                            variant='destructive'
                            onClick={() =>
                              reviewMutation.mutate({ id: item.id, approved: false })
                            }
                          >
                            {t('Reject')}
                          </Button>
                        </div>
                      )
                    }
                  />
                  <Pager
                    page={pages.applications}
                    pageData={applicationsQuery.data?.data}
                    onPageChange={(page) => setPage('applications', page)}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value='profiles'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='text-base'>{t('Profiles')}</CardTitle>
              </CardHeader>
              <CardContent>
                <ProfileTable items={profilesQuery.data?.data.items ?? []} />
                <Pager
                  page={pages.profiles}
                  pageData={profilesQuery.data?.data}
                  onPageChange={(page) => setPage('profiles', page)}
                />
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value='files'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='text-base'>{t('Files')}</CardTitle>
              </CardHeader>
              <CardContent>
                <FileTable items={filesQuery.data?.data.items ?? []} />
                <Pager
                  page={pages.files}
                  pageData={filesQuery.data?.data}
                  onPageChange={(page) => setPage('files', page)}
                />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function ProfileTable({
  items,
  onDisable,
}: {
  items: InvoiceProfile[]
  onDisable?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Title')}</TableHead>
          <TableHead>{t('Type')}</TableHead>
          <TableHead>{t('Tax number')}</TableHead>
          <TableHead>{t('Default')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          {onDisable && <TableHead>{t('Actions')}</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.length ? (
          items.map((item) => (
            <TableRow key={item.id}>
              <TableCell>{item.title}</TableCell>
              <TableCell>{displayEnumLabel(item.profile_type, t)}</TableCell>
              <TableCell>{item.tax_no || '-'}</TableCell>
              <TableCell>{item.is_default ? t('Yes') : t('No')}</TableCell>
              <TableCell>
                <StatusBadge status={item.status} />
              </TableCell>
              {onDisable && (
                <TableCell>
                  {item.status === 'ACTIVE' && (
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      onClick={() => onDisable(item.id)}
                    >
                      {t('Disable')}
                    </Button>
                  )}
                </TableCell>
              )}
            </TableRow>
          ))
        ) : (
          <EmptyRows colSpan={onDisable ? 6 : 5} />
        )}
      </TableBody>
    </Table>
  )
}

function ApplicationTable({
  items,
  actions,
}: {
  items: InvoiceApplication[]
  actions?: (item: InvoiceApplication) => ReactNode
}) {
  const { t } = useTranslation()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Application No')}</TableHead>
          <TableHead>{t('User')}</TableHead>
          <TableHead>{t('Amount')}</TableHead>
          <TableHead>{t('Invoice Type')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Invoice number')}</TableHead>
          <TableHead>{t('Created At')}</TableHead>
          {actions && <TableHead>{t('Actions')}</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.length ? (
          items.map((item) => (
            <TableRow key={item.id}>
              <TableCell className='font-mono text-xs'>
                {item.application_no}
              </TableCell>
              <TableCell>#{item.user_id}</TableCell>
              <TableCell>{formatDisplayMoney(item.amount, item.currency)}</TableCell>
              <TableCell>{displayEnumLabel(item.invoice_type, t)}</TableCell>
              <TableCell>
                <StatusBadge status={item.status} />
              </TableCell>
              <TableCell>{item.invoice_no || '-'}</TableCell>
              <TableCell>{formatDisplayDateTime(item.created_at)}</TableCell>
              {actions && <TableCell>{actions(item)}</TableCell>}
            </TableRow>
          ))
        ) : (
          <EmptyRows colSpan={actions ? 8 : 7} />
        )}
      </TableBody>
    </Table>
  )
}

function FileTable({ items }: { items: InvoiceFile[] }) {
  const { t } = useTranslation()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Application ID')}</TableHead>
          <TableHead>{t('File name')}</TableHead>
          <TableHead>{t('File type')}</TableHead>
          <TableHead>{t('File URL')}</TableHead>
          <TableHead>{t('Created At')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.length ? (
          items.map((item) => (
            <TableRow key={item.id}>
              <TableCell>#{item.invoice_application_id}</TableCell>
              <TableCell>{item.file_name}</TableCell>
              <TableCell>{item.file_type}</TableCell>
              <TableCell>
                <a
                  href={item.file_url}
                  target='_blank'
                  rel='noreferrer'
                  className='text-primary underline-offset-4 hover:underline'
                >
                  {t('Open file')}
                </a>
              </TableCell>
              <TableCell>{formatDisplayDateTime(item.created_at)}</TableCell>
            </TableRow>
          ))
        ) : (
          <EmptyRows colSpan={5} />
        )}
      </TableBody>
    </Table>
  )
}
