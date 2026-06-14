import { useMemo, type ElementType } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  BadgeDollarSign,
  BarChart3,
  CreditCard,
  Landmark,
  LineChart,
  ReceiptText,
  RefreshCw,
  ShieldCheck,
  Timer,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getFinanceSummary } from '@/features/finance-console/api'
import {
  getAdminBankTransfers,
  getAdminPaymentOrders,
  getCommercialBillingRecords,
  getCommercialBillingSummary,
  getCommercialUsageRecords,
} from './api'
import {
  displayEnumLabel,
  formatDisplayDateTime,
  formatDisplayMoney,
  formatDisplayNumber,
} from '@/lib/commercial-display'

const PAGE_SIZE = 8

function statusVariant(
  status?: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status || '').toLowerCase()
  if (['paid', 'approved', 'issued', 'active', 'success'].includes(normalized))
    return 'default'
  if (['pending', 'processing'].includes(normalized)) return 'secondary'
  if (['failed', 'rejected', 'disabled', 'canceled', 'expired'].includes(normalized))
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

function MetricCard(props: {
  title: string
  value: string
  description: string
  icon: ElementType
}) {
  const Icon = props.icon

  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle className='flex items-center gap-2 text-sm'>
          <span className='bg-muted flex size-8 items-center justify-center rounded-lg'>
            <Icon className='size-4' aria-hidden='true' />
          </span>
          {props.title}
        </CardTitle>
        <CardDescription>{props.description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className='text-2xl font-semibold tabular-nums'>
          {props.value}
        </div>
      </CardContent>
    </Card>
  )
}

function EmptyState({ message }: { message: string }) {
  return (
    <TableRow>
      <TableCell colSpan={5} className='h-24 text-center text-muted-foreground'>
        {message}
      </TableCell>
    </TableRow>
  )
}

function FoundationPreview(props: { title: string; description: string }) {
  const { t } = useTranslation()

  return (
    <Card className='border-dashed'>
      <CardHeader>
        <CardTitle className='flex items-center gap-2'>
          <ShieldCheck className='size-4' />
          {props.title}
        </CardTitle>
        <CardDescription>{props.description}</CardDescription>
      </CardHeader>
      <CardContent>
        <Badge variant='outline'>{t('Foundation Preview')}</Badge>
      </CardContent>
    </Card>
  )
}

export function QuotaDashboard() {
  const { t } = useTranslation()
  const summaryQuery = useQuery({
    queryKey: ['commercial', 'quota-dashboard', 'summary'],
    queryFn: getCommercialBillingSummary,
    retry: false,
  })
  const recordsQuery = useQuery({
    queryKey: ['commercial', 'quota-dashboard', 'records'],
    queryFn: () => getCommercialBillingRecords({ p: 1, page_size: PAGE_SIZE }),
    retry: false,
  })
  const summary = summaryQuery.data?.data
  const records = recordsQuery.data?.data?.items ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Quota Dashboard')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Track tenant quota balance and consumption readiness.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Button
          size='sm'
          variant='outline'
          onClick={() => void summaryQuery.refetch()}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
          <MetricCard
            title={t('Total Quota')}
            value={formatDisplayNumber(
              (summary?.balance_quota ?? 0) + (summary?.total_consumption_amount ?? 0)
            )}
            description={t('Available quota plus billed consumption')}
            icon={WalletCards}
          />
          <MetricCard
            title={t('Available Quota')}
            value={formatDisplayNumber(summary?.balance_quota)}
            description={t('Current quota available for requests')}
            icon={BadgeDollarSign}
          />
          <MetricCard
            title={t('Frozen Quota')}
            value='0'
            description={t('Reserved quota is not tracked in this foundation view')}
            icon={ShieldCheck}
          />
          <MetricCard
            title={t('Consumed Quota')}
            value={formatDisplayNumber(summary?.total_consumption_amount)}
            description={t('Cumulative billed consumption')}
            icon={ReceiptText}
          />
        </div>
        <FoundationPreview
          title={t('Quota Trend')}
          description={t(
            'Quota runtime foundation is available; a dedicated historical quota trend API is planned for the next analytics iteration.'
          )}
        />
        <Card>
          <CardHeader>
            <CardTitle>{t('Recent Quota Changes')}</CardTitle>
            <CardDescription>
              {t('Latest quota charges from billing records')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Created At')}</TableHead>
                  <TableHead>{t('Provider')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Quota Change')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.length === 0 ? (
                  <EmptyState message={t('No data')} />
                ) : (
                  records.map((record) => (
                    <TableRow key={record.id}>
                      <TableCell>{formatDisplayDateTime(record.created_at)}</TableCell>
                      <TableCell>{record.provider_name || '-'}</TableCell>
                      <TableCell>{record.model_name || '-'}</TableCell>
                      <TableCell>
                        -{formatDisplayNumber(record.quota_charged)}
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={record.billing_status} />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function UsageAnalyticsDashboard() {
  const { t } = useTranslation()
  const summaryQuery = useQuery({
    queryKey: ['commercial', 'usage-analytics', 'summary'],
    queryFn: getCommercialBillingSummary,
    retry: false,
  })
  const usagesQuery = useQuery({
    queryKey: ['commercial', 'usage-analytics', 'records'],
    queryFn: () => getCommercialUsageRecords({ p: 1, page_size: PAGE_SIZE }),
    retry: false,
  })
  const summary = summaryQuery.data?.data
  const usages = usagesQuery.data?.data?.items ?? []
  const tokenTotals = useMemo(
    () =>
      usages.reduce(
        (acc, row) => {
          acc.input += row.input_tokens ?? 0
          acc.output += row.output_tokens ?? 0
          acc.total += row.total_tokens ?? 0
          return acc
        },
        { input: 0, output: 0, total: 0 }
      ),
    [usages]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Usage Analytics')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Analyze request volume, token usage, model ranking, and provider ranking.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
          <MetricCard
            title={t('Total Requests')}
            value={formatDisplayNumber(summary?.total_requests)}
            description={t('Cumulative request count')}
            icon={LineChart}
          />
          <MetricCard
            title={t('Input Tokens')}
            value={formatDisplayNumber(tokenTotals.input)}
            description={t('Input token total from recent usage records')}
            icon={Timer}
          />
          <MetricCard
            title={t('Output Tokens')}
            value={formatDisplayNumber(tokenTotals.output)}
            description={t('Output token total from recent usage records')}
            icon={Timer}
          />
          <MetricCard
            title={t('Total Tokens')}
            value={formatDisplayNumber(summary?.total_tokens ?? tokenTotals.total)}
            description={t('Cumulative token usage')}
            icon={Timer}
          />
        </div>
        <div className='grid gap-4 md:grid-cols-2'>
          <MetricCard
            title={t('Recent 30d Tokens')}
            value={formatDisplayNumber(summary?.recent_30d_tokens)}
            description={t('Token usage over the last 30 days')}
            icon={BarChart3}
          />
        </div>
        <div className='grid gap-4 xl:grid-cols-2'>
          <RankingCard
            title={t('Model Usage Ranking')}
            rows={summary?.model_consumption_ranking}
          />
          <RankingCard
            title={t('Provider Usage Ranking')}
            rows={summary?.provider_consumption_ranking}
          />
        </div>
        <Card>
          <CardHeader>
            <CardTitle>{t('Recent Usage')}</CardTitle>
            <CardDescription>{t('Latest metered request records')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Provider')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Requests')}</TableHead>
                  <TableHead>{t('Tokens')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usages.length === 0 ? (
                  <EmptyState message={t('No data')} />
                ) : (
                  usages.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell>{row.provider_name || '-'}</TableCell>
                      <TableCell>{row.model_name || '-'}</TableCell>
                      <TableCell>{formatDisplayNumber(row.request_count)}</TableCell>
                      <TableCell>{formatDisplayNumber(row.total_tokens)}</TableCell>
                      <TableCell>
                        <StatusBadge status={row.status} />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function RankingCard({
  title,
  rows,
}: {
  title: string
  rows?: Array<{ name: string; quota_charged: number; total_tokens: number; request_count: number }>
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className='divide-border rounded-lg border'>
          {(rows ?? []).length === 0 ? (
            <div className='text-muted-foreground px-4 py-8 text-center text-sm'>
              {t('No data')}
            </div>
          ) : (
            (rows ?? []).slice(0, 8).map((row) => (
              <div
                key={row.name}
                className='flex items-center justify-between gap-4 px-4 py-3 text-sm'
              >
                <div className='min-w-0'>
                  <div className='truncate font-medium'>{row.name || '-'}</div>
                  <div className='text-muted-foreground text-xs'>
                    {formatDisplayNumber(row.request_count)} {t('Requests')} ·{' '}
                    {formatDisplayNumber(row.total_tokens)} {t('Tokens')}
                  </div>
                </div>
                <Badge variant='outline'>{formatDisplayNumber(row.quota_charged)}</Badge>
              </div>
            ))
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export function BillingDashboard() {
  const { t } = useTranslation()
  const summaryQuery = useQuery({
    queryKey: ['commercial', 'billing-dashboard', 'summary'],
    queryFn: getCommercialBillingSummary,
    retry: false,
  })
  const recordsQuery = useQuery({
    queryKey: ['commercial', 'billing-dashboard', 'records'],
    queryFn: () => getCommercialBillingRecords({ p: 1, page_size: PAGE_SIZE }),
    retry: false,
  })
  const summary = summaryQuery.data?.data
  const records = recordsQuery.data?.data?.items ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Billing Dashboard')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Review consumption, trends, and model billing records.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='grid gap-4 md:grid-cols-3'>
          <MetricCard
            title={t('Today Consumption')}
            value={t('Foundation Preview')}
            description={t('Daily billing breakdown is reserved for analytics API')}
            icon={ReceiptText}
          />
          <MetricCard
            title={t('This Month Consumption')}
            value={formatDisplayMoney(
              summary?.recent_30d_consumption,
              summary?.consumption_currency
            )}
            description={t('Current foundation view uses recent 30d consumption')}
            icon={LineChart}
          />
          <MetricCard
            title={t('Total Consumption')}
            value={formatDisplayMoney(
              summary?.total_consumption_amount,
              summary?.consumption_currency
            )}
            description={t('Cumulative billing runtime amount')}
            icon={BadgeDollarSign}
          />
        </div>
        <RankingCard
          title={t('Model Consumption Ranking')}
          rows={summary?.model_consumption_ranking}
        />
        <Card>
          <CardHeader>
            <CardTitle>{t('Recent Billing Records')}</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Provider')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Tokens')}</TableHead>
                  <TableHead>{t('Consumption')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.length === 0 ? (
                  <EmptyState message={t('No data')} />
                ) : (
                  records.map((record) => (
                    <TableRow key={record.id}>
                      <TableCell>{record.provider_name || '-'}</TableCell>
                      <TableCell>{record.model_name || '-'}</TableCell>
                      <TableCell>{formatDisplayNumber(record.total_tokens)}</TableCell>
                      <TableCell>{formatDisplayNumber(record.quota_charged)}</TableCell>
                      <TableCell>
                        <StatusBadge status={record.billing_status} />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function PaymentCenter() {
  const { t } = useTranslation()
  const paymentsQuery = useQuery({
    queryKey: ['commercial', 'payment-center', 'orders'],
    queryFn: () => getAdminPaymentOrders({ p: 1, page_size: PAGE_SIZE }),
    retry: false,
  })
  const transfersQuery = useQuery({
    queryKey: ['commercial', 'payment-center', 'bank-transfers'],
    queryFn: () => getAdminBankTransfers({ p: 1, page_size: PAGE_SIZE }),
    retry: false,
  })
  const payments = paymentsQuery.data?.data?.items ?? []
  const transfers = transfersQuery.data?.data?.items ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Payment Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Track payment orders, bank transfers, and manual review status.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <Card>
          <CardHeader>
            <CardTitle>{t('Payment Orders')}</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Order No')}</TableHead>
                  <TableHead>{t('Provider')}</TableHead>
                  <TableHead>{t('Amount')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Created At')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {payments.length === 0 ? (
                  <EmptyState message={t('No data')} />
                ) : (
                  payments.map((order) => (
                    <TableRow key={order.id}>
                      <TableCell>{order.order_no}</TableCell>
                      <TableCell>{order.provider}</TableCell>
                      <TableCell>{formatDisplayMoney(order.amount, order.currency)}</TableCell>
                      <TableCell>
                        <StatusBadge status={order.status} />
                      </TableCell>
                      <TableCell>{formatDisplayDateTime(order.created_at)}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t('Bank Transfer Records')}</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Account Name')}</TableHead>
                  <TableHead>{t('Amount')}</TableHead>
                  <TableHead>{t('Review Status')}</TableHead>
                  <TableHead>{t('Transfer Time')}</TableHead>
                  <TableHead>{t('Proof')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {transfers.length === 0 ? (
                  <EmptyState message={t('No data')} />
                ) : (
                  transfers.map((record) => (
                    <TableRow key={record.id}>
                      <TableCell>{record.bank_account_name || '-'}</TableCell>
                      <TableCell>{formatDisplayMoney(record.transfer_amount)}</TableCell>
                      <TableCell>
                        <StatusBadge status={record.review_status} />
                      </TableCell>
                      <TableCell>{formatDisplayDateTime(record.transfer_time)}</TableCell>
                      <TableCell>
                        {record.proof_url ? (
                          <Button
                            size='sm'
                            variant='outline'
                            render={<a href={record.proof_url} target='_blank' rel='noreferrer' />}
                          >
                            {t('View')}
                          </Button>
                        ) : (
                          '-'
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function RevenueShareDashboard() {
  const { t } = useTranslation()
  const summaryQuery = useQuery({
    queryKey: ['commercial', 'revenue-share-dashboard', 'summary'],
    queryFn: () => getFinanceSummary({ days: 30 }),
    retry: false,
  })
  const summary = summaryQuery.data?.data
  const currency = summary?.revenue_share.currency
  const channels = summary?.revenue_share.top_channels ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Revenue Share Dashboard')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Review channel revenue share foundation metrics.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='grid gap-4 md:grid-cols-4'>
          <MetricCard
            title={t('Channel Revenue')}
            value={formatDisplayMoney(summary?.revenue_share.gross_amount, currency)}
            description={t('Gross channel revenue')}
            icon={Landmark}
          />
          <MetricCard
            title={t('Distributor Revenue')}
            value={formatDisplayMoney(summary?.revenue_share.distributor_amount, currency)}
            description={t('Distributor share foundation amount')}
            icon={BadgeDollarSign}
          />
          <MetricCard
            title={t('Pending Settlement')}
            value={t('Foundation Preview')}
            description={t('Settlement workflow is planned after revenue share foundation')}
            icon={Timer}
          />
          <MetricCard
            title={t('Settled Amount')}
            value={t('Foundation Preview')}
            description={t('Settlement records are not enabled in this iteration')}
            icon={CreditCard}
          />
        </div>
        <Card>
          <CardHeader>
            <CardTitle>{t('Top Channels')}</CardTitle>
            <CardDescription>{t('Revenue share ranking by channel')}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-border rounded-lg border'>
              {channels.length === 0 ? (
                <div className='text-muted-foreground px-4 py-8 text-center text-sm'>
                  {t('No data')}
                </div>
              ) : (
                channels.map((channel) => (
                  <div
                    key={channel.distribution_channel_id}
                    className='flex items-center justify-between gap-4 px-4 py-3 text-sm'
                  >
                    <div className='min-w-0'>
                      <div className='truncate font-medium'>
                        {channel.name || `#${channel.distribution_channel_id}`}
                      </div>
                      <div className='text-muted-foreground text-xs'>
                        {formatDisplayNumber(channel.record_count)} {t('Records')}
                      </div>
                    </div>
                    <Badge variant='outline'>
                      {formatDisplayMoney(channel.gross_amount, currency)}
                    </Badge>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
        <FoundationPreview
          title={t('Settlement Boundary')}
          description={t(
            'Revenue share foundation is visible here. Real settlement remains a later iteration.'
          )}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function CommercialGateway() {
  const { t } = useTranslation()
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Commercial Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Choose a commercial dashboard to inspect tenant operation metrics.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
          {[
            { title: t('Quota Dashboard'), to: '/quota-dashboard' },
            { title: t('Usage Analytics'), to: '/usage-analytics' },
            { title: t('Billing Dashboard'), to: '/billing-dashboard' },
            { title: t('Payment Center'), to: '/payment-center' },
            { title: t('Revenue Share Dashboard'), to: '/revenue-share' },
          ].map((item) => (
            <Card key={item.to}>
              <CardHeader>
                <CardTitle>{item.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <Button render={<Link to={item.to} />}>{t('Open')}</Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
