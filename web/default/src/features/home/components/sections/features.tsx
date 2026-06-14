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
import {
  Building2,
  CreditCard,
  Layers3,
  PackageOpen,
  Store,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

const featureItems = [
  {
    icon: Layers3,
    titleKey: 'Unified Model Access',
    descriptionKey: 'Connect once and switch across models.',
    tags: ['OpenAI', 'Claude', 'Gemini', 'DeepSeek', 'Qwen'],
  },
  {
    icon: PackageOpen,
    titleKey: 'Quota and Billing',
    descriptionKey: 'Quota Engine, Usage Metering, and Billing Runtime.',
    tags: ['Quota Engine', 'Usage Metering', 'Billing Runtime'],
  },
  {
    icon: Building2,
    titleKey: 'Enterprise Multi-Tenancy',
    descriptionKey: 'Tenant, Organization, and Department governance.',
    tags: ['Tenant', 'Organization', 'Department'],
  },
  {
    icon: CreditCard,
    titleKey: 'Payment and Finance',
    descriptionKey: 'Payment, Voucher, Invoice, and Revenue Share.',
    tags: ['Payment', 'Voucher', 'Invoice', 'Revenue Share'],
  },
  {
    icon: Store,
    titleKey: 'Agent Marketplace',
    descriptionKey: 'Future support for Skill, Agent, and SaaS Marketplace.',
    tags: ['Skill', 'Agent', 'SaaS Marketplace'],
  },
]

export function Features() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-6 py-16 md:py-24'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-10 max-w-2xl'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Core Capabilities')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Commercial capabilities for enterprise AI delivery')}
          </h2>
        </AnimateInView>

        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
          {featureItems.map((item, index) => {
            const Icon = item.icon
            return (
              <AnimateInView
                key={item.titleKey}
                delay={index * 80}
                animation='fade-up'
                className='border-border bg-background rounded-xl border p-5'
              >
                <div className='bg-muted mb-4 flex size-10 items-center justify-center rounded-lg'>
                  <Icon className='size-5' />
                </div>
                <h3 className='text-base font-semibold'>{t(item.titleKey)}</h3>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(item.descriptionKey)}
                </p>
                <div className='mt-5 flex flex-wrap gap-2'>
                  {item.tags.map((tag) => (
                    <span
                      key={tag}
                      className='bg-muted text-muted-foreground rounded-md px-2.5 py-1 text-xs font-medium'
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </AnimateInView>
            )
          })}
        </div>
      </div>
    </section>
  )
}
