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
import { AnimateInView } from '@/components/animate-in-view'

const phases = [
  {
    phaseKey: 'Phase 1',
    titleKey: 'AI Gateway',
    descriptionKey: 'Unified model routing, provider governance, and access control.',
  },
  {
    phaseKey: 'Phase 2',
    titleKey: 'MaaS Platform',
    descriptionKey: 'Quota, billing, subscriptions, vouchers, invoices, and tenant operations.',
  },
  {
    phaseKey: 'Phase 3',
    titleKey: 'Agent Marketplace',
    descriptionKey: 'Skills, agents, and SaaS marketplace capabilities for enterprise teams.',
  },
  {
    phaseKey: 'Phase 4',
    titleKey: 'Enterprise AI OS',
    descriptionKey: 'A complete operating layer for enterprise AI applications and workflows.',
  },
]

export function Roadmap() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-6 py-16 md:py-24'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-10 text-center'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Product Roadmap')}
          </p>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('From gateway to enterprise AI operating layer')}
          </h2>
        </AnimateInView>

        <div className='grid gap-4 md:grid-cols-4'>
          {phases.map((phase, index) => (
            <AnimateInView
              key={phase.phaseKey}
              delay={index * 100}
              animation='fade-up'
              className='border-border bg-background relative rounded-xl border p-5'
            >
              <div className='text-muted-foreground text-xs font-semibold tracking-widest uppercase'>
                {t(phase.phaseKey)}
              </div>
              <h3 className='mt-4 text-lg font-semibold'>{t(phase.titleKey)}</h3>
              <p className='text-muted-foreground mt-3 text-sm leading-6'>
                {t(phase.descriptionKey)}
              </p>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}
