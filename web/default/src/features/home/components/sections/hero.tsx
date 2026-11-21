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
import {
  ArrowRight,
  Layers3,
  KeyRound,
  ReceiptText,
  MessageSquare,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const canRegister =
    status?.register_enabled === true &&
    status?.password_register_enabled === true

  return (
    <section className='quinta-hero relative z-10 overflow-hidden px-6 pt-20 pb-14 md:pt-28 md:pb-20'>
      <div className='mx-auto grid max-w-6xl items-center gap-14 lg:grid-cols-[1.1fr_1fr]'>
        <div
          aria-hidden
          className='absolute inset-0 -z-10 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_70%_55%_at_50%_25%,black_20%,transparent_100%)] bg-[size:4rem_4rem] opacity-[0.08]'
        />

        <div className='flex min-w-0 flex-col items-start text-left'>
          <p className='text-primary mb-5 flex items-center gap-2 text-sm font-medium'>
            <span
              className='bg-primary h-2 w-2 rounded-full'
              aria-hidden='true'
            />
            {t('Built for enterprise AI')}
          </p>
          <h1
            className='landing-animate-fade-up text-[clamp(2.25rem,4.7vw,4rem)] leading-[1.16] font-bold tracking-tight'
            style={{ animationDelay: '0ms' }}
          >
            {t('One gateway for enterprise AI')}
            <br />
            <span className='text-foreground'>
              {t('Connect models to your business')}
            </span>
          </h1>
          <p
            className='landing-animate-fade-up text-muted-foreground mt-6 max-w-2xl text-base leading-7 opacity-0 md:text-xl'
            style={{ animationDelay: '80ms' }}
          >
            {t(
              'Enterprise AI Gateway and MaaS Platform for unified model access, quota management, billing, and tenant permissions.'
            )}
          </p>
          <div
            className='landing-animate-fade-up mt-8 flex flex-col items-stretch gap-3 opacity-0 sm:flex-row sm:items-center'
            style={{ animationDelay: '160ms' }}
          >
            {props.isAuthenticated ? (
              <Button
                className='group h-12 rounded-xl px-6'
                render={<Link to='/dashboard' />}
              >
                {t('Go to Dashboard')}
                <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
            ) : (
              <>
                <Button
                  className='group h-12 rounded-xl px-6'
                  render={<Link to={canRegister ? '/sign-up' : '/sign-in'} />}
                >
                  {t('Start Using')}
                  <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
                </Button>
                <Button
                  variant='outline'
                  className='border-border/70 hover:border-border hover:bg-muted/50 h-12 rounded-xl px-6'
                  render={<Link to='/pricing' />}
                >
                  {t('View Product')}
                </Button>
              </>
            )}
          </div>
        </div>
        <div className='quinta-route-map bg-card rounded-3xl border p-6 shadow-xl shadow-slate-900/5 sm:p-8'>
          <div className='mb-7 flex items-center justify-between border-b pb-5'>
            <span className='font-semibold'>Quinta AI Gateway</span>
            <Layers3 aria-hidden='true' className='text-primary size-5' />
          </div>
          <p className='text-muted-foreground mb-3 text-sm'>
            {t('Model access')}
          </p>
          <div className='grid grid-cols-3 gap-2'>
            {['Qwen', 'DeepSeek', 'Claude'].map((name) => (
              <div
                key={name}
                className='bg-muted/40 rounded-xl border px-2 py-3 text-center text-sm font-medium'
              >
                {name}
              </div>
            ))}
          </div>
          <div className='bg-border mx-auto h-8 w-px' aria-hidden='true' />
          <div className='bg-primary text-primary-foreground rounded-2xl p-5'>
            <div className='flex items-center gap-3'>
              <KeyRound aria-hidden='true' className='size-5' />
              <span className='font-semibold'>
                {t('One connection for your team')}
              </span>
            </div>
            <p className='mt-2 text-sm leading-relaxed opacity-90'>
              {t('Manage access, usage and billing in one place.')}
            </p>
          </div>
          <div className='bg-border mx-auto h-8 w-px' aria-hidden='true' />
          <div className='grid grid-cols-2 gap-3'>
            {[
              { icon: MessageSquare, label: 'Online chat', to: '/playground' },
              { icon: ReceiptText, label: 'Billing', to: '/billing' },
            ].map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className='hover:border-primary hover:text-primary focus-visible:outline-primary flex min-h-12 items-center justify-center gap-2 rounded-xl border px-2 py-3 text-sm transition-colors focus-visible:outline-2'
              >
                <item.icon className='size-4' aria-hidden='true' />
                {t(item.label)}
              </Link>
            ))}
          </div>
          <p className='text-muted-foreground mt-5 text-xs leading-5'>
            {t(
              'Available models depend on your account and configured channels.'
            )}
          </p>
        </div>
      </div>
    </section>
  )
}
