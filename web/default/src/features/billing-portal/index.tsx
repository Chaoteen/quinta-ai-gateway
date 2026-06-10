/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMemo, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  CreditCard,
  Database,
  FileText,
  ListChecks,
  ReceiptText,
  RefreshCw,
  WalletCards,
} from 'lucide-react'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  getBillingPayments,
  getBillingRecords,
  getBillingSubscriptions,
  getBillingSummary,
  getBillingUsages,
} from './api'
import type {
  BillingPortalPage,
  BillingRecord,
  BillingSubscription,
  PaymentOrder,
  UsageRecord,
} from './types'

type BillingTab = 'overview' | 'payments' | 'usage' | 'bills' | 'subscriptions'

const PAGE_SIZE = 10

function formatNumber(value?: number) {
  return new Intl.NumberFormat().format(value ?? 0)
}

function formatMoney(value?: number, currency = 'USD') {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    maximumFractionDigits: 2,
  }).format(value ?? 0)
}

function formatDate(value?: number) {
  if (!value) return '-'
  return new Date(value * 1000).toLocaleString()
}

function statusVariant(
  status?: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status || '').toLowerCase()
  if (['paid', 'active', 'settled', 'committed'].includes(normalized))
    return 'default'
  if (['pending', 'reserved'].includes(normalized)) return 'secondary'
  if (['failed', 'expired', 'canceled', 'cancelled'].includes(normalized))
    return 'destructive'
  return 'outline'
}

export function BillingPortal() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<BillingTab>('overview')
  const [pages, setPages] = useState<Record<BillingTab, number>>({
    overview: 1,
    payments: 1,
    usage: 1,
    bills: 1,
    subscriptions: 1,
  })

  const summaryQuery = useQuery({
    queryKey: ['billing-portal', 'summary'],
    queryFn: getBillingSummary,
  })
  const paymentsQuery = useQuery({
    queryKey: ['billing-portal', 'payments', pages.payments],
    queryFn: () =>
      getBillingPayments({ p: pages.payments, page_size: PAGE_SIZE }),
  })
  const usageQuery = useQuery({
    queryKey: ['billing-portal', 'usage', pages.usage],
    queryFn: () => getBillingUsages({ p: pages.usage, page_size: PAGE_SIZE }),
  })
  const recordsQuery = useQuery({
    queryKey: ['billing-portal', 'records', pages.bills],
    queryFn: () => getBillingRecords({ p: pages.bills, page_size: PAGE_SIZE }),
  })
  const subscriptionsQuery = useQuery({
    queryKey: ['billing-portal', 'subscriptions', pages.subscriptions],
    queryFn: () =>
      getBillingSubscriptions({
        p: pages.subscriptions,
        page_size: PAGE_SIZE,
      }),
  })

  const summary = summaryQuery.data?.data
  const isRefreshing = [
    summaryQuery,
    paymentsQuery,
    usageQuery,
    recordsQuery,
    subscriptionsQuery,
  ].some((query) => query.isFetching)

  const refetchAll = () => {
    void summaryQuery.refetch()
    void paymentsQuery.refetch()
    void usageQuery.refetch()
    void recordsQuery.refetch()
    void subscriptionsQuery.refetch()
  }

  const tabItems = useMemo(
    () => [
      { value: 'overview', label: t('Overview'), icon: WalletCards },
      { value: 'payments', label: t('Payments'), icon: CreditCard },
      { value: 'usage', label: t('Usage'), icon: Database },
      { value: 'bills', label: t('Bills'), icon: ReceiptText },
      { value: 'subscriptions', label: t('Subscriptions'), icon: ListChecks },
    ],
    [t]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Billing')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={refetchAll}
          disabled={isRefreshing}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as BillingTab)}
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

          <TabsContent value='overview'>
            <div className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
                <MetricTile
                  label={t('Current balance')}
                  value={formatNumber(summary?.balance_quota)}
                />
                <MetricTile
                  label={t('Total payments')}
                  value={formatMoney(
                    summary?.total_recharge_amount,
                    summary?.total_recharge_currency || 'USD'
                  )}
                />
                <MetricTile
                  label={t('Total consumption')}
                  value={formatNumber(summary?.total_consumption_amount)}
                  detail={summary?.consumption_currency || 'QUOTA'}
                />
                <MetricTile
                  label={t('Last 30 days')}
                  value={formatNumber(summary?.recent_30d_consumption)}
                  detail={t('quota charged')}
                />
                <MetricTile
                  label={t('Total tokens')}
                  value={formatNumber(summary?.total_tokens)}
                />
                <MetricTile
                  label={t('Total requests')}
                  value={formatNumber(summary?.total_requests)}
                />
                <MetricTile
                  label={t('30-day tokens')}
                  value={formatNumber(summary?.recent_30d_tokens)}
                />
                <MetricTile
                  label={t('30-day requests')}
                  value={formatNumber(summary?.recent_30d_requests)}
                />
              </div>

              <div className='grid gap-4 xl:grid-cols-2'>
                <RankingTable
                  title={t('Model consumption ranking')}
                  rows={summary?.model_consumption_ranking || []}
                />
                <RankingTable
                  title={t('Provider consumption ranking')}
                  rows={summary?.provider_consumption_ranking || []}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value='payments'>
            <PaymentsTable
              page={paymentsQuery.data?.data}
              onPageChange={(page) =>
                setPages((prev) => ({ ...prev, payments: page }))
              }
            />
          </TabsContent>
          <TabsContent value='usage'>
            <UsageTable
              page={usageQuery.data?.data}
              onPageChange={(page) =>
                setPages((prev) => ({ ...prev, usage: page }))
              }
            />
          </TabsContent>
          <TabsContent value='bills'>
            <BillsTable
              page={recordsQuery.data?.data}
              onPageChange={(page) =>
                setPages((prev) => ({ ...prev, bills: page }))
              }
            />
          </TabsContent>
          <TabsContent value='subscriptions'>
            <SubscriptionsTable
              page={subscriptionsQuery.data?.data}
              onPageChange={(page) =>
                setPages((prev) => ({ ...prev, subscriptions: page }))
              }
            />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function MetricTile(props: { label: string; value: string; detail?: string }) {
  return (
    <div className='rounded-lg border px-4 py-3'>
      <div className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </div>
      <div className='mt-1 truncate text-xl font-semibold'>{props.value}</div>
      {props.detail && (
        <div className='text-muted-foreground mt-1 text-xs'>{props.detail}</div>
      )}
    </div>
  )
}

function RankingTable(props: {
  title: string
  rows: { name: string; quota_charged: number; total_tokens: number }[]
}) {
  const { t } = useTranslation()
  return (
    <div className='rounded-lg border'>
      <div className='border-b px-4 py-3 text-sm font-semibold'>
        {props.title}
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Name')}</TableHead>
            <TableHead>{t('Consumption')}</TableHead>
            <TableHead>{t('Tokens')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.rows.length === 0 ? (
            <EmptyRow colSpan={3} />
          ) : (
            props.rows.map((row) => (
              <TableRow key={row.name}>
                <TableCell className='font-medium'>{row.name || '-'}</TableCell>
                <TableCell>{formatNumber(row.quota_charged)}</TableCell>
                <TableCell>{formatNumber(row.total_tokens)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function PaymentsTable(props: {
  page?: BillingPortalPage<PaymentOrder>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataPanel page={props.page} onPageChange={props.onPageChange}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Order')}</TableHead>
            <TableHead>{t('Provider')}</TableHead>
            <TableHead>{t('Business type')}</TableHead>
            <TableHead>{t('Amount')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Created at')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!props.page?.items?.length ? (
            <EmptyRow colSpan={6} />
          ) : (
            props.page.items.map((row) => (
              <TableRow key={row.id}>
                <TableCell className='font-mono text-xs'>
                  {row.order_no}
                </TableCell>
                <TableCell>{row.provider}</TableCell>
                <TableCell>{row.business_type}</TableCell>
                <TableCell>{formatMoney(row.amount, row.currency)}</TableCell>
                <TableCell>
                  <StatusBadge status={row.status} />
                </TableCell>
                <TableCell>{formatDate(row.created_at)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </DataPanel>
  )
}

function UsageTable(props: {
  page?: BillingPortalPage<UsageRecord>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataPanel page={props.page} onPageChange={props.onPageChange}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Provider')}</TableHead>
            <TableHead>{t('Model')}</TableHead>
            <TableHead>{t('Request')}</TableHead>
            <TableHead>{t('Tokens')}</TableHead>
            <TableHead>{t('Requests')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Occurred at')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!props.page?.items?.length ? (
            <EmptyRow colSpan={7} />
          ) : (
            props.page.items.map((row) => (
              <TableRow key={row.id}>
                <TableCell>{row.provider_name || '-'}</TableCell>
                <TableCell>{row.model_name || '-'}</TableCell>
                <TableCell className='font-mono text-xs'>
                  {row.request_id || '-'}
                </TableCell>
                <TableCell>{formatNumber(row.total_tokens)}</TableCell>
                <TableCell>{formatNumber(row.request_count)}</TableCell>
                <TableCell>
                  <StatusBadge status={row.status} />
                </TableCell>
                <TableCell>{formatDate(row.occurred_at)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </DataPanel>
  )
}

function BillsTable(props: {
  page?: BillingPortalPage<BillingRecord>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataPanel page={props.page} onPageChange={props.onPageChange}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Request')}</TableHead>
            <TableHead>{t('Provider')}</TableHead>
            <TableHead>{t('Model')}</TableHead>
            <TableHead>{t('Charged quota')}</TableHead>
            <TableHead>{t('Tokens')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Created at')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!props.page?.items?.length ? (
            <EmptyRow colSpan={7} />
          ) : (
            props.page.items.map((row) => (
              <TableRow key={row.id}>
                <TableCell className='font-mono text-xs'>
                  {row.request_id || '-'}
                </TableCell>
                <TableCell>{row.provider_name || '-'}</TableCell>
                <TableCell>{row.model_name || '-'}</TableCell>
                <TableCell>{formatNumber(row.quota_charged)}</TableCell>
                <TableCell>{formatNumber(row.total_tokens)}</TableCell>
                <TableCell>
                  <StatusBadge status={row.billing_status} />
                </TableCell>
                <TableCell>{formatDate(row.created_at)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </DataPanel>
  )
}

function SubscriptionsTable(props: {
  page?: BillingPortalPage<BillingSubscription>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataPanel page={props.page} onPageChange={props.onPageChange}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Subscription')}</TableHead>
            <TableHead>{t('Plan')}</TableHead>
            <TableHead>{t('Token quota')}</TableHead>
            <TableHead>{t('Request quota')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('End time')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!props.page?.items?.length ? (
            <EmptyRow colSpan={6} />
          ) : (
            props.page.items.map((row) => (
              <TableRow key={row.id}>
                <TableCell className='font-mono text-xs'>#{row.id}</TableCell>
                <TableCell>#{row.plan_id}</TableCell>
                <TableCell>{formatNumber(row.token_quota_snapshot)}</TableCell>
                <TableCell>
                  {formatNumber(row.request_quota_snapshot)}
                </TableCell>
                <TableCell>
                  <StatusBadge status={row.lifecycle_status || row.status} />
                </TableCell>
                <TableCell>{formatDate(row.end_time)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </DataPanel>
  )
}

function DataPanel<T>(props: {
  page?: BillingPortalPage<T>
  onPageChange: (page: number) => void
  children: ReactNode
}) {
  const { t } = useTranslation()
  const page = props.page?.page || 1
  const total = props.page?.total || 0
  const pageSize = props.page?.page_size || PAGE_SIZE
  const maxPage = Math.max(1, Math.ceil(total / pageSize))
  return (
    <div className='rounded-lg border'>
      {props.children}
      <div className='flex flex-wrap items-center justify-between gap-2 border-t px-4 py-3 text-sm'>
        <div className='text-muted-foreground'>
          {t('Total')}: {formatNumber(total)}
        </div>
        <div className='flex items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={page <= 1}
            onClick={() => props.onPageChange(page - 1)}
          >
            {t('Previous')}
          </Button>
          <span className='text-muted-foreground min-w-16 text-center text-xs'>
            {page} / {maxPage}
          </span>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={page >= maxPage}
            onClick={() => props.onPageChange(page + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
    </div>
  )
}

function StatusBadge(props: { status?: string }) {
  return (
    <Badge variant={statusVariant(props.status)}>
      {(props.status || '-').toUpperCase()}
    </Badge>
  )
}

function EmptyRow(props: { colSpan: number }) {
  const { t } = useTranslation()
  return (
    <TableRow>
      <TableCell
        colSpan={props.colSpan}
        className='text-muted-foreground h-24 text-center'
      >
        <FileText className='mx-auto mb-2 size-5 opacity-60' />
        {t('No data')}
      </TableCell>
    </TableRow>
  )
}
