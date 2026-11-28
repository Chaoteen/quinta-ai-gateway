import { useState } from 'react'
import { AxiosError } from 'axios'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { handleServerError } from '@/lib/handle-server-error'
import { isRootUser } from '@/lib/rbac'
import { isVerificationRequiredError } from '@/lib/secure-verification'
import { SectionPageLayout } from '@/components/layout'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'
import { MemberAssignment } from './components/member-assignment'
import { PolicyEditor } from './components/policy-editor'
import { ScopedRoleEditor } from './components/scoped-role-editor'
import type { Workspace } from './types'

const labels: Record<string, string> = {
  supplier_admin: 'Supplier administrator',
  distributor_admin: 'Distributor administrator',
  enterprise_admin: 'Enterprise administrator',
}
const fieldClass =
  'mt-1 block w-full rounded-md border bg-background px-3 py-2 text-sm'
export function RolePermissions() {
  const { t } = useTranslation()
  const cache = useQueryClient()
  const actor = useAuthStore((state) => state.auth.user)
  const root = isRootUser(actor)
  const [tenant, setTenant] = useState(
    root ? '' : String(actor?.tenant_id || '')
  )
  const [family, setFamily] = useState(
    root ? 'enterprise_admin' : actor?.role_key || ''
  )
  const validTenant =
    tenant === '' ||
    (Number.isSafeInteger(Number(tenant)) && Number(tenant) > 1)
  const verification = useSecureVerification()
  const query = useQuery({
    queryKey: ['rbac-workspace', tenant],
    enabled: validTenant,
    refetchOnWindowFocus: 'always',
    refetchInterval: 30000,
    queryFn: async () => {
      const { data } = await api.get('/api/rbac/workspace', {
        params: tenant ? { tenant_id: Number(tenant) } : {},
      })
      if (!data.success) throw new Error('Role operation unavailable')
      return data.data as Workspace
    },
  })
  const mutation = useMutation({
    mutationFn: async (input: {
      path: string
      body: unknown
      method: 'post' | 'put' | 'delete'
    }) => {
      const { data } = await api.request({
        url: input.path,
        method: input.method,
        data: input.body,
        ...{ skipErrorHandler: true, skipBusinessError: true },
      })
      if (!data.success) throw new Error('Role operation unavailable')
    },
    onSuccess: async () => {
      await cache.invalidateQueries({ queryKey: ['rbac-workspace'] })
      await cache.invalidateQueries({ queryKey: ['rbac-state'] })
      toast.success(t('Role configuration saved'))
    },
    onError: (error) => {
      if (isVerificationRequiredError(error)) return
      const conflict =
        error instanceof AxiosError && error.response?.status === 409
      handleServerError(
        error,
        t(
          conflict
            ? 'Role changed. Refresh and retry.'
            : 'Role operation unavailable'
        )
      )
    },
  })
  const write = (
    path: string,
    body: unknown,
    method: 'post' | 'put' | 'delete' = 'post'
  ): void => {
    // The verification wrapper may return before the dialog completes. Keep
    // success/invalidation on the actual mutation, including its verified retry.
    void verification
      .withVerification(() => mutation.mutateAsync({ path, body, method }))
      .catch(() => {
        // The mutation's onError owns the translated error message.
      })
  }
  const policy = query.data?.policies.find((entry) => entry.family === family)
  const roles = (query.data?.roles || []).filter(
    (entry) => entry.family === family
  )
  const manager =
    root ||
    (actor?.can_delegate_roles === true &&
      query.data?.state.can_delegate === true)
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Roles and permissions')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='max-w-[100rem] space-y-6'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Platform sets permission ceilings. Business administrators can assign scoped roles only within those ceilings.'
            )}
          </p>
          <div className='grid gap-4 rounded-lg border p-4 sm:grid-cols-2'>
            <label>
              {t('Tenant ID')}
              <input
                className={fieldClass}
                inputMode='numeric'
                value={tenant}
                disabled={!root || mutation.isPending}
                onChange={(event) => setTenant(event.target.value)}
              />
            </label>
            <label>
              {t('Administrator type')}
              <select
                className={fieldClass}
                value={family}
                disabled={!root || mutation.isPending}
                onChange={(event) => setFamily(event.target.value)}
              >
                {Object.entries(labels).map(([key, label]) => (
                  <option key={key} value={key}>
                    {t(label)}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {!validTenant && (
            <p role='alert'>
              {t('Enter a dedicated tenant ID greater than 1')}
            </p>
          )}
          {query.isPending && validTenant && <p>{t('Loading...')}</p>}
          {query.isError && (
            <p role='alert'>{t('Permission matrix unavailable')}</p>
          )}
          {policy && root && (
            <PolicyEditor
              key={family}
              policy={policy}
              busy={mutation.isPending || verification.open}
              write={write}
            />
          )}
          {policy && manager && (
            <ScopedRoleEditor
              key={'role-' + family + '-' + tenant}
              policy={policy}
              roles={roles}
              tenantId={validTenant ? Number(tenant) : 0}
              busy={mutation.isPending || verification.open}
              write={write}
            />
          )}
          {policy && manager && (
            <MemberAssignment
              key={'assignment-' + family + '-' + tenant}
              family={family}
              roles={roles}
              root={root}
              busy={mutation.isPending || verification.open}
              write={write}
            />
          )}
          {query.data && !manager && (
            <p>
              {t(
                'You can view your effective permissions, but cannot delegate roles.'
              )}
            </p>
          )}
          {query.data?.truncated && (
            <p role='status'>
              {t('Too many roles. Filter by tenant ID to narrow the list.')}
            </p>
          )}
          {query.data && (
            <section className='space-y-3'>
              <h3 className='font-semibold'>{t('Recent role changes')}</h3>
              <div className='overflow-x-auto rounded-lg border'>
                <table className='w-full text-sm'>
                  <thead>
                    <tr>
                      {['Time', 'Action', 'User ID', 'Role'].map((x) => (
                        <th className='p-3 text-start' key={x}>
                          {t(x)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {query.data.audits.map((a) => (
                      <tr className='border-t' key={a.id}>
                        <td className='p-3'>
                          {new Date(a.created_at * 1000).toLocaleString()}
                        </td>
                        <td className='p-3'>{t('rbac.audit.' + a.action)}</td>
                        <td className='p-3'>{a.target_user_id || '—'}</td>
                        <td className='p-3'>{a.role_id || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
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
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
