import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { scopedRoleSchema, type ScopedRoleValues } from '../lib/schema'
import type { Policy, Role, WriteRoleConfiguration } from '../types'
import { GrantsEditor } from './grants-editor'

const actions = [
  'view',
  'use',
  'create',
  'operate',
  'update',
  'delete',
] as const
export function ScopedRoleEditor(props: {
  policy: Policy
  roles: Role[]
  tenantId: number
  busy: boolean
  write: WriteRoleConfiguration
}) {
  const { t } = useTranslation()
  const [editingId, setEditingId] = useState('0')
  const role = props.roles.find((entry) => entry.id === Number(editingId))
  const form = useForm<ScopedRoleValues>({
    resolver: zodResolver(scopedRoleSchema),
    defaultValues: { name: '', active: true, grants: {} },
  })
  useEffect(() => {
    form.reset({
      name: role?.name || '',
      active: role?.active ?? true,
      grants: Object.fromEntries(
        Object.entries(props.policy.grants).map(([resource, cap]) => [
          resource,
          {
            ...cap,
            ...Object.fromEntries(
              actions.map((action) => [
                action,
                !!role?.grants[resource]?.[action] && cap[action],
              ])
            ),
          },
        ])
      ),
    })
  }, [form, role, props.policy])
  return (
    <form
      className='space-y-4 rounded-lg border p-4'
      onSubmit={form.handleSubmit((values) =>
        props.write('/api/rbac/custom-roles', {
          ...values,
          id: Number(editingId),
          version: role?.version || 0,
          tenant_id: props.tenantId,
          family: props.policy.family,
        })
      )}
    >
      <h3 className='font-semibold'>{t('Scoped role configuration')}</h3>
      <div className='grid gap-4 sm:grid-cols-2'>
        <label>
          {t('Role')}
          <select
            className='bg-background mt-1 block w-full rounded-md border px-3 py-2 text-sm'
            value={editingId}
            disabled={props.busy}
            onChange={(event) => setEditingId(event.target.value)}
          >
            <option value='0'>{t('Create role')}</option>
            {props.roles.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.name} · #{entry.id}
                {entry.active ? '' : ' · ' + t('Disabled')}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t('Role name')}
          <input
            className='bg-background mt-1 block w-full rounded-md border px-3 py-2 text-sm'
            {...form.register('name')}
            disabled={props.busy}
            aria-invalid={!!form.formState.errors.name}
          />
          {form.formState.errors.name && (
            <span role='alert' className='text-destructive text-sm'>
              {t('Role name must contain 1 to 128 UTF-8 bytes')}
            </span>
          )}
        </label>
      </div>
      <GrantsEditor
        value={form.watch('grants')}
        ceiling={props.policy.grants}
        onChange={(value) => form.setValue('grants', value)}
        disabled={props.busy}
      />
      <label className='flex items-center gap-2'>
        <input
          type='checkbox'
          {...form.register('active')}
          disabled={props.busy}
        />
        {t('Role enabled')}
      </label>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Disabling a role removes its business permissions. Audit records and financial history are retained.'
        )}
      </p>
      <Button type='submit' disabled={props.busy || props.tenantId <= 1}>
        {t('Verify and save role')}
      </Button>
    </form>
  )
}
