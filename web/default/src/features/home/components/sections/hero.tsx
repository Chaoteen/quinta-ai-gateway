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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 flex flex-col items-center overflow-hidden px-6 pt-28 pb-14 md:pt-36 md:pb-20'>
      <div
        aria-hidden
        className='absolute inset-0 -z-10 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_70%_55%_at_50%_25%,black_20%,transparent_100%)] bg-[size:4rem_4rem] opacity-[0.08]'
      />

      <div className='flex max-w-4xl flex-col items-center text-center'>
        <h1
          className='landing-animate-fade-up text-[clamp(2rem,6vw,4.25rem)] leading-[1.08] font-bold tracking-tight'
          style={{ animationDelay: '0ms' }}
        >
          {t('Unified API Gateway for')}
          <br />
          <span className='text-foreground'>
            {t('All Your AI Models')}
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
              className='group rounded-lg'
              render={<Link to='/dashboard' />}
            >
              {t('Go to Dashboard')}
              <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Button>
          ) : (
            <>
              <Button
                className='group rounded-lg'
                render={<Link to='/sign-up' />}
              >
                {t('Start Using')}
                <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
              <Button
                variant='outline'
                className='border-border/50 hover:border-border hover:bg-muted/50 rounded-lg'
                render={<Link to='/about' />}
              >
                {t('View Product')}
              </Button>
            </>
          )}
        </div>
      </div>
    </section>
  )
}
