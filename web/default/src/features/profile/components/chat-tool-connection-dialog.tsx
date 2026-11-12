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
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { fetchActiveChatKey } from '@/features/chat/hooks/use-active-chat-key'
import { resolveChatUrl, type ChatPreset } from '@/features/chat/lib/chat-links'
import { getApiKeys } from '@/features/keys/api'
import { CCSwitchDialog } from '@/features/keys/components/dialogs/cc-switch-dialog'
import type { ApiKey } from '@/features/keys/types'
import { getUserModels } from '@/features/playground/api'

async function ownedKeys() {
  const keys: ApiKey[] = []
  for (let p = 1; ; p++) {
    const result = await getApiKeys({ p, size: 50 })
    if (!result.success) throw new Error('Failed to load API keys')
    const page = result.data?.items ?? []
    keys.push(...page)
    if (page.length < 50 || keys.length >= (result.data?.total ?? 0)) break
  }
  return keys
}
const usable = (key: ApiKey) =>
  key.status === 1 &&
  (key.expired_time === -1 || key.expired_time > Date.now() / 1000) &&
  (key.unlimited_quota || key.remain_quota > 0)

export function ChatToolConnectionDialog({
  preset,
  serverAddress,
  boundKeyId,
  onClose,
}: {
  preset: ChatPreset
  serverAddress: string
  boundKeyId?: number
  onClose: () => void
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const userId = useAuthStore((s) => s.auth.user?.id)
  const [keyId, setKeyId] = useState(String(boundKeyId ?? ''))
  const [busy, setBusy] = useState(false)
  const [ccKey, setCCKey] = useState('')
  const {
    data: keys = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ['chat-owned-keys', userId],
    queryFn: ownedKeys,
    staleTime: 0,
  })
  const selected = keys.find((key) => String(key.id) === keyId)
  const bound = keys.find((key) => key.id === boundKeyId)
  const ready = Boolean(bound && usable(bound))
  const userGroup = useAuthStore((s) => s.auth.user?.group)
  const group = bound?.group || userGroup || 'default'
  const { data: availableModels = [] } = useQuery({
    queryKey: ['chat-tool-models', userId, group],
    queryFn: () => getUserModels(group),
    enabled: Boolean(bound),
  })
  const models = availableModels
    .filter(
      (model) =>
        !bound?.model_limits_enabled ||
        (bound.model_limits ?? '').split(',').includes(model.value)
    )
    .map((model) => model.value)

  const save = async (create: boolean) => {
    setBusy(true)
    try {
      const response = await api.post('/api/user/self/chat-key', {
        app: preset.name,
        create,
        token_id: create ? 0 : Number(keyId),
      })
      if (!response.data.success) {
        toast.error(t(response.data.message || 'Failed to update settings'))
        return
      }
      const result = response.data.data
      const auth = useAuthStore.getState().auth
      const current = auth.user
      if (current) {
        const settings =
          typeof current.setting === 'string'
            ? JSON.parse(current.setting || '{}')
            : (current.setting ?? {})
        auth.setUser({
          ...current,
          setting: JSON.stringify({
            ...settings,
            enabled_chat_apps: result.enabled_chat_apps,
            chat_key_bindings: result.chat_key_bindings,
          }),
        })
      }
      setKeyId(String(result.token_id))
      await client.invalidateQueries({ queryKey: ['chat-owned-keys', userId] })
      await client.invalidateQueries({ queryKey: ['api-keys'] })
      toast.success(t('Gateway key bound and tool enabled'))
    } catch {
      toast.error(t('Failed to update settings'))
    } finally {
      setBusy(false)
    }
  }
  const useKey = async (copy: boolean) => {
    setBusy(true)
    try {
      const key = await fetchActiveChatKey(preset.name)
      if (copy) {
        await navigator.clipboard.writeText(key)
        toast.success(t('Copied'))
        return
      }
      if (preset.url === 'ccswitch' || preset.url.startsWith('ccswitch:')) {
        setCCKey(key)
        return
      }
      window.open(
        resolveChatUrl({ template: preset.url, apiKey: key, serverAddress }),
        '_blank',
        'noopener'
      )
    } catch (error) {
      toast.error(
        t(error instanceof Error ? error.message : 'Failed to load API key')
      )
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <Dialog
        open
        onOpenChange={(open) => {
          if (!open && !busy) onClose()
        }}
      >
        <DialogContent className='max-h-[90dvh] overflow-y-auto sm:max-w-xl'>
          <DialogHeader>
            <DialogTitle>
              {t('Configure {{tool}}', { tool: preset.name })}
            </DialogTitle>
            <DialogDescription>
              {t(
                'Use a key generated by this gateway. Create a dedicated key or select one you already own; no upstream API key is required here.'
              )}
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-5'>
            <div className='space-y-2 rounded-md border p-3'>
              <p className='text-sm'>
                {t(
                  'Create a dedicated key with a 30-day expiry and a quota limit equal to your current account balance. This does not add balance.'
                )}
              </p>
              <Button
                disabled={busy || isPending}
                onClick={() => void save(true)}
              >
                {t('Create dedicated key and enable')}
              </Button>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='chat-key-select'>
                {t('Or select an existing gateway key')}
              </Label>
              <select
                id='chat-key-select'
                className='border-input bg-background h-9 w-full rounded-md border px-3 text-sm'
                value={keyId}
                disabled={busy || isPending}
                onChange={(event) => setKeyId(event.target.value)}
              >
                <option value=''>{t('Select a gateway key')}</option>
                {keys.map((key) => (
                  <option key={key.id} value={key.id} disabled={!usable(key)}>
                    {key.name} · {key.key}
                    {!usable(key) ? ` · ${t('Unavailable')}` : ''}
                  </option>
                ))}
              </select>
              {isError && (
                <p role='alert' className='text-destructive text-sm'>
                  {t('Failed to load API keys')}
                </p>
              )}
              {!isPending && keys.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No gateway keys yet. Use the create button above.')}
                </p>
              )}
              <Button
                variant='outline'
                disabled={busy || !selected || !usable(selected)}
                onClick={() => void save(false)}
              >
                {t('Bind selected key and enable')}
              </Button>
            </div>
            <div className='space-y-3 border-t pt-4'>
              <p className='text-sm font-medium'>
                {t(ready ? 'Connection ready' : 'Complete key binding first')}
              </p>
              <Label htmlFor='chat-gateway-address'>
                {t('Gateway API address')}
              </Label>
              <Input
                id='chat-gateway-address'
                value={`${serverAddress.replace(/\/$/, '')}/v1`}
                readOnly
              />
              <Label htmlFor='chat-available-models'>
                {t('Available models')}
              </Label>
              <Input
                id='chat-available-models'
                value={models.join(', ')}
                readOnly
              />
              <Label htmlFor='chat-bound-key'>{t('Bound gateway key')}</Label>
              <Input
                id='chat-bound-key'
                value={bound ? `${bound.name} · ${bound.key}` : t('Not bound')}
                readOnly
              />
              <div className='flex flex-wrap gap-2'>
                <Button
                  disabled={busy || !ready}
                  onClick={() => void useKey(false)}
                >
                  {t('Open tool and import connection')}
                </Button>
                <Button
                  variant='outline'
                  disabled={busy || !ready}
                  onClick={() => void useKey(true)}
                >
                  {t('Copy gateway key')}
                </Button>
                <Button
                  variant='outline'
                  onClick={() =>
                    void navigator.clipboard
                      .writeText(`${serverAddress.replace(/\/$/, '')}/v1`)
                      .then(() => toast.success(t('Copied')))
                  }
                >
                  {t('Copy API address')}
                </Button>
              </div>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Install desktop clients first. Import only into tools you trust. You can also paste the copied address and key into the client manually.'
                )}
              </p>
            </div>
          </div>
        </DialogContent>
      </Dialog>
      <CCSwitchDialog
        open={Boolean(ccKey)}
        tokenKey={ccKey}
        onOpenChange={(open) => {
          if (!open) setCCKey('')
        }}
      />
    </>
  )
}
