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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import { fetchUsableChatKeyId } from '@/features/chat/hooks/use-active-chat-key'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'
import { chatLinkRequiresApiKey } from '@/features/chat/lib/chat-links'

export function ChatToolsCard() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const { allChatPresets, chatPresets } = useChatPresets()
  const [saving, setSaving] = useState(false)
  const {
    data: keyId,
    isPending,
    isError,
  } = useQuery({
    queryKey: ['chat-usable-key-id', auth.user?.id],
    queryFn: async () => (await fetchUsableChatKeyId()) ?? null,
    enabled: Boolean(auth.user?.id),
    retry: false,
    staleTime: 0,
  })
  const toggle = async (name: string, checked: boolean) => {
    const selected = chatPresets.map((preset) => preset.name)
    const next = checked
      ? [...selected, name]
      : selected.filter((item) => item !== name)
    setSaving(true)
    try {
      const response = await api.put('/api/user/self/chat-preferences', {
        enabled_chat_apps: next,
      })
      if (!response.data.success) throw new Error('save failed')
      const current = useAuthStore.getState().auth.user
      if (current) {
        const settings =
          typeof current.setting === 'string'
            ? JSON.parse(current.setting || '{}')
            : (current.setting ?? {})
        auth.setUser({
          ...current,
          setting: JSON.stringify({
            ...settings,
            enabled_chat_apps: response.data.data.enabled_chat_apps,
          }),
        })
      }
      toast.success(t('Chat tool preferences saved'))
    } catch {
      toast.error(t('Failed to update settings'))
    } finally {
      setSaving(false)
    }
  }
  return (
    <section id='chat-tools' className='scroll-mt-6'>
      <TitledCard
        title={t('Chat tools')}
        icon={<MessageCircle className='h-4 w-4' />}
        description={t(
          'Only tools you enable appear in the chat menu. Your selection syncs with your account.'
        )}
      >
        <div className='space-y-4'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Desktop tools require installation. Web tools are provided by third parties. Importing a connection shares your gateway API key with the selected tool.'
            )}
          </p>
          {(!keyId || isError) && !isPending && (
            <div className='space-y-2 rounded-md border p-3 text-sm'>
              <p>
                {t(
                  isError
                    ? 'Unable to check API keys. Refresh to try again.'
                    : 'Create an enabled API key with available quota before activating a tool that needs one.'
                )}
              </p>
              <Button variant='outline' size='sm' render={<Link to='/keys' />}>
                {t('API Keys')}
              </Button>
            </div>
          )}
          <div className='divide-y'>
            {allChatPresets.map((preset) => {
              const checked = chatPresets.some(
                (item) => item.name === preset.name
              )
              const needsKey = chatLinkRequiresApiKey(preset.url)
              return (
                <div
                  key={preset.id}
                  className='flex min-h-16 items-center justify-between gap-4 py-3'
                >
                  <div className='min-w-0'>
                    <label
                      htmlFor={`chat-tool-${preset.id}`}
                      className='block text-sm font-medium break-words'
                    >
                      {preset.name}
                    </label>
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        preset.type === 'web'
                          ? 'Third-party web app'
                          : 'Installed desktop app'
                      )}
                    </p>
                  </div>
                  <Switch
                    id={`chat-tool-${preset.id}`}
                    aria-label={preset.name}
                    checked={checked}
                    disabled={
                      saving || (!checked && needsKey && (isPending || !keyId))
                    }
                    onCheckedChange={(value) => void toggle(preset.name, value)}
                  />
                </div>
              )
            })}
          </div>
          {allChatPresets.length === 0 && (
            <p className='text-muted-foreground text-sm'>
              {t('No supported chat tools configured.')}
            </p>
          )}
        </div>
      </TitledCard>
    </section>
  )
}
