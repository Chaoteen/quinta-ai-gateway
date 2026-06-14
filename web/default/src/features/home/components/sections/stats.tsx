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

const metrics = [
  { value: '20+', labelKey: 'Supported Models' },
  { value: '1', labelKey: 'Unified Gateway' },
  { value: '∞', labelKey: 'Enterprise Tenants' },
  { value: '100%', labelKey: 'Billing Capabilities' },
]

export function Stats() {
  const { t } = useTranslation()

  return (
    <section className='border-border/40 bg-muted/20 relative z-10 border-y'>
      <div className='mx-auto max-w-6xl px-6 py-10 md:py-12'>
        <div className='grid grid-cols-2 gap-4 md:grid-cols-4'>
          {metrics.map((metric) => (
            <div
              key={metric.labelKey}
              className='bg-background/70 border-border rounded-lg border p-5 text-center'
            >
              <div className='text-3xl font-semibold tracking-tight md:text-4xl'>
                {metric.value}
              </div>
              <div className='text-muted-foreground mt-2 text-sm'>
                {t(metric.labelKey)}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
