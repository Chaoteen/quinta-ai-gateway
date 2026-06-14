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
import { ArrowDown, Building2, CreditCard, Layers3, Network } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

const modelProviders = ['OpenAI', 'Claude', 'Gemini', 'DeepSeek', 'Qwen', 'Llama']
const commerceCapabilities = [
  'Quota Engine',
  'Billing',
  'Subscription',
  'Voucher',
  'Invoice',
]
const enterpriseApps = [
  'Enterprise AI Apps',
  'Agent Marketplace',
  'Knowledge Base',
  'Workflow',
]

function NodePill(props: { label: string; emphasis?: boolean }) {
  return (
    <div
      className={
        props.emphasis
          ? 'bg-foreground text-background flex min-h-12 items-center justify-center rounded-lg px-5 text-sm font-semibold shadow-sm'
          : 'bg-background text-foreground border-border flex min-h-11 items-center justify-center rounded-lg border px-4 text-sm font-medium'
      }
    >
      {props.label}
    </div>
  )
}

function DownConnector() {
  return (
    <div className='text-muted-foreground flex justify-center py-3'>
      <ArrowDown className='size-5' />
    </div>
  )
}

export function PlatformArchitecture() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-6 py-16 md:py-24'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-10 text-center'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Platform Architecture')}
          </p>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('From model access to commercial operations')}
          </h2>
        </AnimateInView>

        <AnimateInView
          animation='scale-in'
          className='border-border bg-muted/20 rounded-xl border p-4 md:p-6'
        >
          <div className='space-y-1'>
            <div>
              <div className='mb-3 flex items-center gap-2 text-sm font-semibold'>
                <Network className='size-4' />
                {t('Model Providers')}
              </div>
              <div className='grid grid-cols-2 gap-2 md:grid-cols-6'>
                {modelProviders.map((item) => (
                  <NodePill key={item} label={item} />
                ))}
              </div>
            </div>

            <DownConnector />

            <div>
              <div className='mb-3 flex items-center gap-2 text-sm font-semibold'>
                <Layers3 className='size-4' />
                {t('Gateway Layer')}
              </div>
              <NodePill label='Quinta AI Gateway' emphasis />
            </div>

            <DownConnector />

            <div>
              <div className='mb-3 flex items-center gap-2 text-sm font-semibold'>
                <CreditCard className='size-4' />
                {t('Commercial Runtime')}
              </div>
              <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5'>
                {commerceCapabilities.map((item) => (
                  <NodePill key={item} label={item} />
                ))}
              </div>
            </div>

            <DownConnector />

            <div>
              <div className='mb-3 flex items-center gap-2 text-sm font-semibold'>
                <Building2 className='size-4' />
                {t('Enterprise Applications')}
              </div>
              <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4'>
                {enterpriseApps.map((item) => (
                  <NodePill key={item} label={item} />
                ))}
              </div>
            </div>
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
