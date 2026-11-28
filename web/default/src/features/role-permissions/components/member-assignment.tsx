import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { memberSchema, type MemberValues } from '../lib/schema'
import type { Role, WriteRoleConfiguration } from '../types'

export function MemberAssignment(props: {
  family: string
  roles: Role[]
  root: boolean
  busy: boolean
  write: WriteRoleConfiguration
}) {
  const { t } = useTranslation()
  const form = useForm<MemberValues>({
    resolver: zodResolver(memberSchema),
    defaultValues: { target: '', roleId: '' },
  })
  const submit = form.handleSubmit((values, event) => {
    const intent = (event?.nativeEvent as SubmitEvent).submitter?.getAttribute(
      'value'
    )
    if (intent === 'revoke')
      props.write(
        '/api/rbac/assignments/' + Number(values.target),
        undefined,
        'delete'
      )
    else if (intent === 'primary' && props.root)
      props.write('/api/rbac/assign', {
        user_id: Number(values.target),
        role_key: props.family,
      })
    else if (
      props.roles.some(
        (role) => role.id === Number(values.roleId) && role.active
      )
    )
      props.write('/api/rbac/assignments', {
        user_id: Number(values.target),
        role_id: Number(values.roleId),
      })
    else form.setError('roleId', { message: 'Select a role' })
  })
  return (
    <form className='space-y-4 rounded-lg border p-4' onSubmit={submit}>
      <h3 className='font-semibold'>{t('Assign roles to members')}</h3>
      <div className='grid gap-4 sm:grid-cols-2'>
        <label>
          {t('User ID')}
          <input
            className='bg-background mt-1 block w-full rounded-md border px-3 py-2 text-sm'
            inputMode='numeric'
            {...form.register('target')}
            disabled={props.busy}
            aria-invalid={!!form.formState.errors.target}
          />
          {form.formState.errors.target && (
            <span role='alert' className='text-destructive text-sm'>
              {t('Enter a positive user ID')}
            </span>
          )}
        </label>
        <label>
          {t('Scoped role')}
          <select
            className='bg-background mt-1 block w-full rounded-md border px-3 py-2 text-sm'
            aria-label={t('Scoped role')}
            {...form.register('roleId')}
            disabled={props.busy}
          >
            <option value=''>{t('Select a role')}</option>
            {props.roles
              .filter((role) => role.active)
              .map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name} · #{role.id}
                </option>
              ))}
          </select>
          {form.formState.errors.roleId && (
            <span role='alert' className='text-destructive text-sm'>
              {t('Select a role')}
            </span>
          )}
        </label>
      </div>
      <div className='flex flex-wrap gap-3'>
        <Button
          type='submit'
          name='intent'
          value='assign'
          disabled={props.busy}
        >
          {t('Verify and assign role')}
        </Button>
        <Button
          type='submit'
          name='intent'
          value='revoke'
          variant='outline'
          disabled={props.busy}
        >
          {t('Revoke scoped role')}
        </Button>
        {props.root && (
          <Button
            type='submit'
            name='intent'
            value='primary'
            variant='outline'
            disabled={props.busy}
          >
            {t('Set as primary administrator')}
          </Button>
        )}
      </div>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Assignments require a matching tenant. Users must sign in again after assignment or revocation. Only the platform can appoint primary administrators.'
        )}
      </p>
    </form>
  )
}
