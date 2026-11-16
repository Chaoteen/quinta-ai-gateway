import { useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'
import { getSuppliers, mutateSupplier } from './api'
import { SupplierForm, type SupplierFormState } from './supplier-form'

function Records(props: { headings: string[]; rows: ReactNode[][] }) {
  const { t } = useTranslation()
  return (
    <div className='overflow-x-auto rounded-lg border'>
      <Table>
        <TableHeader>
          <TableRow>
            {props.headings.map((h) => (
              <TableHead key={h}>{t(h)}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.rows.map((row, i) => (
            <TableRow key={i}>
              {row.map((cell, j) => (
                <TableCell
                  key={j}
                  className='max-w-sm break-words whitespace-normal'
                >
                  {cell}
                </TableCell>
              ))}
            </TableRow>
          ))}
          {props.rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={props.headings.length}>
                {t('No records yet')}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
export function PlatformSuppliers() {
  const { t } = useTranslation()
  const cache = useQueryClient()
  const query = useQuery({
    queryKey: ['platform-suppliers'],
    queryFn: getSuppliers,
  })
  const [form, setForm] = useState<SupplierFormState | null>(null)
  const [busy, setBusy] = useState(false)
  const verification = useSecureVerification()
  const data = query.data
  async function write(
    path: string,
    body: unknown,
    method: 'post' | 'patch' = 'post'
  ) {
    setBusy(true)
    try {
      await verification.withVerification(async () => {
        await mutateSupplier(path, body, method)
        await cache.invalidateQueries({ queryKey: ['platform-suppliers'] })
        setForm(null)
        toast.success(t('Supplier record saved'))
      })
    } catch (error: unknown) {
      const e = error as {
        response?: { data?: { message?: string } }
        message?: string
      }
      toast.error(
        e.response?.data?.message || e.message || t('Operation failed')
      )
    } finally {
      setBusy(false)
    }
  }
  const date = (stamp: number) =>
    stamp ? new Date(stamp * 1000).toLocaleString() : '—'
  const money = (minor: number, currency: string) =>
    `${currency} ${(minor / 100).toFixed(2)}`
  const accountName = (id: number) =>
    data?.accounts.find((a) => a.id === id)?.name || String(id)
  function statementActions(s: import('./api').Statement) {
    if (s.status === 'pending_review')
      return (
        <Button
          size='sm'
          variant='outline'
          onClick={() => setForm({ kind: 'review', statementId: s.id })}
        >
          {t('Confirm reconciliation')}
        </Button>
      )
    if (s.status === 'reviewed')
      return (
        <Button
          size='sm'
          variant='outline'
          onClick={() =>
            setForm({
              kind: 'payment',
              statementId: s.id,
              seed: {
                amount: ((s.amount_minor - s.paid_minor) / 100).toFixed(2),
              },
            })
          }
        >
          {t('Record external payment')}
        </Button>
      )
    return null
  }
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Supplier management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' onClick={() => query.refetch()}>
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-5'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Manage official API procurement, contracts, account bindings, cost prices and manual reconciliation. API access does not automatically grant resale rights.'
            )}
          </p>
          <p className='rounded-lg border p-3 text-sm'>
            {t(
              'Enable MFA in your profile before making changes. API secrets stay in Channels; enter only account IDs and Key resource IDs here. Payments are external bank records, not automatic transfers.'
            )}{' '}
            <Link to='/profile' className='underline'>
              {t('Security settings')}
            </Link>
          </p>
          {query.isPending && <p>{t('Loading…')}</p>}
          {query.isError && (
            <p role='alert'>
              {t(
                'Could not load suppliers. Retry or check administrator permissions.'
              )}
            </p>
          )}
          {data && (
            <>
              <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
                {data.providers.map((provider) => (
                  <section
                    key={provider.id}
                    className='space-y-2 rounded-lg border p-4'
                  >
                    <h3 className='font-semibold'>{provider.name}</h3>
                    <p className='text-muted-foreground text-sm'>
                      {provider.guidance}
                    </p>
                    <p className='text-xs break-all'>
                      {t('Channel base address')}: {provider.endpoint}
                    </p>
                    <div className='flex flex-wrap gap-3 text-sm'>
                      <a
                        href={provider.docs}
                        target='_blank'
                        rel='noopener noreferrer'
                        className='underline'
                      >
                        {t('Official documentation')}
                      </a>
                      <a
                        href={provider.partnership}
                        target='_blank'
                        rel='noopener noreferrer'
                        className='underline'
                      >
                        {t('Official business contact')}
                      </a>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() =>
                          setForm({
                            kind: 'supplier',
                            seed: {
                              provider: provider.id,
                              name: provider.name,
                            },
                          })
                        }
                      >
                        {t('Register supplier')}
                      </Button>
                    </div>
                  </section>
                ))}
              </div>
              <Tabs defaultValue='suppliers'>
                <TabsList className='h-auto flex-wrap'>
                  <TabsTrigger value='suppliers'>{t('Contracts')}</TabsTrigger>
                  <TabsTrigger value='accounts'>
                    {t('Upstream accounts')}
                  </TabsTrigger>
                  <TabsTrigger value='prices'>
                    {t('Procurement prices')}
                  </TabsTrigger>
                  <TabsTrigger value='statements'>
                    {t('Statements and settlement')}
                  </TabsTrigger>
                  <TabsTrigger value='audit'>{t('Audit trail')}</TabsTrigger>
                </TabsList>
                <TabsContent value='suppliers'>
                  <Records
                    headings={[
                      'Supplier',
                      'Business model',
                      'Contract and authorization reference',
                      'Authorization expiry',
                      'Status',
                      'Actions',
                    ]}
                    rows={data.suppliers.map((s) => [
                      <div>
                        {s.name}
                        <div className='text-muted-foreground text-xs'>
                          {s.legal_entity}
                        </div>
                      </div>,
                      t(s.mode),
                      s.agreement_ref,
                      date(s.agreement_expires_at),
                      t(s.status),
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() =>
                          setForm({ kind: 'supplier', supplier: s })
                        }
                      >
                        {t('Edit')}
                      </Button>,
                    ])}
                  />
                </TabsContent>
                <TabsContent value='accounts' className='space-y-3'>
                  <div className='flex flex-wrap gap-3'>
                    <Button
                      onClick={() => setForm({ kind: 'account' })}
                      disabled={
                        !data.suppliers.some((s) => s.status === 'active')
                      }
                    >
                      {t('Bind upstream account')}
                    </Button>
                    <Link to='/channels' className='self-center underline'>
                      {t('Create or manage channels')}
                    </Link>
                  </div>
                  <Records
                    headings={[
                      'Upstream account',
                      'Supplier',
                      'Channel',
                      'Region',
                      'Key resource ID',
                      'Status',
                      'Actions',
                    ]}
                    rows={data.accounts.map((a) => [
                      a.name,
                      data.suppliers.find((s) => s.id === a.supplier_id)?.name,
                      a.channel_id,
                      a.region,
                      a.key_ref,
                      t(a.status),
                      <Button
                        variant='outline'
                        size='sm'
                        disabled={busy}
                        onClick={() =>
                          write(
                            `/accounts/${a.id}/status`,
                            {
                              status:
                                a.status === 'active' ? 'paused' : 'active',
                            },
                            'patch'
                          )
                        }
                      >
                        {t(a.status === 'active' ? 'Pause' : 'Resume')}
                      </Button>,
                    ])}
                  />
                </TabsContent>
                <TabsContent value='prices' className='space-y-3'>
                  <Button
                    onClick={() => setForm({ kind: 'price' })}
                    disabled={!data.accounts.length}
                  >
                    {t('Record procurement price')}
                  </Button>
                  <Records
                    headings={[
                      'Upstream account',
                      'Model',
                      'Price effective date',
                      'Procurement currency',
                      'Procurement metering unit',
                      'Price per selected unit',
                      'Input / output per million',
                      'Cache read / write per million',
                    ]}
                    rows={data.prices.map((p) => [
                      accountName(p.account_id),
                      p.model,
                      date(p.effective_at),
                      data.suppliers.find(
                        (s) =>
                          s.id ===
                          data.accounts.find((a) => a.id === p.account_id)
                            ?.supplier_id
                      )?.currency,
                      t(p.unit),
                      p.unit_price_micros / 1e6,
                      `${p.input_micros / 1e6} / ${p.output_micros / 1e6}`,
                      `${p.cache_read_micros / 1e6} / ${p.cache_write_micros / 1e6}`,
                    ])}
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Prices use the supplier currency and selected metering unit, with up to six decimal places. Token rates use input, output and cache prices; other units use unit price. Historical prices are retained and customer selling prices are unchanged.'
                    )}
                  </p>
                </TabsContent>
                <TabsContent value='statements' className='space-y-3'>
                  <Button
                    onClick={() => setForm({ kind: 'statement' })}
                    disabled={!data.accounts.length}
                  >
                    {t('Register provider statement')}
                  </Button>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Usage is a dated platform snapshot. Compare provider bills and resolve differences manually before recording payment; snapshots are not proof of the supplier charge.'
                    )}
                  </p>
                  <Records
                    headings={[
                      'Statement',
                      'Period',
                      'Platform request / input / output',
                      'Billed / paid',
                      'Status',
                      'Actions',
                    ]}
                    rows={data.statements.map((s) => [
                      <div>
                        {s.external_ref}
                        <div className='text-muted-foreground text-xs'>
                          {accountName(s.account_id)} · {s.invoice_ref}
                        </div>
                      </div>,
                      `${date(s.start_at)} — ${date(s.end_at)}`,
                      `${s.platform_requests} / ${s.platform_input} / ${s.platform_output}`,
                      `${money(s.amount_minor, s.currency)} / ${money(s.paid_minor, s.currency)}`,
                      <div>
                        {t(s.status)}
                        <div className='text-muted-foreground text-xs'>
                          {s.review_note}
                        </div>
                      </div>,
                      statementActions(s),
                    ])}
                  />
                  <h3 className='font-semibold'>{t('Payment records')}</h3>
                  <Records
                    headings={[
                      'Reference',
                      'Statement ID',
                      'Amount',
                      'Created at',
                    ]}
                    rows={data.payments.map((p) => [
                      p.reference,
                      p.statement_id,
                      money(
                        p.amount_minor,
                        data.statements.find((s) => s.id === p.statement_id)
                          ?.currency || ''
                      ),
                      date(p.created_at),
                    ])}
                  />
                </TabsContent>
                <TabsContent value='audit' className='space-y-3'>
                  <h3 className='font-semibold'>
                    {t('Contract revision history')}
                  </h3>
                  <Records
                    headings={[
                      'Supplier',
                      'Version',
                      'Contract and authorization reference',
                      'Authorization expiry',
                      'Status',
                      'Operator ID',
                    ]}
                    rows={(data.revisions || []).map((r) => [
                      r.name,
                      r.version,
                      r.agreement_ref,
                      date(r.agreement_expires_at),
                      t(r.status),
                      r.actor_id,
                    ])}
                  />
                  <h3 className='font-semibold'>{t('Audit trail')}</h3>
                  <Records
                    headings={[
                      'Created at',
                      'Operator ID',
                      'Action',
                      'Resource ID',
                    ]}
                    rows={data.audit.map((a) => [
                      date(a.created_at),
                      a.actor_id,
                      a.action,
                      a.resource_id,
                    ])}
                  />
                </TabsContent>
              </Tabs>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Showing the latest 500 records per category. No supplier account is opened automatically; real API credentials and contract scope require your company to complete onboarding.'
                )}
              </p>
            </>
          )}
        </div>
        {form && data && (
          <SupplierForm
            key={JSON.stringify(form)}
            form={form}
            data={data}
            busy={busy}
            onClose={() => setForm(null)}
            onSubmit={write}
          />
        )}
        <SecureVerificationDialog
          open={verification.open}
          onOpenChange={(open) => {
            if (!open) verification.cancel()
          }}
          methods={verification.methods}
          state={verification.state}
          onVerify={async (method, code) => {
            await verification.executeVerification(method, code)
          }}
          onCancel={verification.cancel}
          onCodeChange={verification.setCode}
          onMethodChange={verification.switchMethod}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
