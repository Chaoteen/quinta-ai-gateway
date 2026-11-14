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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { Button } from '@/components/ui/button'
import { getUserModels, getUserGroups } from './api'
import { PlaygroundChat } from './components/playground-chat'
import { PlaygroundInput } from './components/playground-input'
import { usePlaygroundState, useChatHandler } from './hooks'
import { createUserMessage, createLoadingAssistantMessage } from './lib'
import type { Message as MessageType } from './types'

export function Playground() {
  const { t } = useTranslation()
  const [webSearch, setWebSearch] = useState(false)
  const userId = useAuthStore((state) => state.auth.user?.id)
  const {
    config,
    parameterEnabled,
    messages,
    models,
    groups,
    updateMessages,
    setModels,
    setGroups,
    updateConfig,
  } = usePlaygroundState(userId)

  const { sendChat, stopGeneration, isGenerating, isRetrieving } =
    useChatHandler({
      webSearch,
      config,
      parameterEnabled,
      onMessageUpdate: updateMessages,
    })

  // Edit dialog state
  const [editingMessageKey, setEditingMessageKey] = useState<string | null>(
    null
  )

  // Load models
  const {
    data: modelsData,
    isFetching: isLoadingModels,
    isError: isModelsError,
  } = useQuery({
    queryKey: ['playground-models', userId, config.group],
    queryFn: () => getUserModels(config.group),
  })

  // Load groups
  const { data: groupsData } = useQuery({
    queryKey: ['playground-groups', userId],
    queryFn: getUserGroups,
  })

  // Update models when data changes
  useEffect(() => {
    if (!modelsData) return

    setModels(modelsData)

    // Set default model if current model is not available
    const isCurrentModelValid = modelsData.some((m) => m.value === config.model)
    if (!isCurrentModelValid) {
      updateConfig('model', modelsData[0]?.value ?? '')
    }
  }, [modelsData, config.model, setModels, updateConfig])

  // Update groups when data changes
  useEffect(() => {
    if (!groupsData) return

    setGroups(groupsData)

    const hasCurrentGroup = groupsData.some((g) => g.value === config.group)
    if (!hasCurrentGroup && groupsData.length > 0) {
      const fallback =
        groupsData.find((g) => g.value === 'default')?.value ??
        groupsData[0].value
      updateConfig('group', fallback)
    }
  }, [groupsData, setGroups, config.group, updateConfig])

  const canSend =
    !isLoadingModels &&
    !isModelsError &&
    Boolean(modelsData?.some((m) => m.value === config.model)) &&
    groups.some((g) => g.value === config.group)
  const checkReady = useCallback(() => {
    if (canSend && !isGenerating) return true
    if (!isGenerating)
      toast.info(
        t(
          'No available model in this group. Contact the administrator to configure an upstream channel.'
        )
      )
    return false
  }, [canSend, isGenerating, t])

  const handleSendMessage = (text: string) => {
    if (!checkReady()) return
    const userMessage = createUserMessage(text)
    const assistantMessage = createLoadingAssistantMessage()

    const newMessages = [...messages, userMessage, assistantMessage]
    updateMessages(newMessages)

    // Send chat request
    sendChat(newMessages)
  }

  const handleCopyMessage = (message: MessageType) => {
    // Copy is handled in MessageActions component
    // eslint-disable-next-line no-console
    console.log('Message copied:', message.key)
  }

  const handleRegenerateMessage = (message: MessageType) => {
    if (!checkReady()) return
    // Find the message index and regenerate from there
    const messageIndex = messages.findIndex((m) => m.key === message.key)
    if (messageIndex === -1) return

    // Remove messages after this one and regenerate
    const messagesUpToHere = messages.slice(0, messageIndex)
    const loadingMessage = createLoadingAssistantMessage()
    const newMessages = [...messagesUpToHere, loadingMessage]

    updateMessages(newMessages)
    sendChat(newMessages)
  }

  const handleEditMessage = useCallback((message: MessageType) => {
    setEditingMessageKey(message.key)
  }, [])

  const handleEditOpenChange = useCallback((open: boolean) => {
    if (!open) setEditingMessageKey(null)
  }, [])

  // Apply edit and optionally re-submit from the edited user message
  const applyEdit = useCallback(
    (newContent: string, submit: boolean) => {
      if (!editingMessageKey || (submit && !checkReady())) return
      const index = messages.findIndex((m) => m.key === editingMessageKey)
      if (index === -1) return

      const updated = messages.map((m) =>
        m.key === editingMessageKey
          ? { ...m, versions: [{ ...m.versions[0], content: newContent }] }
          : m
      )

      setEditingMessageKey(null)

      if (!submit || updated[index].from !== 'user') {
        updateMessages(updated)
        return
      }

      const toSubmit = [
        ...updated.slice(0, index + 1),
        createLoadingAssistantMessage(),
      ]
      updateMessages(toSubmit)
      sendChat(toSubmit)
    },
    [editingMessageKey, messages, updateMessages, sendChat, checkReady]
  )

  const handleDeleteMessage = (message: MessageType) => {
    const newMessages = messages.filter((m) => m.key !== message.key)
    updateMessages(newMessages)
  }

  const lastMessage = messages.at(-1)
  const canRetrySavedError =
    lastMessage?.restoredFromStorage &&
    lastMessage.from === 'assistant' &&
    lastMessage.status === 'error'

  return (
    <div className='relative flex size-full flex-col overflow-hidden'>
      <div className='flex flex-wrap items-center justify-between gap-2 border-b px-6 py-3'>
        <p className='text-sm font-medium'>
          {t('Online chat')} ·{' '}
          {t('Current model: {{model}} · Group: {{group}}', {
            model: config.model || t('No model selected'),
            group: config.group,
          })}
        </p>
        {canRetrySavedError && (
          <Button
            variant='outline'
            size='sm'
            disabled={!canSend || isGenerating}
            onClick={() => handleRegenerateMessage(lastMessage)}
          >
            {t('Retry with current model')}
          </Button>
        )}
      </div>
      <p role='status' className='text-muted-foreground px-6 py-3 text-sm'>
        {t(
          isLoadingModels
            ? 'Loading available models…'
            : isModelsError
              ? 'Failed to load models. Please refresh and try again.'
              : !canSend && !isGenerating
                ? 'No available model in this group. Contact the administrator to configure an upstream channel.'
                : 'Chat here with platform documentation. Model calls consume account credit.'
        )}
      </p>
      {isRetrieving && (
        <p role='status' className='text-muted-foreground px-6 text-sm'>
          {t('Retrieving platform documents and selected web sources…')}
        </p>
      )}
      {/* Full-width scroll container: scrolling works even over side whitespace */}
      <div className='flex flex-1 flex-col overflow-hidden'>
        <PlaygroundChat
          messages={messages}
          onCopyMessage={handleCopyMessage}
          onRegenerateMessage={handleRegenerateMessage}
          onEditMessage={handleEditMessage}
          onDeleteMessage={handleDeleteMessage}
          isGenerating={isGenerating}
          editingKey={editingMessageKey}
          onCancelEdit={handleEditOpenChange}
          onSaveEdit={(newContent) => applyEdit(newContent, false)}
          onSaveEditAndSubmit={(newContent) => applyEdit(newContent, true)}
        />
      </div>

      {/* Input area: center content and constrain to the same container width */}
      <div className='mx-auto w-full max-w-4xl'>
        <PlaygroundInput
          webSearch={webSearch}
          onWebSearchChange={setWebSearch}
          disabled={isGenerating}
          canSubmit={canSend}
          groups={groups}
          groupValue={config.group}
          isGenerating={isGenerating}
          isModelLoading={isLoadingModels}
          modelValue={config.model}
          models={models}
          onGroupChange={(value) => updateConfig('group', value)}
          onModelChange={(value) => updateConfig('model', value)}
          onStop={stopGeneration}
          onSubmit={handleSendMessage}
        />
      </div>
    </div>
  )
}
