import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableHeader,
  TableHead,
  TableRow,
  TableBody,
  TableCell,
} from '@/components/ui/table'

type Audit = {
  id: number
  actor_id: number
  target_id: number
  target_tenant_id: number
  credential_type: string
  created_at: number
}
export function ResetAudits() {
  const { t } = useTranslation()
  const [cursors, setCursors] = useState<number[]>([0])
  const before = cursors[cursors.length - 1]
  const query = useQuery({
    queryKey: ['credential-reset-audits', before],
    retry: false,
    queryFn: async () => {
      const r = await api.get('/api/platform/security/credential-resets', {
        params: before ? { before } : {},
        validateStatus: (s: number) => (s >= 200 && s < 300) || s >= 500,
        skipBusinessError: true,
      } as Record<string, unknown>)
      if (r.status >= 500 || !r.data.success)
        throw new Error('Audit unavailable')
      return r.data.data as { items: Audit[]; has_more: boolean }
    },
  })
  const data = query.isError ? undefined : query.data
  return (
    <section className='space-y-3 rounded-lg border p-4'>
      <h3 className='font-semibold'>{t('Credential reset audit')}</h3>
      <p className='text-muted-foreground text-sm'>
        {t('Local audit records are not independent tamper-proof storage.')}
      </p>
      <Button variant='outline' onClick={() => query.refetch()}>
        {t('Refresh')}
      </Button>
      {query.isPending && <p>{t('Loading…')}</p>}
      {query.isError && (
        <p role='alert'>{t('Audit unavailable. Retry to check records.')}</p>
      )}
      {data && (
        <>
          <div className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  {[
                    'ID',
                    'Operator ID',
                    'Target account ID',
                    'Tenant ID',
                    'Credential type',
                    'Time',
                  ].map((k) => (
                    <TableHead key={k}>{t(k)}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell>{a.id}</TableCell>
                    <TableCell>{a.actor_id}</TableCell>
                    <TableCell>{a.target_id}</TableCell>
                    <TableCell>{a.target_tenant_id}</TableCell>
                    <TableCell>{a.credential_type}</TableCell>
                    <TableCell>
                      {new Date(a.created_at * 1000).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {data.items.length === 0 && (
            <p>{t('No credential reset records.')}</p>
          )}
        </>
      )}
      <div className='flex gap-2'>
        <Button
          variant='outline'
          disabled={cursors.length === 1 || query.isFetching}
          onClick={() => setCursors((c) => c.slice(0, -1))}
        >
          {t('Previous')}
        </Button>
        <Button
          variant='outline'
          disabled={!data?.has_more || query.isFetching}
          onClick={() => {
            if (data?.items.length)
              setCursors((c) => [...c, data.items[data.items.length - 1].id])
          }}
        >
          {t('Next')}
        </Button>
      </div>
    </section>
  )
}
