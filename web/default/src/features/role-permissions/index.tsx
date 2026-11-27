import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { SectionPageLayout } from '@/components/layout'
import { SecureVerificationDialog, useSecureVerification } from '@/features/auth/secure-verification'

type Grant = { view: boolean; use: boolean; create: boolean; operate: boolean; update: boolean; delete: boolean; scope: string }
const roleLabels: Record<string,string> = { root: 'Platform administrator', supplier_admin: 'Supplier administrator', distributor_admin: 'Distributor administrator', enterprise_admin: 'Enterprise administrator', user: 'User' }
export function RolePermissions() {
 const { t } = useTranslation()
 const [role,setRole] = useState('enterprise_admin')
 const [target,setTarget] = useState('')
 const [busy,setBusy] = useState(false)
 const verification = useSecureVerification()
 const query = useQuery({ queryKey: ['role-permissions'], queryFn: async () => {
  const { data } = await api.get('/api/rbac/roles')
  if (!data.success) throw new Error(data.message || 'Permission matrix unavailable')
  return data.data as Record<string,Record<string,Grant>>
 } })
 async function assign() {
  if (!Number.isSafeInteger(Number(target)) || Number(target)<1) { toast.error(t('Enter a valid user ID')); return }
  setBusy(true)
  try { await verification.withVerification(async () => {
   const {data} = await api.post('/api/rbac/assign',{user_id:Number(target),role_key:role})
   if (!data.success) throw new Error(data.message)
   toast.success(t('Role assigned. The user must sign in again.'))
  }) } catch (e) { toast.error(e instanceof Error ? e.message : t('Operation failed')) } finally {setBusy(false)}
 }
 return <SectionPageLayout>
 <SectionPageLayout.Title>{t('Roles and permissions')}</SectionPageLayout.Title>
 <SectionPageLayout.Content><div className='space-y-6'>
 <p>{t('Permissions are restricted by resource ownership. Platform procurement and invoice issuance remain platform-only.')}</p>
 <div className='space-y-3 rounded-lg border p-4'>
 <label className='block'>{t('User ID')}<input className='mt-1 block w-full rounded border p-2' inputMode='numeric' value={target} onChange={e=>setTarget(e.target.value)} /></label>
 <label className='block'>{t('Role')}<select className='mt-1 block w-full rounded border p-2' value={role} onChange={e=>setRole(e.target.value)}>
 {Object.entries(roleLabels).filter(([key])=>key!=='root').map(([key,label])=><option key={key} value={key}>{t(label)}</option>)}
 </select></label>
 <p className='text-muted-foreground text-sm'>{t('The user must already belong to a dedicated active tenant. This operation does not move funds or change tenant ownership.')}</p>
 <Button disabled={busy} onClick={assign}>{t('Verify and assign role')}</Button>
 </div>
 {query.isPending && <p>{t('Loading...')}</p>}
 {query.isError && <p role='alert'>{t('Permission matrix unavailable')}</p>}
 {query.data && <div className='overflow-x-auto rounded border'><table className='w-full text-sm'><thead><tr>{['Resource','Scope','Visible','Usable','Create','Operate','Update','Delete'].map(x=><th key={x} className='p-3 text-start'>{t(x)}</th>)}</tr></thead>
 <tbody>{Object.entries(query.data[role] || {}).map(([resource,g])=><tr className='border-t' key={resource}><td className='p-3'>{t('rbac.resource.'+resource)}</td><td className='p-3'>{t('rbac.scope.'+g.scope)}</td>{(['view','use','create','operate','update','delete'] as const).map(action=><td key={action} className='p-3'>{t(g[action]?'Allowed':'Denied')}</td>)}</tr>)}</tbody></table></div>}
 <SecureVerificationDialog open={verification.open} onOpenChange={(open) => { if (!open) verification.cancel() }} methods={verification.methods} state={verification.state} onVerify={async (method,code) => { await verification.executeVerification(method,code) }} onCancel={verification.cancel} onCodeChange={verification.setCode} onMethodChange={verification.switchMethod} />
 </div></SectionPageLayout.Content></SectionPageLayout>
}
