import { useMemo, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Activity,
  BarChart3,
  CreditCard,
  Landmark,
  LineChart,
  RefreshCw,
} from 'lucide-react'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
  getFinanceRecentBilling,
  getFinanceRecentPayments,
  getFinanceRecentRedemptions,
  getFinanceRecentSubscriptions,
  getFinanceSummary,
  getFinanceTopChannels,
  getFinanceTopModels,
  getFinanceTopProviders,
  getFinanceTopTenants,
} from './api'
import {
  displayEnumLabel,
  formatDisplayDateTime,
  formatDisplayMoney,
  formatDisplayNumber,
} from '@/lib/commercial-display'
import type {
  BillingRecord,
  FinanceMetricItem,
  FinancePage,
  FinanceProviderPayment,
  FinanceSummary,
  FinanceTenantMetricItem,
  FinanceTopChannelItem,
  PaymentOrder,
  UserSubscription,
  VoucherRedemption,
} from './types'

type FinanceTab = 'dashboard' | 'rankings' | 'activity'

const PAGE_SIZE = 10

function formatQuota(value?: number, currency = 'QUOTA') {
  return `${formatDisplayNumber(value)} ${currency}`
}

function statusVariant(
  status?: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status || '').toLowerCase()
  if (['paid', 'active', 'settled', 'success', 'calculated'].includes(normalized))
    return 'default'
  if (['pending', 'draft'].includes(normalized)) return 'secondary'
  if (['failed', 'expired', 'canceled', 'cancelled', 'disabled'].includes(normalized))
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

function MetricTile({
  label,
  value,
  detail,
  icon,
}: {
  label: string
  value: ReactNode
  detail?: ReactNode
  icon?: ReactNode
}) {
  return (
    <Card className='rounded-lg'>
      <CardContent className='flex min-h-28 items-start justify-between gap-3 p-4'>
        <div className='min-w-0'>
          <div className='text-sm text-muted-foreground'>{label}</div>
          <div className='mt-2 break-words text-2xl font-semibold'>{value}</div>
          {detail && <div className='mt-1 text-xs text-muted-foreground'>{detail}</div>}
        </div>
        {icon && <div className='rounded-md border bg-muted/40 p-2 text-muted-foreground'>{icon}</div>}
      </CardContent>
    </Card>
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
  pageData?: FinancePage<unknown>
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

export function FinanceConsole() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<FinanceTab>('dashboard')
  const [days, setDays] = useState(30)
  const [pages, setPages] = useState({
    tenants: 1,
    models: 1,
    providers: 1,
    channels: 1,
    payments: 1,
    redemptions: 1,
    subscriptions: 1,
    billing: 1,
  })

  const summaryQuery = useQuery({
    queryKey: ['finance-console', 'summary', days],
    queryFn: () => getFinanceSummary({ days }),
  })
  const topTenantsQuery = useQuery({
    queryKey: ['finance-console', 'top-tenants', pages.tenants],
    queryFn: () =>
      getFinanceTopTenants({ p: pages.tenants, page_size: PAGE_SIZE }),
  })
  const topModelsQuery = useQuery({
    queryKey: ['finance-console', 'top-models', pages.models],
    queryFn: () =>
      getFinanceTopModels({ p: pages.models, page_size: PAGE_SIZE }),
  })
  const topProvidersQuery = useQuery({
    queryKey: ['finance-console', 'top-providers', pages.providers],
    queryFn: () =>
      getFinanceTopProviders({ p: pages.providers, page_size: PAGE_SIZE }),
  })
  const topChannelsQuery = useQuery({
    queryKey: ['finance-console', 'top-channels', pages.channels],
    queryFn: () =>
      getFinanceTopChannels({ p: pages.channels, page_size: PAGE_SIZE }),
  })
  const paymentsQuery = useQuery({
    queryKey: ['finance-console', 'recent-payments', pages.payments],
    queryFn: () =>
      getFinanceRecentPayments({ p: pages.payments, page_size: PAGE_SIZE }),
  })
  const redemptionsQuery = useQuery({
    queryKey: ['finance-console', 'recent-redemptions', pages.redemptions],
    queryFn: () =>
      getFinanceRecentRedemptions({
        p: pages.redemptions,
        page_size: PAGE_SIZE,
      }),
  })
  const subscriptionsQuery = useQuery({
    queryKey: ['finance-console', 'recent-subscriptions', pages.subscriptions],
    queryFn: () =>
      getFinanceRecentSubscriptions({
        p: pages.subscriptions,
        page_size: PAGE_SIZE,
      }),
  })
  const billingQuery = useQuery({
    queryKey: ['finance-console', 'recent-billing', pages.billing],
    queryFn: () =>
      getFinanceRecentBilling({ p: pages.billing, page_size: PAGE_SIZE }),
  })

  const summary = summaryQuery.data?.data
  const isRefreshing = [
    summaryQuery,
    topTenantsQuery,
    topModelsQuery,
    topProvidersQuery,
    topChannelsQuery,
    paymentsQuery,
    redemptionsQuery,
    subscriptionsQuery,
    billingQuery,
  ].some((query) => query.isFetching)

  const refetchAll = () => {
    void summaryQuery.refetch()
    void topTenantsQuery.refetch()
    void topModelsQuery.refetch()
    void topProvidersQuery.refetch()
    void topChannelsQuery.refetch()
    void paymentsQuery.refetch()
    void redemptionsQuery.refetch()
    void subscriptionsQuery.refetch()
    void billingQuery.refetch()
  }

  const tabItems = useMemo(
    () => [
      { value: 'dashboard', label: t('Dashboard'), icon: Landmark },
      { value: 'rankings', label: t('Top Ranking'), icon: BarChart3 },
      { value: 'activity', label: t('Recent Activity'), icon: Activity },
    ],
    [t]
  )

  const setPage = (key: keyof typeof pages, page: number) =>
    setPages((current) => ({ ...current, [key]: page }))

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Finance')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-2'>
          {[7, 30, 90].map((value) => (
            <Button
              key={value}
              type='button'
              variant={days === value ? 'default' : 'outline'}
              size='sm'
              onClick={() => setDays(value)}
            >
              {t('{{days}} days', { days: value })}
            </Button>
          ))}
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
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as FinanceTab)}
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

          <TabsContent value='dashboard'>
            <div className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
                <MetricTile
                  label={t('Total recharge')}
                  value={formatDisplayMoney(
                    summary?.revenue.total_recharge_amount,
                    summary?.revenue.currency
                  )}
                  detail={t('All paid payment orders')}
                  icon={<CreditCard className='size-4' />}
                />
                <MetricTile
                  label={t('This month recharge')}
                  value={formatDisplayMoney(
                    summary?.revenue.month_recharge_amount,
                    summary?.revenue.currency
                  )}
                />
                <MetricTile
                  label={t('Recent 30d recharge')}
                  value={formatDisplayMoney(
                    summary?.revenue.recent_30d_recharge,
                    summary?.revenue.currency
                  )}
                />
                <MetricTile
                  label={t('Payment success rate')}
                  value={`${formatDisplayNumber(summary?.revenue.payment_success_rate)}%`}
                  detail={`${formatDisplayNumber(summary?.revenue.paid_payment_order_count)} / ${formatDisplayNumber(summary?.revenue.payment_order_count)}`}
                />
                <MetricTile
                  label={t('Total consumption')}
                  value={formatQuota(
                    summary?.consumption.total_consumption_amount,
                    summary?.consumption.currency
                  )}
                  icon={<LineChart className='size-4' />}
                />
                <MetricTile
                  label={t('This month consumption')}
                  value={formatQuota(
                    summary?.consumption.month_consumption_amount,
                    summary?.consumption.currency
                  )}
                />
                <MetricTile
                  label={t('Total requests')}
                  value={formatDisplayNumber(summary?.consumption.total_requests)}
                />
                <MetricTile
                  label={t('Total tokens')}
                  value={formatDisplayNumber(summary?.consumption.total_tokens)}
                />
              </div>

              <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
                <MetricTile
                  label={t('Active tenants')}
                  value={formatDisplayNumber(summary?.activity.active_tenant_count)}
                />
                <MetricTile
                  label={t('Active users')}
                  value={formatDisplayNumber(summary?.activity.active_user_count)}
                />
                <MetricTile
                  label={t('Active subscriptions')}
                  value={formatDisplayNumber(
                    summary?.activity.active_subscription_count
                  )}
                />
                <MetricTile
                  label={t('Active channels')}
                  value={formatDisplayNumber(summary?.activity.active_channel_count)}
                />
              </div>

              <div className='grid gap-4 xl:grid-cols-2'>
                <PaymentDashboardPanel
                  providers={summary?.payment.provider_breakdown}
                  trend={summary?.payment.daily_trend}
                  currency={summary?.revenue.currency}
                />
                <VoucherRevenuePanel summary={summary} />
              </div>
            </div>
          </TabsContent>

          <TabsContent value='rankings'>
            <div className='grid gap-4 xl:grid-cols-2'>
              <TenantRankingTable
                title={t('Top Tenants')}
                rows={topTenantsQuery.data?.data.items}
                page={pages.tenants}
                pageData={topTenantsQuery.data?.data}
                onPageChange={(page) => setPage('tenants', page)}
              />
              <MetricRankingTable
                title={t('Top Models')}
                rows={topModelsQuery.data?.data.items}
                page={pages.models}
                pageData={topModelsQuery.data?.data}
                onPageChange={(page) => setPage('models', page)}
              />
              <MetricRankingTable
                title={t('Top Providers')}
                rows={topProvidersQuery.data?.data.items}
                page={pages.providers}
                pageData={topProvidersQuery.data?.data}
                onPageChange={(page) => setPage('providers', page)}
              />
              <ChannelRankingTable
                title={t('Top Channels')}
                rows={topChannelsQuery.data?.data.items}
                page={pages.channels}
                pageData={topChannelsQuery.data?.data}
                onPageChange={(page) => setPage('channels', page)}
              />
            </div>
          </TabsContent>

          <TabsContent value='activity'>
            <div className='grid gap-4 xl:grid-cols-2'>
              <RecentPaymentsTable
                rows={paymentsQuery.data?.data.items}
                page={pages.payments}
                pageData={paymentsQuery.data?.data}
                onPageChange={(page) => setPage('payments', page)}
              />
              <RecentRedemptionsTable
                rows={redemptionsQuery.data?.data.items}
                page={pages.redemptions}
                pageData={redemptionsQuery.data?.data}
                onPageChange={(page) => setPage('redemptions', page)}
              />
              <RecentSubscriptionsTable
                rows={subscriptionsQuery.data?.data.items}
                page={pages.subscriptions}
                pageData={subscriptionsQuery.data?.data}
                onPageChange={(page) => setPage('subscriptions', page)}
              />
              <RecentBillingTable
                rows={billingQuery.data?.data.items}
                page={pages.billing}
                pageData={billingQuery.data?.data}
                onPageChange={(page) => setPage('billing', page)}
              />
            </div>
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function PaymentDashboardPanel({
  providers,
  trend,
  currency,
}: {
  providers?: FinanceProviderPayment[]
  trend?: { date: string; amount: number; orders: number }[]
  currency?: string
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Payment Dashboard')}</CardTitle>
      </CardHeader>
      <CardContent className='space-y-4'>
        <CompactTable
          headers={[t('Provider'), t('Paid amount'), t('Orders')]}
          emptyCols={3}
          rows={(providers || []).map((item) => [
            item.provider || '-',
            formatDisplayMoney(item.amount, currency),
            formatDisplayNumber(item.orders),
          ])}
        />
        <CompactTable
          headers={[t('Date'), t('Paid amount'), t('Orders')]}
          emptyCols={3}
          rows={(trend || []).map((item) => [
            item.date,
            formatDisplayMoney(item.amount, currency),
            formatDisplayNumber(item.orders),
          ])}
        />
      </CardContent>
    </Card>
  )
}

function VoucherRevenuePanel({ summary }: { summary?: FinanceSummary }) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Voucher and Revenue Share')}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className='grid gap-3 sm:grid-cols-2'>
          <SummaryStat
            label={t('Issued vouchers')}
            value={formatDisplayNumber(summary?.voucher.total_issued)}
          />
          <SummaryStat
            label={t('Redeemed vouchers')}
            value={formatDisplayNumber(summary?.voucher.total_redeemed)}
            detail={`${formatDisplayNumber(summary?.voucher.redemption_rate)}%`}
          />
          <SummaryStat
            label={t('Platform revenue share')}
            value={formatQuota(
              summary?.revenue_share.platform_amount,
              summary?.revenue_share.currency
            )}
          />
          <SummaryStat
            label={t('Distributor revenue share')}
            value={formatQuota(
              summary?.revenue_share.distributor_amount,
              summary?.revenue_share.currency
            )}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function SummaryStat({
  label,
  value,
  detail,
}: {
  label: string
  value: ReactNode
  detail?: ReactNode
}) {
  return (
    <div className='rounded-md border bg-muted/30 p-3'>
      <div className='text-sm text-muted-foreground'>{label}</div>
      <div className='mt-2 break-words text-xl font-semibold'>{value}</div>
      {detail && <div className='mt-1 text-xs text-muted-foreground'>{detail}</div>}
    </div>
  )
}

function CompactTable({
  headers,
  rows,
  emptyCols,
}: {
  headers: string[]
  rows: ReactNode[][]
  emptyCols: number
}) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {headers.map((header) => (
            <TableHead key={header}>{header}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.length ? (
          rows.map((row, index) => (
            <TableRow key={index}>
              {row.map((cell, cellIndex) => (
                <TableCell key={cellIndex}>{cell}</TableCell>
              ))}
            </TableRow>
          ))
        ) : (
          <EmptyRows colSpan={emptyCols} />
        )}
      </TableBody>
    </Table>
  )
}

function TenantRankingTable({
  title,
  rows,
  page,
  pageData,
  onPageChange,
}: {
  title: string
  rows?: FinanceTenantMetricItem[]
  page: number
  pageData?: FinancePage<FinanceTenantMetricItem>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Tenant'), t('Amount'), t('Records')]}
          emptyCols={3}
          rows={(rows || []).map((item) => [
            item.name || `#${item.tenant_id}`,
            formatQuota(item.amount),
            formatDisplayNumber(item.count),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function MetricRankingTable({
  title,
  rows,
  page,
  pageData,
  onPageChange,
}: {
  title: string
  rows?: FinanceMetricItem[]
  page: number
  pageData?: FinancePage<FinanceMetricItem>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Name'), t('Consumption'), t('Requests'), t('Tokens')]}
          emptyCols={4}
          rows={(rows || []).map((item) => [
            item.name || '-',
            formatQuota(item.amount),
            formatDisplayNumber(item.request_count),
            formatDisplayNumber(item.total_tokens),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function ChannelRankingTable({
  title,
  rows,
  page,
  pageData,
  onPageChange,
}: {
  title: string
  rows?: FinanceTopChannelItem[]
  page: number
  pageData?: FinancePage<FinanceTopChannelItem>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Channel'), t('Gross amount'), t('Platform amount'), t('Records')]}
          emptyCols={4}
          rows={(rows || []).map((item) => [
            item.name || `#${item.distribution_channel_id}`,
            formatQuota(item.gross_amount),
            formatQuota(item.platform_amount),
            formatDisplayNumber(item.record_count),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function RecentPaymentsTable({
  rows,
  page,
  pageData,
  onPageChange,
}: {
  rows?: PaymentOrder[]
  page: number
  pageData?: FinancePage<PaymentOrder>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Recent Payments')}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Order No'), t('Provider'), t('Amount'), t('Status'), t('Created At')]}
          emptyCols={5}
          rows={(rows || []).map((item) => [
            <span className='font-mono text-xs'>{item.order_no}</span>,
            item.provider,
            formatDisplayMoney(item.amount, item.currency),
            <StatusBadge status={item.status} />,
            formatDisplayDateTime(item.created_at),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function RecentRedemptionsTable({
  rows,
  page,
  pageData,
  onPageChange,
}: {
  rows?: VoucherRedemption[]
  page: number
  pageData?: FinancePage<VoucherRedemption>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Recent Redemptions')}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Code'), t('User'), t('Type'), t('Result'), t('Created At')]}
          emptyCols={5}
          rows={(rows || []).map((item) => [
            <span className='font-mono text-xs'>{item.voucher_code}</span>,
            `#${item.user_id}`,
            displayEnumLabel(item.redemption_type, t),
            <StatusBadge status={item.redemption_result} />,
            formatDisplayDateTime(item.created_at),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function RecentSubscriptionsTable({
  rows,
  page,
  pageData,
  onPageChange,
}: {
  rows?: UserSubscription[]
  page: number
  pageData?: FinancePage<UserSubscription>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Recent Subscriptions')}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('User'), t('Plan'), t('Amount'), t('Status'), t('Ends At')]}
          emptyCols={5}
          rows={(rows || []).map((item) => [
            `#${item.user_id}`,
            `#${item.plan_id}`,
            formatQuota(item.amount_total),
            <StatusBadge status={item.status} />,
            formatDisplayDateTime(item.end_time),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}

function RecentBillingTable({
  rows,
  page,
  pageData,
  onPageChange,
}: {
  rows?: BillingRecord[]
  page: number
  pageData?: FinancePage<BillingRecord>
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <CardTitle className='text-base'>{t('Recent Billing')}</CardTitle>
      </CardHeader>
      <CardContent>
        <CompactTable
          headers={[t('Provider'), t('Model'), t('Quota'), t('Requests'), t('Created At')]}
          emptyCols={5}
          rows={(rows || []).map((item) => [
            item.provider_name || '-',
            item.model_name || '-',
            formatQuota(item.quota_charged),
            formatDisplayNumber(item.request_count),
            formatDisplayDateTime(item.created_at),
          ])}
        />
        <Pager page={page} pageData={pageData} onPageChange={onPageChange} />
      </CardContent>
    </Card>
  )
}
