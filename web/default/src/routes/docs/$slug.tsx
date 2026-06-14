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
import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { getUserGuideDoc } from '@/features/docs/docs-content'

function UserGuideDocPage() {
  const { slug } = Route.useParams()
  const { t } = useTranslation()
  const doc = getUserGuideDoc(slug)

  if (!doc) {
    throw notFound()
  }

  const Icon = doc.icon

  return (
    <PublicLayout>
      <main className='mx-auto flex w-full max-w-4xl flex-col gap-8 px-4 py-24 md:py-28'>
        <Button
          variant='ghost'
          className='w-fit gap-2 px-0'
          render={<Link to='/docs' />}
        >
          <ArrowLeft className='size-4' />
          {t('Back to documentation center')}
        </Button>

        <header className='space-y-4'>
          <div className='bg-muted flex size-12 items-center justify-center rounded-lg'>
            <Icon className='size-6' aria-hidden='true' />
          </div>
          <div className='space-y-3'>
            <h1 className='text-3xl font-semibold tracking-tight md:text-4xl'>
              {t(doc.titleKey)}
            </h1>
            <p className='text-muted-foreground text-base leading-7'>
              {t(doc.descriptionKey)}
            </p>
          </div>
        </header>

        <div className='space-y-6'>
          {doc.sections.map((section) => (
            <section
              key={section.headingKey}
              className='border-border border-t pt-6'
            >
              <h2 className='text-xl font-semibold tracking-tight'>
                {t(section.headingKey)}
              </h2>
              <p className='text-muted-foreground mt-3 leading-7'>
                {t(section.bodyKey)}
              </p>
            </section>
          ))}
        </div>
      </main>
    </PublicLayout>
  )
}

export const Route = createFileRoute('/docs/$slug')({
  component: UserGuideDocPage,
})
