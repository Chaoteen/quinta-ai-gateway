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
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { SubscriptionsDialogs } from './components/subscriptions-dialogs'
import { SubscriptionsPrimaryButtons } from './components/subscriptions-primary-buttons'
import { SubscriptionsProvider } from './components/subscriptions-provider'
import { SubscriptionsTable } from './components/subscriptions-table'
import { UserSubscriptionsSection } from './components/user-subscriptions-section'

export function Subscriptions() {
  const { t } = useTranslation()
  return (
    <SubscriptionsProvider>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Subscription Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('Manage subscription plans and assigned user subscriptions')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <SubscriptionsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-8'>
            <div className='space-y-3'>
              <div>
                <h2 className='text-base font-semibold'>
                  {t('Subscription Plans')}
                </h2>
                <p className='text-muted-foreground text-sm'>
                  {t('Manage plan metadata, pricing, quotas and status')}
                </p>
              </div>
              <SubscriptionsTable />
            </div>
            <UserSubscriptionsSection />
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <SubscriptionsDialogs />
    </SubscriptionsProvider>
  )
}
