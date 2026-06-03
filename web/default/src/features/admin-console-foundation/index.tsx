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

import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { formatTimestamp } from '@/lib/format'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatusBadge } from '@/components/status-badge'
import { EmptyState } from '@/components/empty-state'
import { LoadingState } from '@/components/loading-state'
import { getReadonlyResource } from './api'
import type { ReadonlyRecord, ReadonlyResource } from './types'

type Column = {
  key: string
  title: string
  render?: (record: ReadonlyRecord) => ReactNode
}

const RESOURCE_TITLES: Record<ReadonlyResource, string> = {
  tenants: 'Tenants',
  organizations: 'Organizations',
  departments: 'Departments',
  distribution_channels: 'Distribution Channels',
}

function numberValue(record: ReadonlyRecord, key: string): number {
  const value = (record as unknown as Record<string, unknown>)[key]
  return typeof value === 'number' ? value : 0
}

function textValue(record: ReadonlyRecord, key: string): string {
  const value = (record as unknown as Record<string, unknown>)[key]
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  return '-'
}

function useColumns(resource: ReadonlyResource): Column[] {
  const { t } = useTranslation()
  return useMemo(() => {
    const statusColumn: Column = {
      key: 'status',
      title: t('Status'),
      render: (record) => {
        const enabled = numberValue(record, 'status') === 1
        return (
          <StatusBadge
            label={enabled ? t('Enabled') : t('Disabled')}
            variant={enabled ? 'success' : 'neutral'}
            copyable={false}
          />
        )
      },
    }
    const createdAtColumn: Column = {
      key: 'created_at',
      title: t('Created At'),
      render: (record) => formatTimestamp(numberValue(record, 'created_at')),
    }
    const common: Column[] = [
      { key: 'id', title: 'ID' },
      { key: 'name', title: t('Name') },
      statusColumn,
    ]

    switch (resource) {
      case 'tenants':
        return [...common, createdAtColumn]
      case 'organizations':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'tenant_id', title: 'tenant_id' },
          statusColumn,
          createdAtColumn,
        ]
      case 'departments':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'tenant_id', title: 'tenant_id' },
          { key: 'organization_id', title: 'organization_id' },
          statusColumn,
          createdAtColumn,
        ]
      case 'distribution_channels':
        return [
          { key: 'id', title: 'ID' },
          { key: 'name', title: t('Name') },
          { key: 'tenant_id', title: 'tenant_id' },
          statusColumn,
          createdAtColumn,
        ]
    }
  }, [resource, t])
}

export function ReadonlyManagementPage(props: { resource: ReadonlyResource }) {
  const { t } = useTranslation()
  const [items, setItems] = useState<ReadonlyRecord[]>([])
  const [loading, setLoading] = useState(false)
  const columns = useColumns(props.resource)
  const title = t(RESOURCE_TITLES[props.resource])

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getReadonlyResource(props.resource)
      setItems(res.items || [])
    } catch {
      toast.error(t('Failed to load data'))
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [props.resource, t])

  useEffect(() => {
    loadData()
  }, [loadData])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{title}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Read-only admin console view')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Button variant='outline' size='sm' onClick={loadData} disabled={loading}>
          <RefreshCw className='h-4 w-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {loading ? (
          <LoadingState />
        ) : items.length === 0 ? (
          <EmptyState title={t('No Data')} />
        ) : (
          <div className='border-border overflow-x-auto rounded-md border'>
            <Table>
              <TableHeader>
                <TableRow>
                  {columns.map((column) => (
                    <TableHead key={column.key}>{column.title}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={`${props.resource}-${item.id}`}>
                    {columns.map((column) => (
                      <TableCell key={column.key} className='whitespace-nowrap'>
                        {column.render ? column.render(item) : textValue(item, column.key)}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
