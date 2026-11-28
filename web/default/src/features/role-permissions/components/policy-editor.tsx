import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { policySchema, type PolicyValues } from '../lib/schema'
import type { Policy, WriteRoleConfiguration } from '../types'
import { GrantsEditor } from './grants-editor'

export function PolicyEditor(props: {
  policy: Policy
  busy: boolean
  write: WriteRoleConfiguration
}) {
  const { t } = useTranslation()
  const form = useForm<PolicyValues>({
    resolver: zodResolver(policySchema),
    defaultValues: {
      delegate: props.policy.delegate,
      grants: props.policy.grants,
    },
  })
  useEffect(() => {
    form.reset({ delegate: props.policy.delegate, grants: props.policy.grants })
  }, [form, props.policy])
  return (
    <form
      className='space-y-4 rounded-lg border p-4'
      onSubmit={form.handleSubmit((values) =>
        props.write(
          '/api/rbac/policy',
          {
            ...values,
            family: props.policy.family,
            version: props.policy.version,
          },
          'put'
        )
      )}
    >
      <h3 className='font-semibold'>{t('Platform permission ceiling')}</h3>
      <GrantsEditor
        value={form.watch('grants')}
        ceiling={props.policy.ceiling}
        onChange={(value) => form.setValue('grants', value)}
        disabled={props.busy}
      />
      <label className='flex items-center gap-2'>
        <input
          type='checkbox'
          {...form.register('delegate')}
          disabled={props.busy}
        />
        {t('Allow primary administrators to create and assign scoped roles')}
      </label>
      <Button type='submit' disabled={props.busy}>
        {t('Verify and save policy')}
      </Button>
    </form>
  )
}
