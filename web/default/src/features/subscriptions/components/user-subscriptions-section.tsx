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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Ban, PauseCircle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/status-badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getUserRoleKey, ROLE_KEY } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import {
  getAdminUserSubscriptions,
  invalidateUserSubscription,
  renewUserSubscription,
  suspendUserSubscription,
} from '../api'
import { formatTimestamp } from '../lib'
import type { UserSubscriptionRecord } from '../types'

function canManageUserSubscriptions(roleKey: string) {
  return roleKey === ROLE_KEY.ROOT || roleKey === ROLE_KEY.TENANT_ADMIN
}

function lifecycleVariant(status: string) {
  return status === 'active' ? 'success' : 'neutral'
}

export function UserSubscriptionsSection() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const roleKey = getUserRoleKey(user)
  const canManage = canManageUserSubscriptions(roleKey)
  const [busyId, setBusyId] = useState<number | null>(null)
  const [refresh, setRefresh] = useState(0)

  const { data, isLoading } = useQuery({
    queryKey: ['admin-user-subscriptions', refresh],
    queryFn: async () => {
      const res = await getAdminUserSubscriptions({ page: 1, limit: 50 })
      return res.data || []
    },
    placeholderData: (prev) => prev,
  })

  const rows = useMemo(() => data || [], [data])

  const runAction = async (
    record: UserSubscriptionRecord,
    action: 'cancel' | 'suspend' | 'renew'
  ) => {
    const id = record.subscription.id
    setBusyId(id)
    try {
      const res =
        action === 'cancel'
          ? await invalidateUserSubscription(id)
          : action === 'suspend'
            ? await suspendUserSubscription(id)
            : await renewUserSubscription(id)
      if (res.success) {
        toast.success(t('Operation succeeded'))
        setRefresh((value) => value + 1)
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className='space-y-3'>
      <div>
        <h2 className='text-base font-semibold'>{t('User Subscriptions')}</h2>
        <p className='text-muted-foreground text-sm'>
          {t('Review assigned user subscriptions and lifecycle state')}
        </p>
      </div>
      <div className='rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{t('User')}</TableHead>
              <TableHead>{t('Plan')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Validity')}</TableHead>
              <TableHead>{t('Token Quota')}</TableHead>
              <TableHead>{t('Request Quota')}</TableHead>
              {canManage && (
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={canManage ? 8 : 7} className='py-8 text-center'>
                  {t('Loading...')}
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={canManage ? 8 : 7} className='py-8 text-center'>
                  {t('No subscription records')}
                </TableCell>
              </TableRow>
            ) : (
              rows.map((record) => {
                const sub = record.subscription
                const status = sub.lifecycle_status
                return (
                  <TableRow key={sub.id}>
                    <TableCell>#{sub.id}</TableCell>
                    <TableCell>#{sub.user_id}</TableCell>
                    <TableCell>
                      <div className='max-w-[180px]'>
                        <div className='truncate font-medium'>
                          {sub.plan_name || `#${sub.plan_id}`}
                        </div>
                        {sub.plan_code && (
                          <div className='text-muted-foreground truncate text-xs'>
                            {sub.plan_code}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        label={t(status)}
                        variant={lifecycleVariant(status)}
                        copyable={false}
                      />
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {formatTimestamp(sub.start_time)} -{' '}
                      {formatTimestamp(sub.end_time)}
                    </TableCell>
                    <TableCell>
                      {sub.token_quota_snapshot || 0}
                    </TableCell>
                    <TableCell>{sub.request_quota_snapshot || 0}</TableCell>
                    {canManage && (
                      <TableCell>
                        <div className='flex justify-end gap-1'>
                          <Button
                            variant='ghost'
                            size='icon'
                            disabled={busyId === sub.id || status === 'cancelled'}
                            onClick={() => runAction(record, 'cancel')}
                            title={t('Cancel')}
                          >
                            <Ban className='h-4 w-4' />
                          </Button>
                          <Button
                            variant='ghost'
                            size='icon'
                            disabled={busyId === sub.id || status === 'suspended'}
                            onClick={() => runAction(record, 'suspend')}
                            title={t('Suspend')}
                          >
                            <PauseCircle className='h-4 w-4' />
                          </Button>
                          <Button
                            variant='ghost'
                            size='icon'
                            disabled={busyId === sub.id}
                            onClick={() => runAction(record, 'renew')}
                            title={t('Renew')}
                          >
                            <RefreshCw className='h-4 w-4' />
                          </Button>
                        </div>
                      </TableCell>
                    )}
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
