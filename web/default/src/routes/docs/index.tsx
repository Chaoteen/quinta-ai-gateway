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
import { createFileRoute } from '@tanstack/react-router'
import {
  BookOpen,
  CircleHelp,
  CreditCard,
  FileKey,
  FileText,
  KeyRound,
  LayoutDashboard,
  ReceiptText,
  Ticket,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const docs = [
  {
    titleKey: 'Quick Start',
    descriptionKey: 'Set up Quinta AI Gateway and complete the first request.',
    path: 'docs/user-guide/quick_start.md',
    icon: BookOpen,
  },
  {
    titleKey: 'API Access Guide',
    descriptionKey: 'Connect applications through the unified gateway API.',
    path: 'docs/user-guide/api_access.md',
    icon: FileKey,
  },
  {
    titleKey: 'Account and API Key',
    descriptionKey: 'Manage accounts, API keys, permissions, and key safety.',
    path: 'docs/user-guide/account_and_api_key.md',
    icon: KeyRound,
  },
  {
    titleKey: 'Quota and Billing',
    descriptionKey: 'Understand quota balance, usage metering, and billing records.',
    path: 'docs/user-guide/quota_and_billing.md',
    icon: CreditCard,
  },
  {
    titleKey: 'Subscription Management',
    descriptionKey: 'Use plans and subscriptions to manage recurring service rights.',
    path: 'docs/user-guide/subscription.md',
    icon: ReceiptText,
  },
  {
    titleKey: 'Voucher Management',
    descriptionKey: 'Issue, redeem, and audit quota or subscription vouchers.',
    path: 'docs/user-guide/voucher.md',
    icon: Ticket,
  },
  {
    titleKey: 'Invoice Management',
    descriptionKey: 'Track invoice applications, statuses, and billing alignment.',
    path: 'docs/user-guide/invoice.md',
    icon: FileText,
  },
  {
    titleKey: 'Admin Console',
    descriptionKey: 'Operate tenant users, channels, billing, usage, and settings.',
    path: 'docs/user-guide/admin_console.md',
    icon: LayoutDashboard,
  },
  {
    titleKey: 'FAQ',
    descriptionKey: 'Review common questions about access, billing, and operations.',
    path: 'docs/user-guide/faq.md',
    icon: CircleHelp,
  },
]

function DocsPage() {
  const { t } = useTranslation()

  return (
    <PublicLayout>
      <main className='mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-24 md:py-28'>
        <div className='max-w-3xl space-y-3'>
          <h1 className='text-3xl font-semibold tracking-tight md:text-4xl'>
            {t('Quinta AI Gateway Documentation Center')}
          </h1>
          <p className='text-muted-foreground text-base leading-7'>
            {t(
              'Enterprise AI Gateway & MaaS Platform documentation for onboarding, API access, billing, subscriptions, vouchers, invoices, and admin operations.'
            )}
          </p>
        </div>

        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
          {docs.map((item) => {
            const Icon = item.icon
            return (
              <Card key={item.path} size='sm'>
                <CardHeader>
                  <CardTitle className='flex items-center gap-2 text-base'>
                    <span className='bg-muted flex size-8 items-center justify-center rounded-md'>
                      <Icon className='size-4' aria-hidden='true' />
                    </span>
                    {t(item.titleKey)}
                  </CardTitle>
                  <CardDescription>{t(item.descriptionKey)}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className='text-muted-foreground rounded-md border px-3 py-2 text-xs font-medium'>
                    {item.path}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </main>
    </PublicLayout>
  )
}

export const Route = createFileRoute('/docs/')({
  component: DocsPage,
})
