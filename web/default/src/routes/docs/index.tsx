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
import { createFileRoute, Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { userGuideDocs } from '@/features/docs/docs-content'

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
          {userGuideDocs.map((item) => {
            const Icon = item.icon
            return (
              <Link
                key={item.slug}
                to='/docs/$slug'
                params={{ slug: item.slug }}
                className='group block rounded-xl focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none'
              >
                <Card
                  size='sm'
                  className='h-full transition-colors group-hover:border-primary/40 group-hover:bg-muted/30'
                >
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
                    <div className='text-primary flex items-center gap-2 text-sm font-medium'>
                      {t('Open document')}
                      <ArrowRight className='size-4 transition-transform group-hover:translate-x-0.5' />
                    </div>
                  </CardContent>
                </Card>
              </Link>
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
