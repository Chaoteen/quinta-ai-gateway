import { useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Ban,
  Gift,
  History,
  Layers3,
  Plus,
  RefreshCw,
  Ticket,
} from 'lucide-react'
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
} from '@/lib/commercial-display'
import { getUserRoleKey, ROLE_KEY } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import {
  createVoucherBatch,
  disableVoucher,
  disableVoucherBatch,
  generateVouchers,
  getVoucherBatches,
  getVoucherHistory,
  getVoucherRedemptions,
  getVouchers,
  redeemVoucher,
} from './api'
import type {
  Voucher,
  VoucherBatch,
  VoucherPage,
  VoucherRedemption,
} from './types'

const PAGE_SIZE = 10

type AdminTab = 'batches' | 'vouchers' | 'redemptions'

function statusVariant(
  status?: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status || '').toLowerCase()
  if (['active', 'unused', 'success'].includes(normalized)) return 'default'
  if (['draft', 'pending'].includes(normalized)) return 'secondary'
  if (['disabled', 'expired', 'failed'].includes(normalized))
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

function Pager({
  page,
  pageData,
  onPageChange,
}: {
  page: number
  pageData?: VoucherPage<unknown>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const total = pageData?.total ?? 0
  const maxPage = Math.max(1, Math.ceil(total / PAGE_SIZE))
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

export function VoucherPortal() {
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const [page, setPage] = useState(1)
  const [lastRedemption, setLastRedemption] =
    useState<VoucherRedemption | null>(null)

  const historyQuery = useQuery({
    queryKey: ['vouchers', 'history', page],
    queryFn: () => getVoucherHistory({ p: page, page_size: PAGE_SIZE }),
  })

  const redeemMutation = useMutation({
    mutationFn: () => redeemVoucher(code),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setCode('')
      setLastRedemption(res.data)
      toast.success(t('Redeemed successfully'))
      void historyQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Voucher')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void historyQuery.refetch()}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='grid gap-4 xl:grid-cols-[360px_1fr]'>
          <Card className='rounded-lg'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2 text-base'>
                <Gift className='size-4' />
                {t('Voucher Redemption')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form
                className='space-y-3'
                onSubmit={(event) => {
                  event.preventDefault()
                  if (!code.trim()) {
                    toast.error(t('Voucher code is required'))
                    return
                  }
                  redeemMutation.mutate()
                }}
              >
                <Input
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                  placeholder={t('Voucher code')}
                  className='font-mono uppercase'
                />
                <Button
                  type='submit'
                  className='w-full'
                  disabled={redeemMutation.isPending}
                >
                  <Ticket className='size-4' />
                  {t('Redeem')}
                </Button>
              </form>
              {lastRedemption && (
                <div className='mt-4 rounded-md border bg-muted/30 p-3 text-sm'>
                  <div className='mb-2 font-medium'>{t('Latest result')}</div>
                  <div className='grid gap-2 text-muted-foreground'>
                    <div className='flex justify-between gap-3'>
                      <span>{t('Code')}</span>
                      <span className='font-mono text-foreground'>
                        {lastRedemption.voucher_code}
                      </span>
                    </div>
                    <div className='flex justify-between gap-3'>
                      <span>{t('Type')}</span>
                      <span className='text-foreground'>
                        {displayEnumLabel(lastRedemption.redemption_type, t)}
                      </span>
                    </div>
                    <div className='flex justify-between gap-3'>
                      <span>{t('Result')}</span>
                      <StatusBadge status={lastRedemption.redemption_result} />
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <Card className='rounded-lg'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2 text-base'>
                <History className='size-4' />
                {t('Redemption History')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <RedemptionTable items={historyQuery.data?.data.items ?? []} />
              <Pager
                page={page}
                pageData={historyQuery.data?.data}
                onPageChange={setPage}
              />
            </CardContent>
          </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function VoucherAdminPortal() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const roleKey = getUserRoleKey(user)
  const canManage =
    roleKey === ROLE_KEY.ROOT || roleKey === ROLE_KEY.TENANT_ADMIN
  const canView = canManage || roleKey === ROLE_KEY.FINANCE
  const initialTab: AdminTab = canManage ? 'batches' : 'redemptions'
  const [activeTab, setActiveTab] = useState<AdminTab>(initialTab)
  const [pages, setPages] = useState<Record<AdminTab, number>>({
    batches: 1,
    vouchers: 1,
    redemptions: 1,
  })
  const [batchName, setBatchName] = useState('')
  const [voucherType, setVoucherType] = useState('TOKEN')
  const [generateBatchId, setGenerateBatchId] = useState('')
  const [generateQuantity, setGenerateQuantity] = useState('1')
  const [quotaAmount, setQuotaAmount] = useState('1000')
  const [subscriptionPlanId, setSubscriptionPlanId] = useState('')
  const [batchKeyword, setBatchKeyword] = useState('')
  const [batchStatus, setBatchStatus] = useState('')
  const [batchType, setBatchType] = useState('')
  const [voucherStatus, setVoucherStatus] = useState('')
  const [voucherKeyword, setVoucherKeyword] = useState('')
  const [voucherBatchId, setVoucherBatchId] = useState('')
  const [voucherTypeFilter, setVoucherTypeFilter] = useState('')
  const [redemptionKeyword, setRedemptionKeyword] = useState('')
  const [redemptionStatus, setRedemptionStatus] = useState('')
  const [redemptionType, setRedemptionType] = useState('')

  const batchesQuery = useQuery({
    queryKey: [
      'admin-vouchers',
      'batches',
      pages.batches,
      batchKeyword,
      batchStatus,
      batchType,
    ],
    queryFn: () =>
      getVoucherBatches({
        p: pages.batches,
        page_size: PAGE_SIZE,
        keyword: batchKeyword || undefined,
        status: batchStatus || undefined,
        voucher_type: batchType || undefined,
      }),
    enabled: canManage,
  })
  const vouchersQuery = useQuery({
    queryKey: [
      'admin-vouchers',
      'vouchers',
      pages.vouchers,
      voucherStatus,
      voucherKeyword,
      voucherBatchId,
      voucherTypeFilter,
    ],
    queryFn: () =>
      getVouchers({
        p: pages.vouchers,
        page_size: PAGE_SIZE,
        status: voucherStatus || undefined,
        keyword: voucherKeyword || undefined,
        batch_id: Number(voucherBatchId) || undefined,
        voucher_type: voucherTypeFilter || undefined,
      }),
    enabled: canManage,
  })
  const redemptionsQuery = useQuery({
    queryKey: [
      'admin-vouchers',
      'redemptions',
      pages.redemptions,
      redemptionKeyword,
      redemptionStatus,
      redemptionType,
    ],
    queryFn: () =>
      getVoucherRedemptions({
        p: pages.redemptions,
        page_size: PAGE_SIZE,
        keyword: redemptionKeyword || undefined,
        status: redemptionStatus || undefined,
        voucher_type: redemptionType || undefined,
      }),
    enabled: canView,
  })

  const refreshAll = () => {
    void batchesQuery.refetch()
    void vouchersQuery.refetch()
    void redemptionsQuery.refetch()
  }

  const createBatchMutation = useMutation({
    mutationFn: () =>
      createVoucherBatch({
        name: batchName,
        voucher_type: voucherType,
        status: 'ACTIVE',
      }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      setBatchName('')
      toast.success(t('Created successfully'))
      void batchesQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const generateMutation = useMutation({
    mutationFn: () =>
      generateVouchers(Number(generateBatchId), {
        quantity: Number(generateQuantity),
        quota_amount: Number(quotaAmount) || undefined,
        subscription_plan_id: Number(subscriptionPlanId) || undefined,
      }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      toast.success(t('Generated successfully'))
      void batchesQuery.refetch()
      void vouchersQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const disableVoucherMutation = useMutation({
    mutationFn: disableVoucher,
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      toast.success(t('Disabled successfully'))
      void vouchersQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const disableBatchMutation = useMutation({
    mutationFn: disableVoucherBatch,
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Operation failed'))
        return
      }
      toast.success(t('Disabled successfully'))
      void batchesQuery.refetch()
      void vouchersQuery.refetch()
    },
    onError: () => toast.error(t('Operation failed')),
  })

  const tabItems = useMemo(
    () =>
      [
        canManage && {
          value: 'batches',
          label: t('Voucher Batch'),
          icon: Layers3,
        },
        canManage && {
          value: 'vouchers',
          label: t('Voucher List'),
          icon: Ticket,
        },
        {
          value: 'redemptions',
          label: t('Redemption History'),
          icon: History,
        },
      ].filter(Boolean) as {
        value: AdminTab
        label: string
        icon: typeof Layers3
      }[],
    [canManage, t]
  )

  if (!canView) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Voucher Management')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <Card className='rounded-lg'>
            <CardContent className='py-8 text-center text-muted-foreground'>
              {t('Access denied')}
            </CardContent>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Voucher Management')}</SectionPageLayout.Title>
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
            {tabItems.map((item) => {
              const Icon = item.icon
              return (
                <TabsTrigger key={item.value} value={item.value}>
                  <Icon className='size-4' />
                  {item.label}
                </TabsTrigger>
              )
            })}
          </TabsList>

          {canManage && (
            <TabsContent value='batches'>
              <div className='grid gap-4 xl:grid-cols-[360px_1fr]'>
                <Card className='rounded-lg'>
                  <CardHeader>
                    <CardTitle className='text-base'>
                      {t('Create batch')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <form
                      className='space-y-3'
                      onSubmit={(event) => {
                        event.preventDefault()
                        if (!batchName.trim()) {
                          toast.error(t('Name is required'))
                          return
                        }
                        createBatchMutation.mutate()
                      }}
                    >
                      <Input
                        value={batchName}
                        onChange={(event) => setBatchName(event.target.value)}
                        placeholder={t('Batch name')}
                      />
                      <select
                        value={voucherType}
                        onChange={(event) => setVoucherType(event.target.value)}
                        className='h-9 w-full rounded-md border bg-background px-3 text-sm'
                      >
                        <option value='TOKEN'>{t('Token voucher')}</option>
                        <option value='SUBSCRIPTION'>
                          {t('Subscription voucher')}
                        </option>
                      </select>
                      <Button
                        type='submit'
                        className='w-full'
                        disabled={createBatchMutation.isPending}
                      >
                        <Plus className='size-4' />
                        {t('Create')}
                      </Button>
                    </form>
                  </CardContent>
                </Card>

                <Card className='rounded-lg'>
                  <CardHeader>
                    <CardTitle className='text-base'>
                      {t('Generate vouchers')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <form
                      className='grid gap-3 md:grid-cols-5'
                      onSubmit={(event) => {
                        event.preventDefault()
                        if (!Number(generateBatchId)) {
                          toast.error(t('Batch ID is required'))
                          return
                        }
                        generateMutation.mutate()
                      }}
                    >
                      <Input
                        value={generateBatchId}
                        onChange={(event) =>
                          setGenerateBatchId(event.target.value)
                        }
                        placeholder={t('Batch ID')}
                        inputMode='numeric'
                      />
                      <Input
                        value={generateQuantity}
                        onChange={(event) =>
                          setGenerateQuantity(event.target.value)
                        }
                        placeholder={t('Quantity')}
                        inputMode='numeric'
                      />
                      <Input
                        value={quotaAmount}
                        onChange={(event) => setQuotaAmount(event.target.value)}
                        placeholder={t('Quota amount')}
                        inputMode='numeric'
                      />
                      <Input
                        value={subscriptionPlanId}
                        onChange={(event) =>
                          setSubscriptionPlanId(event.target.value)
                        }
                        placeholder={t('Plan ID')}
                        inputMode='numeric'
                      />
                      <Button
                        type='submit'
                        disabled={generateMutation.isPending}
                      >
                        <Ticket className='size-4' />
                        {t('Generate')}
                      </Button>
                    </form>
                  </CardContent>
                </Card>

                <Card className='rounded-lg xl:col-span-2'>
                  <CardContent className='pt-6'>
                    <FilterBar>
                      <Input
                        value={batchKeyword}
                        onChange={(event) => {
                          setBatchKeyword(event.target.value)
                          setPages((prev) => ({ ...prev, batches: 1 }))
                        }}
                        placeholder={t('Search')}
                        className='w-56'
                      />
                      <StatusSelect
                        value={batchStatus}
                        onChange={(value) => {
                          setBatchStatus(value)
                          setPages((prev) => ({ ...prev, batches: 1 }))
                        }}
                        options={['DRAFT', 'ACTIVE', 'DISABLED', 'FINISHED']}
                      />
                      <TypeSelect
                        value={batchType}
                        onChange={(value) => {
                          setBatchType(value)
                          setPages((prev) => ({ ...prev, batches: 1 }))
                        }}
                      />
                    </FilterBar>
                    <BatchTable
                      items={batchesQuery.data?.data.items ?? []}
                      onDisable={(id) => disableBatchMutation.mutate(id)}
                    />
                    <Pager
                      page={pages.batches}
                      pageData={batchesQuery.data?.data}
                      onPageChange={(page) =>
                        setPages((prev) => ({ ...prev, batches: page }))
                      }
                    />
                  </CardContent>
                </Card>
              </div>
            </TabsContent>
          )}

          {canManage && (
            <TabsContent value='vouchers'>
              <Card className='rounded-lg'>
                <CardHeader>
                  <CardTitle className='flex flex-wrap items-center gap-2 text-base'>
                    <Ticket className='size-4' />
                    {t('Voucher List')}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className='mb-3 flex flex-wrap gap-2'>
                    <Input
                      value={voucherKeyword}
                      onChange={(event) => {
                        setVoucherKeyword(event.target.value)
                        setPages((prev) => ({ ...prev, vouchers: 1 }))
                      }}
                      placeholder={t('Search')}
                      className='w-56'
                    />
                    <Input
                      value={voucherBatchId}
                      onChange={(event) => {
                        setVoucherBatchId(event.target.value)
                        setPages((prev) => ({ ...prev, vouchers: 1 }))
                      }}
                      placeholder={t('Batch ID')}
                      inputMode='numeric'
                      className='w-36'
                    />
                    <select
                      value={voucherStatus}
                      onChange={(event) => {
                        setVoucherStatus(event.target.value)
                        setPages((prev) => ({ ...prev, vouchers: 1 }))
                      }}
                      className='h-9 w-40 rounded-md border bg-background px-3 text-sm'
                    >
                      <option value=''>{t('All statuses')}</option>
                      <option value='UNUSED'>{t('Unused')}</option>
                      <option value='REDEEMED'>{t('Redeemed')}</option>
                      <option value='EXPIRED'>{t('Expired')}</option>
                      <option value='DISABLED'>{t('Disabled')}</option>
                    </select>
                    <TypeSelect
                      value={voucherTypeFilter}
                      onChange={(value) => {
                        setVoucherTypeFilter(value)
                        setPages((prev) => ({ ...prev, vouchers: 1 }))
                      }}
                    />
                  </div>
                  <VoucherTable
                    items={vouchersQuery.data?.data.items ?? []}
                    onDisable={(id) => disableVoucherMutation.mutate(id)}
                  />
                  <Pager
                    page={pages.vouchers}
                    pageData={vouchersQuery.data?.data}
                    onPageChange={(page) =>
                      setPages((prev) => ({ ...prev, vouchers: page }))
                    }
                  />
                </CardContent>
              </Card>
            </TabsContent>
          )}

          <TabsContent value='redemptions'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='flex items-center gap-2 text-base'>
                  <History className='size-4' />
                  {t('Redemption History')}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <FilterBar>
                  <Input
                    value={redemptionKeyword}
                    onChange={(event) => {
                      setRedemptionKeyword(event.target.value)
                      setPages((prev) => ({ ...prev, redemptions: 1 }))
                    }}
                    placeholder={t('Search')}
                    className='w-56'
                  />
                  <StatusSelect
                    value={redemptionStatus}
                    onChange={(value) => {
                      setRedemptionStatus(value)
                      setPages((prev) => ({ ...prev, redemptions: 1 }))
                    }}
                    options={['SUCCESS', 'IGNORED', 'FAILED']}
                  />
                  <TypeSelect
                    value={redemptionType}
                    onChange={(value) => {
                      setRedemptionType(value)
                      setPages((prev) => ({ ...prev, redemptions: 1 }))
                    }}
                  />
                </FilterBar>
                <RedemptionTable
                  items={redemptionsQuery.data?.data.items ?? []}
                  showUser
                />
                <Pager
                  page={pages.redemptions}
                  pageData={redemptionsQuery.data?.data}
                  onPageChange={(page) =>
                    setPages((prev) => ({ ...prev, redemptions: page }))
                  }
                />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function BatchTable({
  items,
  onDisable,
}: {
  items: VoucherBatch[]
  onDisable: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataTable
      headers={[
        'ID',
        t('Batch No'),
        t('Name'),
        t('Type'),
        t('Quantity'),
        t('Status'),
        t('Created At'),
        t('Actions'),
      ]}
    >
      {items.length === 0 ? (
        <EmptyRows colSpan={8} />
      ) : (
        items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>{item.id}</TableCell>
            <TableCell className='font-mono'>{item.batch_no}</TableCell>
            <TableCell>{item.name}</TableCell>
            <TableCell>{displayEnumLabel(item.voucher_type, t)}</TableCell>
            <TableCell>{item.quantity}</TableCell>
            <TableCell>
              <StatusBadge status={item.status} />
            </TableCell>
            <TableCell>{formatDisplayDateTime(item.created_at)}</TableCell>
            <TableCell>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={item.status === 'DISABLED'}
                onClick={() => onDisable(item.id)}
              >
                <Ban className='size-4' />
                {t('Disable')}
              </Button>
            </TableCell>
          </TableRow>
        ))
      )}
    </DataTable>
  )
}

function VoucherTable({
  items,
  onDisable,
}: {
  items: Voucher[]
  onDisable: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataTable
      headers={[
        'ID',
        t('Code'),
        t('Batch ID'),
        t('Type'),
        t('Quota amount'),
        t('Plan ID'),
        t('Status'),
        t('Activated By'),
        t('Expires'),
        t('Actions'),
      ]}
    >
      {items.length === 0 ? (
        <EmptyRows colSpan={10} />
      ) : (
        items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>{item.id}</TableCell>
            <TableCell className='font-mono'>{item.voucher_code}</TableCell>
            <TableCell>{item.batch_id}</TableCell>
            <TableCell>{displayEnumLabel(item.voucher_type, t)}</TableCell>
            <TableCell>{item.quota_amount || '-'}</TableCell>
            <TableCell>{item.subscription_plan_id || '-'}</TableCell>
            <TableCell>
              <StatusBadge status={item.status} />
            </TableCell>
            <TableCell>{item.activated_by || '-'}</TableCell>
            <TableCell>{formatDisplayDateTime(item.expired_at)}</TableCell>
            <TableCell>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={item.status !== 'UNUSED'}
                onClick={() => onDisable(item.id)}
              >
                <Ban className='size-4' />
                {t('Disable')}
              </Button>
            </TableCell>
          </TableRow>
        ))
      )}
    </DataTable>
  )
}

function RedemptionTable({
  items,
  showUser = false,
}: {
  items: VoucherRedemption[]
  showUser?: boolean
}) {
  const { t } = useTranslation()
  const headers = [
    'ID',
    t('Code'),
    showUser ? t('User ID') : null,
    t('Type'),
    t('Result'),
    t('Redeemed At'),
  ].filter(Boolean) as string[]
  return (
    <DataTable headers={headers}>
      {items.length === 0 ? (
        <EmptyRows colSpan={headers.length} />
      ) : (
        items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>{item.id}</TableCell>
            <TableCell className='font-mono'>{item.voucher_code}</TableCell>
            {showUser && <TableCell>{item.user_id}</TableCell>}
            <TableCell>{displayEnumLabel(item.redemption_type, t)}</TableCell>
            <TableCell>
              <StatusBadge status={item.redemption_result} />
            </TableCell>
            <TableCell>{formatDisplayDateTime(item.created_at)}</TableCell>
          </TableRow>
        ))
      )}
    </DataTable>
  )
}

function FilterBar({ children }: { children: ReactNode }) {
  return <div className='mb-3 flex flex-wrap gap-2'>{children}</div>
}

function StatusSelect({
  value,
  onChange,
  options,
}: {
  value: string
  onChange: (value: string) => void
  options: string[]
}) {
  const { t } = useTranslation()
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className='h-9 w-40 rounded-md border bg-background px-3 text-sm'
    >
      <option value=''>{t('All statuses')}</option>
      {options.map((option) => (
        <option key={option} value={option}>
          {displayEnumLabel(option, t)}
        </option>
      ))}
    </select>
  )
}

function TypeSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className='h-9 w-48 rounded-md border bg-background px-3 text-sm'
    >
      <option value=''>{t('All types')}</option>
      <option value='TOKEN'>{t('Token voucher')}</option>
      <option value='SUBSCRIPTION'>{t('Subscription voucher')}</option>
    </select>
  )
}

function DataTable({
  headers,
  children,
}: {
  headers: string[]
  children: ReactNode
}) {
  return (
    <div className='overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow>
            {headers.map((header) => (
              <TableHead key={header}>{header}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>{children}</TableBody>
      </Table>
    </div>
  )
}
