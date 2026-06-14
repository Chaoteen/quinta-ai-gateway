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
import { Link } from '@tanstack/react-router'
import { ArrowRight, BadgePercent, Building2, CreditCard, Ticket } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { AnimateInView } from '@/components/animate-in-view'

const pricingItems = [
  {
    icon: CreditCard,
    titleKey: 'Token Packages',
    descriptionKey: 'Prepaid quota packages for predictable model usage.',
  },
  {
    icon: BadgePercent,
    titleKey: 'Subscription',
    descriptionKey: 'Recurring service rights for teams and tenants.',
  },
  {
    icon: Ticket,
    titleKey: 'Voucher',
    descriptionKey: 'Campaign, quota, and subscription benefit redemption.',
  },
  {
    icon: Building2,
    titleKey: 'Enterprise Plan',
    descriptionKey: 'Custom tenant governance, finance workflow, and support.',
  },
]

export function PricingPreview() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-6 py-16 md:py-24'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='border-border bg-muted/20 rounded-xl border p-6 md:p-8'>
          <div className='flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between'>
            <div className='max-w-2xl'>
              <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
                {t('Pricing Preview')}
              </p>
              <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
                {t('Commercial packaging for AI services')}
              </h2>
              <p className='text-muted-foreground mt-3 text-sm leading-6 md:text-base'>
                {t(
                  'Combine token packages, subscriptions, vouchers, and enterprise plans for different tenant scenarios.'
                )}
              </p>
            </div>
            <Button className='w-fit rounded-lg' render={<Link to='/pricing' />}>
              {t('View Pricing')}
              <ArrowRight className='ml-1 size-3.5' />
            </Button>
          </div>

          <div className='mt-8 grid gap-4 md:grid-cols-4'>
            {pricingItems.map((item) => {
              const Icon = item.icon
              return (
                <div
                  key={item.titleKey}
                  className='bg-background border-border rounded-lg border p-4'
                >
                  <Icon className='size-5' />
                  <h3 className='mt-4 text-sm font-semibold'>
                    {t(item.titleKey)}
                  </h3>
                  <p className='text-muted-foreground mt-2 text-sm leading-6'>
                    {t(item.descriptionKey)}
                  </p>
                </div>
              )
            })}
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
