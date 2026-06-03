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

import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { formatQuota, formatTimestamp } from '@/lib/format'
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
import { EmptyState } from '@/components/empty-state'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import { getAllBillingHistory } from '@/features/wallet/api'
import type { TopupRecord } from '@/features/wallet/types'

export function TopUpAdmin() {
  const { t } = useTranslation()
  const [items, setItems] = useState<TopupRecord[]>([])
  const [loading, setLoading] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getAllBillingHistory(1, 50)
      if (res.success && res.data) {
        setItems(res.data.items || [])
      } else {
        toast.error(res.message || t('Failed to load data'))
        setItems([])
      }
    } catch {
      toast.error(t('Failed to load data'))
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    loadData()
  }, [loadData])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('TopUp')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Read-only topup transaction records')}
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
                  <TableHead>ID</TableHead>
                  <TableHead>{t('User ID')}</TableHead>
                  <TableHead>{t('Trade No')}</TableHead>
                  <TableHead>{t('Amount')}</TableHead>
                  <TableHead>{t('Money')}</TableHead>
                  <TableHead>{t('Payment Method')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Created At')}</TableHead>
                  <TableHead>{t('Completed At')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>{item.id}</TableCell>
                    <TableCell>{item.user_id}</TableCell>
                    <TableCell className='font-mono'>{item.trade_no}</TableCell>
                    <TableCell>{formatQuota(item.amount)}</TableCell>
                    <TableCell>{item.money}</TableCell>
                    <TableCell>{item.payment_method || '-'}</TableCell>
                    <TableCell>
                      <StatusBadge
                        label={t(item.status)}
                        variant={item.status === 'success' ? 'success' : 'neutral'}
                        copyable={false}
                      />
                    </TableCell>
                    <TableCell>{formatTimestamp(item.create_time)}</TableCell>
                    <TableCell>
                      {item.complete_time ? formatTimestamp(item.complete_time) : '-'}
                    </TableCell>
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
