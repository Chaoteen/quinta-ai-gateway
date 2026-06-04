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
import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import type { PlanRecord } from '../types'
import { DataTableRowActions } from './data-table-row-actions'

export function useSubscriptionsColumns(): ColumnDef<PlanRecord>[] {
  const { t } = useTranslation()

  return useMemo(
    (): ColumnDef<PlanRecord>[] => [
      {
        accessorFn: (row) => row.plan.id,
        id: 'id',
        meta: { label: 'ID', mobileHidden: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='ID' />
        ),
        cell: ({ row }) => (
          <span className='text-muted-foreground'>#{row.original.plan.id}</span>
        ),
        size: 60,
      },
      {
        accessorFn: (row) => row.plan.name,
        id: 'name',
        meta: { label: t('Plan'), mobileTitle: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Plan')} />
        ),
        cell: ({ row }) => {
          const plan = row.original.plan
          return (
            <div className='max-w-[240px]'>
              <div className='truncate font-medium'>
                {plan.name}
              </div>
              <div className='text-muted-foreground truncate text-xs'>
                {plan.code || `#${plan.id}`}
              </div>
              {plan.description && (
                <div className='text-muted-foreground truncate text-xs'>
                  {plan.description}
                </div>
              )}
            </div>
          )
        },
        size: 240,
      },
      {
        accessorFn: (row) => row.plan.monthly_price,
        id: 'monthly_price',
        meta: { label: t('Monthly Price') },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Monthly Price')} />
        ),
        cell: ({ row }) => (
          <span className='font-semibold text-emerald-600'>
            $
            {Number(row.original.plan.monthly_price ?? 0).toFixed(2)}
          </span>
        ),
        size: 100,
      },
      {
        accessorFn: (row) => row.plan.yearly_price,
        id: 'yearly_price',
        meta: { label: t('Yearly Price'), mobileHidden: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Yearly Price')} />
        ),
        cell: ({ row }) => (
          <span className='text-muted-foreground'>
            ${Number(row.original.plan.yearly_price || 0).toFixed(2)}
          </span>
        ),
        size: 100,
      },
      {
        accessorFn: (row) => row.plan.status,
        id: 'status',
        meta: { label: t('Status'), mobileBadge: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Status')} />
        ),
        cell: ({ row }) =>
          row.original.plan.status === 'enabled' ? (
            <StatusBadge
              label={t('Enable')}
              variant='success'
              copyable={false}
            />
          ) : (
            <StatusBadge
              label={t('Disable')}
              variant='neutral'
              copyable={false}
            />
          ),
        size: 80,
      },
      {
        id: 'token_quota',
        meta: { label: t('Token Quota'), mobileHidden: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Token Quota')} />
        ),
        cell: ({ row }) => {
          const total = Number(
            row.original.plan.token_quota ?? 0
          )
          return (
            <span className='text-muted-foreground'>
              {total > 0 ? total : t('Unlimited')}
            </span>
          )
        },
        size: 100,
      },
      {
        id: 'request_quota',
        meta: { label: t('Request Quota'), mobileHidden: true },
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Request Quota')} />
        ),
        cell: ({ row }) => {
          const total = Number(row.original.plan.request_quota || 0)
          return (
            <span className='text-muted-foreground'>
              {total > 0 ? total : t('Unlimited')}
            </span>
          )
        },
        size: 100,
      },
      {
        id: 'actions',
        cell: ({ row }) => <DataTableRowActions row={row} />,
        size: 80,
      },
    ],
    [t]
  )
}
