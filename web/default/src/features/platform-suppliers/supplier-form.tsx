import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import type { Supplier, SupplierData } from './api'

export type FormKind =
  'supplier' | 'account' | 'price' | 'statement' | 'review' | 'payment'
export type SupplierFormState = {
  kind: FormKind
  supplier?: Supplier
  seed?: Record<string, string>
  statementId?: number
}
type Choice = { value: string; label: string }
type Field = { key: string; label: string; type?: string; choices?: Choice[] }
const option = (value: string, label = value): Choice => ({ value, label })
export function SupplierForm(props: {
  form: SupplierFormState
  data: SupplierData
  busy: boolean
  onClose: () => void
  onSubmit: (path: string, body: unknown) => Promise<void>
}) {
  const { t } = useTranslation()
  const initial = props.form.supplier
  const [values, setValues] = useState<Record<string, string>>(() => ({
    provider: initial?.provider || props.data.providers[0]?.id || '',
    name: initial?.name || '',
    legal_entity: initial?.legal_entity || '',
    contact: initial?.contact || '',
    mode: initial?.mode || 'integration',
    status: initial?.status || 'draft',
    currency: initial?.currency || 'CNY',
    agreement_ref: initial?.agreement_ref || '',
    agreement_expires_at: initial?.agreement_expires_at
      ? new Date(
          initial.agreement_expires_at * 1000 -
            new Date(initial.agreement_expires_at * 1000).getTimezoneOffset() *
              60000
        )
          .toISOString()
          .slice(0, 16)
      : '',
    supplier_id: String(props.data.suppliers[0]?.id || ''),
    account_id: String(props.data.accounts[0]?.id || ''),
    unit: 'million_tokens',
    unit_price: '0',
    input: '0',
    output: '0',
    cache_read: '0',
    cache_write: '0',
    amount: '',
    review: 'false',
    ...props.form.seed,
  }))
  const supplierChoices = props.data.suppliers.map((s) =>
    option(String(s.id), s.name)
  )
  const accountChoices = props.data.accounts.map((a) =>
    option(String(a.id), a.name)
  )
  const fields: Record<FormKind, Field[]> = {
    supplier: [
      {
        key: 'provider',
        label: 'Model provider',
        choices: props.data.providers.map((p) => option(p.id, p.name)),
      },
      { key: 'name', label: 'Supplier name' },
      { key: 'legal_entity', label: 'Contracting legal entity' },
      { key: 'contact', label: 'Business contact' },
      {
        key: 'mode',
        label: 'Business model',
        choices: [
          option('integration', t('Application integration')),
          option('resale', t('Authorized resale')),
        ],
      },
      {
        key: 'currency',
        label: 'Procurement currency',
        choices: [option('CNY'), option('USD')],
      },
      { key: 'agreement_ref', label: 'Contract and authorization reference' },
      {
        key: 'agreement_expires_at',
        label: 'Authorization expiry',
        type: 'datetime-local',
      },
      {
        key: 'status',
        label: 'Supplier status',
        choices: [
          option('draft', t('Draft')),
          option('active', t('Active')),
          option('suspended', t('Suspended')),
        ],
      },
      {
        key: 'review',
        label: 'I reviewed the contract scope and API product authorization',
        choices: [
          option('false', t('Not reviewed')),
          option('true', t('Reviewed and confirmed')),
        ],
      },
    ],
    account: [
      { key: 'supplier_id', label: 'Supplier', choices: supplierChoices },
      { key: 'name', label: 'Upstream account name' },
      { key: 'channel_id', label: 'Existing channel ID', type: 'number' },
      { key: 'account_ref', label: 'Upstream account or workspace ID' },
      { key: 'key_ref', label: 'API Key resource ID (not the secret)' },
      { key: 'region', label: 'Service region' },
    ],
    price: [
      {
        key: 'unit',
        label: 'Procurement metering unit',
        choices: [
          'million_tokens',
          'second',
          'image',
          'request',
          'million_characters',
        ].map((value) => option(value, t(value))),
      },
      { key: 'unit_price', label: 'Price per selected unit' },
      { key: 'account_id', label: 'Upstream account', choices: accountChoices },
      { key: 'model', label: 'Upstream model ID' },
      {
        key: 'effective_at',
        label: 'Price effective date',
        type: 'datetime-local',
      },
      { key: 'input', label: 'Input price per million tokens' },
      { key: 'output', label: 'Output price per million tokens' },
      { key: 'cache_read', label: 'Cache read price per million tokens' },
      { key: 'cache_write', label: 'Cache write price per million tokens' },
    ],
    statement: [
      { key: 'account_id', label: 'Upstream account', choices: accountChoices },
      { key: 'external_ref', label: 'Provider statement reference' },
      { key: 'invoice_ref', label: 'Supplier invoice reference' },
      {
        key: 'start_at',
        label: 'Period start (inclusive)',
        type: 'datetime-local',
      },
      {
        key: 'end_at',
        label: 'Period end (exclusive)',
        type: 'datetime-local',
      },
      { key: 'amount', label: 'Provider billed amount' },
    ],
    review: [
      {
        key: 'note',
        label: 'Reconciliation evidence and discrepancy resolution',
      },
    ],
    payment: [
      { key: 'reference', label: 'Actual bank payment reference' },
      { key: 'amount', label: 'Actual amount paid' },
    ],
  }
  const titles: Record<FormKind, string> = {
    supplier: 'Supplier contract',
    account: 'Bind upstream account',
    price: 'Record procurement price',
    statement: 'Register provider statement',
    review: 'Confirm reconciliation',
    payment: 'Record external payment',
  }
  async function submit(event: FormEvent) {
    event.preventDefault()
    const epoch = (key: string) =>
      values[key] ? Math.floor(new Date(values[key]).getTime() / 1000) : 0
    let path = ''
    let body: unknown
    switch (props.form.kind) {
      case 'supplier':
        body = {
          supplier: {
            id: initial?.id || 0,
            version: initial?.version || 0,
            provider: values.provider,
            name: values.name,
            legal_entity: values.legal_entity,
            contact: values.contact,
            mode: values.mode,
            status: values.status,
            currency: values.currency,
            agreement_ref: values.agreement_ref,
            agreement_expires_at: epoch('agreement_expires_at'),
          },
          review: values.review === 'true',
        }
        break
      case 'account':
        path = '/accounts'
        body = {
          supplier_id: Number(values.supplier_id),
          channel_id: Number(values.channel_id),
          name: values.name,
          account_ref: values.account_ref,
          key_ref: values.key_ref,
          region: values.region,
        }
        break
      case 'price':
        path = '/prices'
        body = {
          account_id: Number(values.account_id),
          unit: values.unit,
          unit_price: values.unit_price,
          model: values.model,
          effective_at: epoch('effective_at'),
          input: values.input,
          output: values.output,
          cache_read: values.cache_read,
          cache_write: values.cache_write,
        }
        break
      case 'statement':
        path = '/statements'
        body = {
          account_id: Number(values.account_id),
          external_ref: values.external_ref,
          invoice_ref: values.invoice_ref,
          start_at: epoch('start_at'),
          end_at: epoch('end_at'),
          amount: values.amount,
        }
        break
      case 'review':
        path = `/statements/${props.form.statementId}/review`
        body = { note: values.note }
        break
      case 'payment':
        path = '/payments'
        body = {
          statement_id: props.form.statementId,
          reference: values.reference,
          amount: values.amount,
        }
        break
    }
    await props.onSubmit(path, body)
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !props.busy) props.onClose()
      }}
    >
      <DialogContent className='max-h-[90svh] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>{t(titles[props.form.kind])}</DialogTitle>
        </DialogHeader>
        <form onSubmit={submit} className='space-y-4'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'All changes require MFA verification. Record actual contracts and payments; this form does not purchase resources or transfer money.'
            )}
          </p>
          {fields[props.form.kind].map((field) => (
            <div key={field.key} className='space-y-1.5'>
              <Label htmlFor={'supplier-' + field.key}>{t(field.label)}</Label>
              {field.choices ? (
                <NativeSelect
                  id={'supplier-' + field.key}
                  value={values[field.key] || ''}
                  onChange={(e) =>
                    setValues((v) => ({ ...v, [field.key]: e.target.value }))
                  }
                >
                  <NativeSelectOption value='' disabled>
                    {t('Select')}
                  </NativeSelectOption>
                  {field.choices.map((c) => (
                    <NativeSelectOption key={c.value} value={c.value}>
                      {c.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              ) : (
                <Input
                  id={'supplier-' + field.key}
                  type={field.type || 'text'}
                  value={values[field.key] || ''}
                  maxLength={field.key === 'note' ? 500 : 200}
                  onChange={(e) =>
                    setValues((v) => ({ ...v, [field.key]: e.target.value }))
                  }
                />
              )}
            </div>
          ))}
          {props.form.kind === 'price' && (
            <p className='text-muted-foreground text-sm'>
              {t(
                'Prices use the supplier currency and selected metering unit, with up to six decimal places. Token rates use input, output and cache prices; other units use unit price. Historical prices are retained and customer selling prices are unchanged.'
              )}
            </p>
          )}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              disabled={props.busy}
              onClick={props.onClose}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={props.busy}>
              {t('Save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
