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
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from 'react'
import { Pencil, Plus, Power, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatTimestamp } from '@/lib/format'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EmptyState } from '@/components/empty-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import {
  createAdminConsoleResource,
  getReadonlyResource,
  updateAdminConsoleResource,
  updateAdminConsoleResourceStatus,
} from './api'
import type {
  AdminConsoleMutationPayload,
  OrganizationRecord,
  ReadonlyRecord,
  ReadonlyResource,
  TenantRecord,
} from './types'

type Column = {
  key: string
  title: string
  render?: (record: ReadonlyRecord) => ReactNode
}

type DialogMode = 'create' | 'edit'

type FormState = {
  name: string
  tenant_id: string
  organization_id: string
  code: string
  status: string
}

type ParentOption = {
  id: number
  name: string
}

const RESOURCE_TITLES: Record<ReadonlyResource, string> = {
  tenants: 'Tenants',
  organizations: 'Organizations',
  departments: 'Departments',
  distribution_channels: 'Distribution Channels',
}

function numberValue(record: ReadonlyRecord, key: string): number {
  const value = (record as unknown as Record<string, unknown>)[key]
  return typeof value === 'number' ? value : 0
}

function textValue(record: ReadonlyRecord, key: string): string {
  const value = (record as unknown as Record<string, unknown>)[key]
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  return '-'
}

function useColumns(resource: ReadonlyResource): Column[] {
  const { t } = useTranslation()
  return useMemo(() => {
    const statusColumn: Column = {
      key: 'status',
      title: t('Status'),
      render: (record) => {
        const enabled = numberValue(record, 'status') === 1
        return (
          <StatusBadge
            label={enabled ? t('Enabled') : t('Disabled')}
            variant={enabled ? 'success' : 'neutral'}
            copyable={false}
          />
        )
      },
    }
    const createdAtColumn: Column = {
      key: 'created_at',
      title: t('Created At'),
      render: (record) => formatTimestamp(numberValue(record, 'created_at')),
    }
    const common: Column[] = [
      { key: 'id', title: 'ID' },
      { key: 'name', title: t('Name') },
      statusColumn,
    ]

    switch (resource) {
      case 'tenants':
        return [...common, createdAtColumn]
      case 'organizations':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'tenant_id', title: 'tenant_id' },
          statusColumn,
          createdAtColumn,
        ]
      case 'departments':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'tenant_id', title: 'tenant_id' },
          { key: 'organization_id', title: 'organization_id' },
          statusColumn,
          createdAtColumn,
        ]
      case 'distribution_channels':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'code', title: t('Code') },
          { key: 'tenant_id', title: 'tenant_id' },
          statusColumn,
          createdAtColumn,
        ]
    }
  }, [resource, t])
}

function getInitialFormState(record?: ReadonlyRecord): FormState {
  const values = (record || {}) as Record<string, unknown>
  return {
    name: typeof values.name === 'string' ? values.name : '',
    tenant_id:
      typeof values.tenant_id === 'number' ? String(values.tenant_id) : '',
    organization_id:
      typeof values.organization_id === 'number'
        ? String(values.organization_id)
        : '',
    code: typeof values.code === 'string' ? values.code : '',
    status: typeof values.status === 'number' ? String(values.status) : '1',
  }
}

function toInt(value: string): number {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : 0
}

function buildMutationPayload(
  resource: ReadonlyResource,
  form: FormState
): AdminConsoleMutationPayload {
  const payload: AdminConsoleMutationPayload = {
    name: form.name.trim(),
    status: toInt(form.status),
  }
  if (resource === 'organizations' || resource === 'departments') {
    payload.tenant_id = toInt(form.tenant_id)
  }
  if (resource === 'departments') {
    payload.organization_id = toInt(form.organization_id)
  }
  if (resource === 'distribution_channels') {
    payload.tenant_id = toInt(form.tenant_id)
    payload.code = form.code.trim()
  }
  return payload
}

function ParentSearchSelector(props: {
  label: string
  value: string
  search: string
  options: ParentOption[]
  loading: boolean
  disabled?: boolean
  placeholder: string
  currentLabel?: string
  onSearchChange: (value: string) => void
  onSearch: () => void
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const hasSelectedOption = props.options.some(
    (option) => String(option.id) === props.value
  )

  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      <div className='flex gap-2'>
        <Input
          value={props.search}
          onChange={(event) => props.onSearchChange(event.target.value)}
          placeholder={props.placeholder}
          disabled={props.disabled || props.loading}
        />
        <Button
          type='button'
          variant='outline'
          onClick={props.onSearch}
          disabled={props.disabled || props.loading}
        >
          {props.loading ? t('Loading...') : t('Search')}
        </Button>
      </div>
      <NativeSelect
        className='w-full'
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        disabled={props.disabled || props.loading}
        required
      >
        <NativeSelectOption value=''>{t('Select')}</NativeSelectOption>
        {props.value && !hasSelectedOption && (
          <NativeSelectOption value={props.value}>
            {props.currentLabel || t('Current Selection')}
          </NativeSelectOption>
        )}
        {props.options.map((option) => (
          <NativeSelectOption key={option.id} value={String(option.id)}>
            {option.name}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </div>
  )
}

function AdminConsoleMutationDialog(props: {
  open: boolean
  mode: DialogMode
  resource: ReadonlyResource
  record?: ReadonlyRecord
  onOpenChange: (open: boolean) => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const [form, setForm] = useState<FormState>(() =>
    getInitialFormState(props.record)
  )
  const [tenantSearch, setTenantSearch] = useState('')
  const [organizationSearch, setOrganizationSearch] = useState('')
  const [tenantOptions, setTenantOptions] = useState<TenantRecord[]>([])
  const [organizationOptions, setOrganizationOptions] = useState<
    OrganizationRecord[]
  >([])
  const [tenantLoading, setTenantLoading] = useState(false)
  const [organizationLoading, setOrganizationLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const isEdit = props.mode === 'edit'
  const needsTenants =
    props.resource === 'organizations' ||
    props.resource === 'departments' ||
    props.resource === 'distribution_channels'

  useEffect(() => {
    if (props.open) {
      setForm(getInitialFormState(props.record))
      setTenantSearch('')
      setOrganizationSearch('')
      setTenantOptions([])
      setOrganizationOptions([])
    }
  }, [props.open, props.record])

  useEffect(() => {
    if (!props.open || !needsTenants) return

    let cancelled = false
    setTenantLoading(true)
    getReadonlyResource('tenants', { page: 1, limit: 50 })
      .then((tenantRes) => {
        if (cancelled) return
        setTenantOptions(tenantRes.items || [])
      })
      .catch(() => {
        if (cancelled) return
        toast.error(t('Failed to load data'))
        setTenantOptions([])
      })
      .finally(() => {
        if (cancelled) return
        setTenantLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [props.open, needsTenants, t])

  const updateField = (field: keyof FormState, value: string) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const updateTenant = (tenantId: string) => {
    setForm((current) => ({
      ...current,
      tenant_id: tenantId,
      organization_id: '',
    }))
    setOrganizationSearch('')
    setOrganizationOptions([])
    if (props.resource === 'departments' && tenantId) {
      void searchOrganizations('', toInt(tenantId))
    }
  }

  const selectedTenantId = toInt(form.tenant_id)

  const searchTenants = async (query = tenantSearch) => {
    setTenantLoading(true)
    try {
      const res = await getReadonlyResource('tenants', {
        page: 1,
        limit: 50,
        q: query,
      })
      setTenantOptions(res.items || [])
    } catch {
      toast.error(t('Failed to load data'))
      setTenantOptions([])
    } finally {
      setTenantLoading(false)
    }
  }

  const searchOrganizations = async (
    query = organizationSearch,
    tenantId = selectedTenantId
  ) => {
    if (!tenantId) {
      setOrganizationOptions([])
      return
    }
    setOrganizationLoading(true)
    try {
      const res = await getReadonlyResource('organizations', {
        page: 1,
        limit: 50,
        q: query,
        tenant_id: tenantId,
      })
      setOrganizationOptions(res.items || [])
    } catch {
      toast.error(t('Failed to load data'))
      setOrganizationOptions([])
    } finally {
      setOrganizationLoading(false)
    }
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    try {
      const payload = buildMutationPayload(props.resource, form)
      const result =
        isEdit && props.record
          ? await updateAdminConsoleResource(
              props.resource,
              props.record.id,
              payload
            )
          : await createAdminConsoleResource(props.resource, payload)

      if (result.success) {
        toast.success(
          isEdit ? t('Updated successfully') : t('Created successfully')
        )
        props.onOpenChange(false)
        props.onSaved()
      } else {
        toast.error(result.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t('Edit') : t('Create')}{' '}
            {t(RESOURCE_TITLES[props.resource])}
          </DialogTitle>
        </DialogHeader>
        <form
          id='admin-console-form'
          className='space-y-4'
          onSubmit={handleSubmit}
        >
          <div className='space-y-2'>
            <Label htmlFor='admin-console-name'>{t('Name')}</Label>
            <Input
              id='admin-console-name'
              value={form.name}
              onChange={(event) => updateField('name', event.target.value)}
              required
            />
          </div>
          {(props.resource === 'organizations' ||
            props.resource === 'departments' ||
            props.resource === 'distribution_channels') && (
            <ParentSearchSelector
              label={t('Tenant')}
              value={form.tenant_id}
              search={tenantSearch}
              options={tenantOptions}
              loading={tenantLoading}
              placeholder={t('Search tenants')}
              currentLabel={t('Current Tenant')}
              onSearchChange={setTenantSearch}
              onSearch={() => searchTenants()}
              onChange={updateTenant}
            />
          )}
          {props.resource === 'departments' && (
            <ParentSearchSelector
              label={t('Organization')}
              value={form.organization_id}
              search={organizationSearch}
              options={organizationOptions}
              loading={organizationLoading}
              disabled={!form.tenant_id}
              placeholder={
                form.tenant_id
                  ? t('Search organizations')
                  : t('Select Tenant First')
              }
              currentLabel={t('Current Organization')}
              onSearchChange={setOrganizationSearch}
              onSearch={() => searchOrganizations()}
              onChange={(value) => updateField('organization_id', value)}
            />
          )}
          {props.resource === 'distribution_channels' && (
            <div className='space-y-2'>
              <Label htmlFor='admin-console-code'>{t('Code')}</Label>
              <Input
                id='admin-console-code'
                value={form.code}
                onChange={(event) => updateField('code', event.target.value)}
                required
              />
            </div>
          )}
          <div className='space-y-2'>
            <Label htmlFor='admin-console-status'>{t('Status')}</Label>
            <NativeSelect
              id='admin-console-status'
              className='w-full'
              value={form.status}
              onChange={(event) => updateField('status', event.target.value)}
            >
              <NativeSelectOption value='1'>{t('Enabled')}</NativeSelectOption>
              <NativeSelectOption value='2'>{t('Disabled')}</NativeSelectOption>
            </NativeSelect>
          </div>
        </form>
        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button form='admin-console-form' type='submit' disabled={submitting}>
            {submitting ? t('Saving...') : t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function ReadonlyManagementPage(props: { resource: ReadonlyResource }) {
  const { t } = useTranslation()
  const [items, setItems] = useState<ReadonlyRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogMode, setDialogMode] = useState<DialogMode>('create')
  const [currentRecord, setCurrentRecord] = useState<ReadonlyRecord>()
  const [statusUpdatingId, setStatusUpdatingId] = useState<number | null>(null)
  const columns = useColumns(props.resource)
  const title = t(RESOURCE_TITLES[props.resource])

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getReadonlyResource(props.resource)
      setItems(res.items || [])
    } catch {
      toast.error(t('Failed to load data'))
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [props.resource, t])

  useEffect(() => {
    loadData()
  }, [loadData])

  const openCreateDialog = () => {
    setCurrentRecord(undefined)
    setDialogMode('create')
    setDialogOpen(true)
  }

  const openEditDialog = (record: ReadonlyRecord) => {
    setCurrentRecord(record)
    setDialogMode('edit')
    setDialogOpen(true)
  }

  const toggleStatus = async (record: ReadonlyRecord) => {
    const nextStatus = numberValue(record, 'status') === 1 ? 2 : 1
    setStatusUpdatingId(record.id)
    try {
      const result = await updateAdminConsoleResourceStatus(
        props.resource,
        record.id,
        nextStatus
      )
      if (result.success) {
        toast.success(t('Updated successfully'))
        loadData()
      } else {
        toast.error(result.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setStatusUpdatingId(null)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{title}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Read-only admin console view')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Button variant='default' size='sm' onClick={openCreateDialog}>
          <Plus className='h-4 w-4' />
          {t('Create')}
        </Button>
        <Button
          variant='outline'
          size='sm'
          onClick={loadData}
          disabled={loading}
        >
          <RefreshCw className='h-4 w-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {loading ? (
          <LoadingState />
        ) : items.length === 0 ? (
          <EmptyState title={t('No Data')} />
        ) : (
          <div className='border-border overflow-x-auto rounded-md border'>
            <Table>
              <TableHeader>
                <TableRow>
                  {columns.map((column) => (
                    <TableHead key={column.key}>{column.title}</TableHead>
                  ))}
                  <TableHead>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={`${props.resource}-${item.id}`}>
                    {columns.map((column) => (
                      <TableCell key={column.key} className='whitespace-nowrap'>
                        {column.render
                          ? column.render(item)
                          : textValue(item, column.key)}
                      </TableCell>
                    ))}
                    <TableCell className='whitespace-nowrap'>
                      <div className='flex items-center gap-2'>
                        <Button
                          variant='outline'
                          size='icon-sm'
                          onClick={() => openEditDialog(item)}
                          title={t('Edit')}
                        >
                          <Pencil className='h-4 w-4' />
                          <span className='sr-only'>{t('Edit')}</span>
                        </Button>
                        <Button
                          variant='outline'
                          size='icon-sm'
                          onClick={() => toggleStatus(item)}
                          disabled={statusUpdatingId === item.id}
                          title={
                            numberValue(item, 'status') === 1
                              ? t('Disable')
                              : t('Enable')
                          }
                        >
                          <Power className='h-4 w-4' />
                          <span className='sr-only'>
                            {numberValue(item, 'status') === 1
                              ? t('Disable')
                              : t('Enable')}
                          </span>
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </SectionPageLayout.Content>
      <AdminConsoleMutationDialog
        open={dialogOpen}
        mode={dialogMode}
        resource={props.resource}
        record={currentRecord}
        onOpenChange={setDialogOpen}
        onSaved={loadData}
      />
    </SectionPageLayout>
  )
}
