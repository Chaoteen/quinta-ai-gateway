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
import { MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'
import {
  chatLinkRequiresApiKey,
  type ChatPreset,
} from '@/features/chat/lib/chat-links'
import { ChatToolConnectionDialog } from './chat-tool-connection-dialog'

export function ChatToolsCard() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const { allChatPresets, chatPresets, keyBindings, serverAddress } =
    useChatPresets()
  const [saving, setSaving] = useState(false)
  const [selected, setSelected] = useState<ChatPreset | null>(null)
  const toggle = async (preset: ChatPreset, checked: boolean) => {
    if (
      checked &&
      chatLinkRequiresApiKey(preset.url) &&
      !keyBindings[preset.name]
    ) {
      setSelected(preset)
      return
    }
    const enabled = chatPresets.map((p) => p.name)
    setSaving(true)
    try {
      const response = await api.put('/api/user/self/chat-preferences', {
        enabled_chat_apps: checked
          ? [...enabled, preset.name]
          : enabled.filter((name) => name !== preset.name),
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
              'Configure each tool here: create or select a gateway key, bind it, then import the connection. Do not enter an upstream provider key.'
            )}
          </p>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Desktop tools require installation. Web tools are provided by third parties. Importing a connection shares your gateway API key with the selected tool.'
            )}
          </p>
          <div className='divide-y'>
            {allChatPresets.map((preset) => {
              const checked = chatPresets.some((p) => p.name === preset.name)
              return (
                <div
                  key={preset.id}
                  className='flex min-h-20 flex-wrap items-center justify-between gap-3 py-3'
                >
                  <div className='min-w-0 flex-1'>
                    <label
                      htmlFor={`chat-tool-${preset.id}`}
                      className='block text-sm font-medium break-words'
                    >
                      {preset.name}
                    </label>
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        keyBindings[preset.name]
                          ? 'Gateway key bound'
                          : 'Connection not configured'
                      )}
                    </p>
                  </div>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => setSelected(preset)}
                  >
                    {t('Configure connection')}
                  </Button>
                  <Switch
                    id={`chat-tool-${preset.id}`}
                    aria-label={preset.name}
                    checked={checked}
                    disabled={saving}
                    onCheckedChange={(value) => void toggle(preset, value)}
                  />
                </div>
              )
            })}
          </div>
        </div>
      </TitledCard>
      {selected && (
        <ChatToolConnectionDialog
          preset={selected}
          serverAddress={serverAddress}
          boundKeyId={keyBindings[selected.name]}
          onClose={() => setSelected(null)}
        />
      )}
    </section>
  )
}
