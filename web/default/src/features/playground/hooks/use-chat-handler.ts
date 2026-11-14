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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { withBasePath } from '@/lib/base-path'
import { getAssistantContext, sendChatCompletion } from '../api'
import { MESSAGE_STATUS, ERROR_MESSAGES } from '../constants'
import {
  buildChatCompletionPayload,
  updateAssistantMessageWithError,
  updateLastAssistantMessage,
  processStreamingContent,
  finalizeMessage,
} from '../lib'
import type { Message, PlaygroundConfig, ParameterEnabled } from '../types'
import { useStreamRequest } from './use-stream-request'

interface UseChatHandlerOptions {
  webSearch: boolean
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  onMessageUpdate: (updater: (prev: Message[]) => Message[]) => void
}

/**
 * Hook for handling chat message sending and receiving
 */
export function useChatHandler({
  webSearch,
  config,
  parameterEnabled,
  onMessageUpdate,
}: UseChatHandlerOptions) {
  const { t } = useTranslation()
  const { sendStreamRequest, stopStream } = useStreamRequest()
  const [isGenerating, setIsGenerating] = useState(false)
  const [isRetrieving, setIsRetrieving] = useState(false)
  const requestRef = useRef<AbortController | null>(null)
  useEffect(
    () => () => {
      requestRef.current?.abort()
      stopStream()
    },
    [stopStream]
  )

  // Handle stream update
  const handleStreamUpdate = useCallback(
    (type: 'reasoning' | 'content', chunk: string) => {
      onMessageUpdate((prev) =>
        updateLastAssistantMessage(prev, (message) => {
          if (message.status === MESSAGE_STATUS.ERROR) return message

          if (type === 'reasoning') {
            // Direct API reasoning_content
            return {
              ...message,
              reasoning: {
                content: (message.reasoning?.content || '') + chunk,
                duration: 0,
              },
              isReasoningStreaming: true,
              status: MESSAGE_STATUS.STREAMING,
            }
          }

          // Content streaming: handle <think> tags
          return {
            ...processStreamingContent(message, chunk),
            status: MESSAGE_STATUS.STREAMING,
          }
        })
      )
    },
    [onMessageUpdate]
  )

  // Handle stream complete
  const handleStreamComplete = useCallback(() => {
    setIsGenerating(false)
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) =>
        message.status === MESSAGE_STATUS.COMPLETE ||
        message.status === MESSAGE_STATUS.ERROR
          ? message
          : { ...finalizeMessage(message), status: MESSAGE_STATUS.COMPLETE }
      )
    )
  }, [onMessageUpdate])

  // Handle stream error
  const handleStreamError = useCallback(
    (error: string, errorCode?: string) => {
      setIsGenerating(false)
      toast.error(error)
      onMessageUpdate((prev) =>
        updateAssistantMessageWithError(prev, error, errorCode)
      )
    },
    [onMessageUpdate]
  )

  // Send streaming chat request
  const sendStreamingChat = useCallback(
    (messages: Message[], context: string) => {
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )
      payload.messages.unshift({ role: 'system', content: context })
      sendStreamRequest(
        payload,
        handleStreamUpdate,
        handleStreamComplete,
        handleStreamError
      )
    },
    [
      config,
      parameterEnabled,
      sendStreamRequest,
      handleStreamUpdate,
      handleStreamComplete,
      handleStreamError,
    ]
  )

  // Send non-streaming chat request
  const sendNonStreamingChat = useCallback(
    async (messages: Message[], context: string, signal: AbortSignal) => {
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )

      payload.messages.unshift({ role: 'system', content: context })
      try {
        const response = await sendChatCompletion(payload, signal)
        if (signal.aborted) return
        setIsGenerating(false)
        const choice = response.choices?.[0]
        if (!choice) throw new Error(t('The model returned no answer.'))

        onMessageUpdate((prev) =>
          updateLastAssistantMessage(prev, (message) => ({
            ...finalizeMessage(
              {
                ...message,
                versions: [
                  {
                    ...message.versions[0],
                    content: choice.message?.content || '',
                  },
                ],
              },
              choice.message?.reasoning_content
            ),
            status: MESSAGE_STATUS.COMPLETE,
          }))
        )
      } catch (error: unknown) {
        if (signal.aborted) return
        const err = error as {
          response?: {
            data?: { message?: string; error?: { code?: string } }
          }
          message?: string
        }
        handleStreamError(
          err?.response?.data?.message ||
            err?.message ||
            ERROR_MESSAGES.API_REQUEST_ERROR,
          err?.response?.data?.error?.code || undefined
        )
      }
    },
    [config, parameterEnabled, onMessageUpdate, handleStreamError, t]
  )

  // Retrieve public references before generation; no upstream credentials reach the browser.
  const sendChat = useCallback(
    async (messages: Message[]) => {
      requestRef.current?.abort()
      const request = new AbortController()
      requestRef.current = request
      setIsGenerating(true)
      setIsRetrieving(true)
      try {
        const query =
          [...messages].reverse().find((message) => message.from === 'user')
            ?.versions[0]?.content ?? ''
        const result = await getAssistantContext(
          query,
          webSearch,
          request.signal
        )
        if (request.signal.aborted) return
        onMessageUpdate((prev) =>
          updateLastAssistantMessage(prev, (message) => ({
            ...message,
            sources: result.sources.map((source) => ({
              ...source,
              href: source.href.startsWith('/')
                ? withBasePath(source.href)
                : source.href,
            })),
          }))
        )
        if (result.warning)
          toast.info(
            t('No web results were found. Try a more specific question.')
          )
        setIsRetrieving(false)
        const links: Record<string, string> = {
          __GATEWAY_BASE_URL__: `${window.location.origin}${withBasePath('/v1')}`,
          __REGISTER_URL__: withBasePath('/sign-up'),
          __KEYS_URL__: withBasePath('/keys'),
          __BILLING_URL__: withBasePath('/billing'),
          __MODELS_URL__: withBasePath('/pricing'),
          __LOGS_URL__: withBasePath('/usage-logs/common'),
          __INVOICE_URL__: withBasePath('/invoices'),
          __API_GUIDE_URL__: withBasePath('/docs/api-access'),
        }
        const context = result.context
          .replaceAll('链接：/docs/', `链接：${withBasePath('/docs/')}`)
          .replace(/__[A-Z_]+__/g, (key) => links[key] ?? key)
        if (config.stream) sendStreamingChat(messages, context)
        else await sendNonStreamingChat(messages, context, request.signal)
      } catch (error: unknown) {
        if (request.signal.aborted) return
        const failure = error as {
          response?: { data?: { message?: string } }
          message?: string
        }
        handleStreamError(
          failure.response?.data?.message ||
            failure.message ||
            t('Could not retrieve sources')
        )
      } finally {
        if (!request.signal.aborted) setIsRetrieving(false)
      }
    },
    [
      webSearch,
      config.stream,
      sendStreamingChat,
      sendNonStreamingChat,
      onMessageUpdate,
      handleStreamError,
      t,
    ]
  )

  // Stop generation
  const stopGeneration = useCallback(() => {
    requestRef.current?.abort()
    setIsGenerating(false)
    setIsRetrieving(false)
    stopStream()
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) =>
        message.status === MESSAGE_STATUS.LOADING ||
        message.status === MESSAGE_STATUS.STREAMING
          ? { ...finalizeMessage(message), status: MESSAGE_STATUS.COMPLETE }
          : message
      )
    )
  }, [stopStream, onMessageUpdate])

  return {
    sendChat,
    stopGeneration,
    isGenerating,
    isRetrieving,
  }
}
