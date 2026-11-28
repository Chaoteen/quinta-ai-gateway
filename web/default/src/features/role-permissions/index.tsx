import { useEffect, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { isRootUser } from '@/lib/rbac'
import { Button } from '@/components/ui/button'
import { SectionPageLayout } from '@/components/layout'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'

type Action = 'view' | 'use' | 'create' | 'operate' | 'update' | 'delete'
type Grant = Record<Action, boolean> & { scope: string }
type Grants = Record<string, Grant>
type Policy = {
  family: string
  delegate: boolean
  version: number
  grants: Grants
  ceiling: Grants
}
type Role = {
  id: number
  tenant_id: number
  family: string
  name: string
  active: boolean
  version: number
  grants: Grants
}
type Workspace = {
  state: {
    family: string
    tenant_id: number
    role_id: number
    can_delegate: boolean
    grants: Grants
  }
  policies: Policy[]
  roles: Role[]
  audits: {
    id: number
    actor_id: number
    target_user_id: number
    role_id: number
    action: string
    created_at: number
  }[]
  truncated: boolean
}
const actions: Action[] = [
  'view',
  'use',
  'create',
  'operate',
  'update',
  'delete',
]
const actionLabels = [
  'Visible',
  'Usable',
  'Create',
  'Operate',
  'Update',
  'Delete',
]
const labels: Record<string, string> = {
  supplier_admin: 'Supplier administrator',
  distributor_admin: 'Distributor administrator',
  enterprise_admin: 'Enterprise administrator',
}
const fieldClass =
  'mt-1 block w-full rounded-md border bg-background px-3 py-2 text-sm'

function GrantsEditor({
  value,
  ceiling,
  onChange,
  disabled = false,
}: {
  value: Grants
  ceiling: Grants
  onChange: (v: Grants) => void
  disabled?: boolean
}) {
  const { t } = useTranslation()
  function toggle(resource: string, action: Action, checked: boolean) {
    const next = {
      ...(value[resource] || ceiling[resource]),
      [action]: checked,
    }
    if (action === 'view' && !checked)
      actions.forEach((a) => {
        next[a] = false
      })
    if (action !== 'view' && checked) next.view = true
    onChange({ ...value, [resource]: next })
  }
  return (
    <div className='overflow-x-auto rounded-lg border'>
      <table className='w-full text-sm'>
        <thead>
          <tr>
            <th className='p-3 text-start'>{t('Resource')}</th>
            <th className='p-3 text-start'>{t('Scope')}</th>
            {actionLabels.map((a) => (
              <th className='p-3 text-center' key={a}>
                {t(a)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Object.entries(ceiling).map(([resource, cap]) => (
            <tr className='border-t' key={resource}>
              <td className='p-3'>{t('rbac.resource.' + resource)}</td>
              <td className='p-3'>{t('rbac.scope.' + cap.scope)}</td>
              {actions.map((action, i) => (
                <td className='p-2 text-center' key={action}>
                  <input
                    type='checkbox'
                    className='accent-primary size-4'
                    aria-label={
                      t('rbac.resource.' + resource) +
                      ' / ' +
                      t(actionLabels[i])
                    }
                    disabled={disabled || !cap[action]}
                    checked={!!value[resource]?.[action] && cap[action]}
                    onChange={(e) => toggle(resource, action, e.target.checked)}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
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
  const [editingId, setEditingId] = useState('0')
  const [name, setName] = useState('')
  const [active, setActive] = useState(true)
  const [draft, setDraft] = useState<Grants>({})
  const [policyDraft, setPolicyDraft] = useState<Grants>({})
  const [delegate, setDelegate] = useState(false)
  const [target, setTarget] = useState('')
  const [assignId, setAssignId] = useState('')
  const [busy, setBusy] = useState(false)
  const verification = useSecureVerification()
  const validTenant =
    tenant === '' ||
    (Number.isSafeInteger(Number(tenant)) && Number(tenant) > 1)
  const query = useQuery({
    queryKey: ['rbac-workspace', tenant],
    enabled: validTenant,
    refetchOnWindowFocus: false,
    queryFn: async () => {
      const { data } = await api.get('/api/rbac/workspace', {
        params: tenant ? { tenant_id: Number(tenant) } : {},
      })
      if (!data.success)
        throw new Error(data.message || 'Role operation unavailable')
      return data.data as Workspace
    },
  })
  const policy = query.data?.policies.find((p) => p.family === family)
  const role = query.data?.roles.find((r) => r.id === Number(editingId))
  const roleList = (query.data?.roles || []).filter((r) => r.family === family)
  const manager = root || query.data?.state.can_delegate === true
  useEffect(() => {
    if (policy) {
      setPolicyDraft(policy.grants)
      setDelegate(policy.delegate)
    }
  }, [policy])
  useEffect(() => {
    setName(role?.name || '')
    setActive(role?.active ?? true)
    setDraft(
      Object.fromEntries(
        Object.entries(policy?.grants || {}).map(([key, cap]) => [
          key,
          {
            ...cap,
            ...Object.fromEntries(
              actions.map((action) => [
                action,
                !!role?.grants[key]?.[action] && cap[action],
              ])
            ),
          },
        ])
      )
    )
  }, [role, policy])
  async function mutate(
    path: string,
    body: unknown,
    method: 'post' | 'put' | 'delete' = 'post'
  ) {
    setBusy(true)
    try {
      await verification.withVerification(async () => {
        const { data } = await api.request({ url: path, method, data: body })
        if (!data.success) throw new Error(data.message)
        await cache.invalidateQueries({ queryKey: ['rbac-workspace'] })
        toast.success(t('Role configuration saved'))
      })
    } catch (e) {
      const error = e as {
        response?: { data?: { message?: string } }
        message?: string
      }
      toast.error(
        error.response?.data?.message || error.message || t('Operation failed')
      )
    } finally {
      setBusy(false)
    }
  }
  const validTarget = Number.isSafeInteger(Number(target)) && Number(target) > 0
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
                value={tenant}
                inputMode='numeric'
                disabled={!root}
                onChange={(e) => {
                  setTenant(e.target.value)
                  setEditingId('0')
                  setAssignId('')
                }}
              />
            </label>
            <label>
              {t('Administrator type')}
              <select
                className={fieldClass}
                value={family}
                disabled={!root}
                onChange={(e) => {
                  setFamily(e.target.value)
                  setEditingId('0')
                  setAssignId('')
                }}
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
            <section className='space-y-4 rounded-lg border p-4'>
              <h3 className='font-semibold'>
                {t('Platform permission ceiling')}
              </h3>
              <GrantsEditor
                value={policyDraft}
                ceiling={policy.ceiling}
                onChange={setPolicyDraft}
                disabled={busy}
              />
              <label className='flex items-center gap-2'>
                <input
                  type='checkbox'
                  checked={delegate}
                  onChange={(e) => setDelegate(e.target.checked)}
                />
                {t(
                  'Allow primary administrators to create and assign scoped roles'
                )}
              </label>
              <Button
                disabled={busy}
                onClick={() =>
                  mutate(
                    '/api/rbac/policy',
                    {
                      family,
                      grants: policyDraft,
                      delegate,
                      version: policy.version,
                    },
                    'put'
                  )
                }
              >
                {t('Verify and save policy')}
              </Button>
            </section>
          )}
          {policy && manager && (
            <section className='space-y-4 rounded-lg border p-4'>
              <h3 className='font-semibold'>
                {t('Scoped role configuration')}
              </h3>
              <div className='grid gap-4 sm:grid-cols-2'>
                <label>
                  {t('Role')}
                  <select
                    className={fieldClass}
                    value={editingId}
                    onChange={(e) => setEditingId(e.target.value)}
                  >
                    <option value='0'>{t('Create role')}</option>
                    {roleList.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} · #{r.id}
                        {r.active ? '' : ' · ' + t('Disabled')}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  {t('Role name')}
                  <input
                    className={fieldClass}
                    value={name}
                    maxLength={100}
                    onChange={(e) => setName(e.target.value)}
                  />
                </label>
              </div>
              <GrantsEditor
                value={draft}
                ceiling={policy.grants}
                onChange={setDraft}
                disabled={busy}
              />
              <label className='flex items-center gap-2'>
                <input
                  type='checkbox'
                  checked={active}
                  onChange={(e) => setActive(e.target.checked)}
                />
                {t('Role enabled')}
              </label>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Disabling a role removes its business permissions. Audit records and financial history are retained.'
                )}
              </p>
              <Button
                disabled={busy || !name.trim() || !tenant || !validTenant}
                onClick={() =>
                  mutate('/api/rbac/custom-roles', {
                    id: Number(editingId),
                    version: role?.version || 0,
                    tenant_id: Number(tenant),
                    family,
                    name,
                    active,
                    grants: draft,
                  })
                }
              >
                {t('Verify and save role')}
              </Button>
            </section>
          )}
          {policy && manager && (
            <section className='space-y-4 rounded-lg border p-4'>
              <h3 className='font-semibold'>{t('Assign roles to members')}</h3>
              <div className='grid gap-4 sm:grid-cols-2'>
                <label>
                  {t('User ID')}
                  <input
                    className={fieldClass}
                    inputMode='numeric'
                    value={target}
                    onChange={(e) => setTarget(e.target.value)}
                  />
                </label>
                <label>
                  {t('Scoped role')}
                  <select
                    className={fieldClass}
                    value={assignId}
                    onChange={(e) => setAssignId(e.target.value)}
                  >
                    <option value=''>{t('Select a role')}</option>
                    {roleList
                      .filter((r) => r.active)
                      .map((r) => (
                        <option key={r.id} value={r.id}>
                          {r.name} · #{r.id}
                        </option>
                      ))}
                  </select>
                </label>
              </div>
              <div className='flex flex-wrap gap-3'>
                <Button
                  disabled={busy || !validTarget || !assignId}
                  onClick={() =>
                    mutate('/api/rbac/assignments', {
                      user_id: Number(target),
                      role_id: Number(assignId),
                    })
                  }
                >
                  {t('Verify and assign role')}
                </Button>
                <Button
                  variant='outline'
                  disabled={busy || !validTarget}
                  onClick={() =>
                    mutate(
                      '/api/rbac/assignments/' + Number(target),
                      undefined,
                      'delete'
                    )
                  }
                >
                  {t('Revoke scoped role')}
                </Button>
                {root && (
                  <Button
                    variant='outline'
                    disabled={busy || !validTarget}
                    onClick={() =>
                      mutate('/api/rbac/assign', {
                        user_id: Number(target),
                        role_key: family,
                      })
                    }
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
            </section>
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
