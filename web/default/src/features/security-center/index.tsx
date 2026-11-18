import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SectionPageLayout } from '@/components/layout'
import { ResetAudits } from './reset-audits'

type Readiness = {
  accounts: {
    id: number
    username: string
    role_key: string
    has_totp: boolean
    has_passkey: boolean
  }[]
  enrollment_complete: boolean
  truncated: boolean
  evaluated_at: number
}
export function SecurityCenter() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['admin-security-readiness'],
    retry: false,
    queryFn: async (): Promise<Readiness> => {
      // Keep service failures on this page so stale readiness is cleared.
      // Authentication and permission errors still use the shared handler.
      const r = await api.get('/api/platform/security/readiness', {
        validateStatus: (status: number) =>
          (status >= 200 && status < 300) || status >= 500,
        skipBusinessError: true,
      } as Record<string, unknown>)
      if (r.status >= 500 || !r.data.success)
        throw new Error('Readiness unavailable')
      return r.data.data
    },
  })
  const data = query.isError ? undefined : query.data
  const steps = [
    'Sign in as the platform administrator. A normal test account cannot enroll MFA for the administrator.',
    'Open Profile, enable two-factor authentication, scan the QR code with your authenticator and verify a code. Never send the QR code, secret or current code to support.',
    'Save the recovery codes in a password manager or secure offline location, separate from the phone and this server. Do not send them in chat or put them in GitHub.',
    'Keep the current session open. In a separate private browser window, test a new login with password and authenticator. Confirm access to Suppliers without closing the working session.',
    'Check recovery with a saved recovery code in a separate session before enforcement, then record that the code was consumed. If regenerating codes, replace all old copies securely.',
    'Confirm an independent server recovery path with your operator. Server recovery must require verified ownership and audit; it must not expose a public bypass URL.',
  ]
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Administrator security')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' onClick={() => query.refetch()}>
          {t('Refresh')}
        </Button>
        <Button render={<Link to='/profile' />}>
          {t('Open security settings')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-5'>
          <ResetAudits />
          <p className='rounded-lg border p-4 text-sm'>
            {t(
              'This page checks authenticator enrollment only. Recovery and global MFA enforcement are not enabled or certified by this check.'
            )}
          </p>
          {query.isPending && <p>{t('Loading…')}</p>}
          {query.isError && (
            <p role='alert'>
              {t(
                'Security status could not be checked. Retry; do not treat an unavailable check as ready.'
              )}
            </p>
          )}
          {data && (
            <>
              <p role='status' className='font-medium'>
                {t(
                  data.enrollment_complete
                    ? 'Administrator enrollment is complete. Recovery must still be tested.'
                    : 'Administrator enrollment is incomplete.'
                )}
              </p>
              <p className='text-muted-foreground text-sm'>
                {t('Checked at')}:{' '}
                {new Date(data.evaluated_at * 1000).toLocaleString()}
              </p>
              <div className='overflow-x-auto rounded-lg border'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      {[
                        'Account',
                        'Role',
                        'Authenticator',
                        'Passkey',
                        'Enrollment',
                      ].map((k) => (
                        <TableHead key={k}>{t(k)}</TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.accounts.map((a) => (
                      <TableRow key={a.id}>
                        <TableCell>{a.username}</TableCell>
                        <TableCell>{t(a.role_key)}</TableCell>
                        <TableCell>
                          {t(a.has_totp ? 'Enrolled' : 'Not enrolled')}
                        </TableCell>
                        <TableCell>
                          {t(a.has_passkey ? 'Enrolled' : 'Not enrolled')}
                        </TableCell>
                        <TableCell>
                          {t(
                            a.has_totp || a.has_passkey
                              ? 'Enrolled'
                              : 'Action required'
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {data.truncated && (
                <p role='alert'>
                  {t(
                    'More than 500 privileged accounts exist. This partial result cannot confirm complete enrollment.'
                  )}
                </p>
              )}
            </>
          )}
          <section className='space-y-3 rounded-lg border p-4'>
            <h3 className='font-semibold'>
              {t('Enrollment and recovery preparation')}
            </h3>
            <ol className='list-decimal space-y-3 pl-5 text-sm leading-relaxed'>
              {steps.map((step) => (
                <li key={step}>{t(step)}</li>
              ))}
            </ol>
          </section>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Use TOTP first for the current IP-based site. Passkey setup requires checking the production hostname and WebAuthn configuration. No recovery secrets are collected on this page.'
            )}
          </p>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
