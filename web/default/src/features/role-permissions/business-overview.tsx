import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { canResource } from '@/lib/rbac'
import { SectionPageLayout } from '@/components/layout'

export function BusinessOverview() {
  const { t } = useTranslation()
  const actor = useAuthStore((state) => state.auth.user)
  const links = [
    { resource: 'model_calls', to: '/playground', label: 'Online chat' },
    { resource: 'members', to: '/users', label: 'User Management' },
    { resource: 'supply_models', to: '/channels', label: 'Channel Management' },
    { resource: 'keys', to: '/keys', label: 'API Keys' },
    { resource: 'payments', to: '/billing', label: 'Billing Center' },
    { resource: 'invoice_requests', to: '/invoices', label: 'Invoices' },
    { resource: 'vouchers', to: '/vouchers', label: 'Vouchers' },
  ] as const
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Business Workspace')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='max-w-[100rem] space-y-5'>
          <p className='text-muted-foreground text-sm'>
            {t('Only resources allowed by your current role are shown.')}
          </p>
          <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
            {links
              .filter((link) =>
                canResource(
                  actor,
                  link.resource,
                  link.resource === 'model_calls' ? 'use' : 'view'
                )
              )
              .map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className='hover:bg-accent focus-visible:ring-ring rounded-lg border p-5 font-medium focus-visible:ring-2'
                >
                  {t(link.label)}
                </Link>
              ))}
            {actor?.can_delegate_roles && (
              <Link
                to='/role-permissions'
                className='hover:bg-accent rounded-lg border p-5 font-medium'
              >
                {t('Roles and permissions')}
              </Link>
            )}
            <Link
              to='/profile'
              className='hover:bg-accent rounded-lg border p-5 font-medium'
            >
              {t('Personal Settings')}
            </Link>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
