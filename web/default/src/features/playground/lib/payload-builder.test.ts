import assert from 'node:assert/strict'
import { test } from 'node:test'
import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../constants'
import type { Message } from '../types'
import { buildChatCompletionPayload } from './payload-builder'

test('gateway errors are excluded from subsequent model context', () => {
  const messages: Message[] = [
    { key: 'user', from: 'user', versions: [{ id: '1', content: 'Hello' }] },
    {
      key: 'answer',
      from: 'assistant',
      status: 'complete',
      versions: [{ id: '2', content: 'Hi' }],
    },
    {
      key: 'error',
      from: 'assistant',
      status: 'error',
      restoredFromStorage: true,
      versions: [
        { id: '3', content: 'No available channel (request id: fixture)' },
      ],
    },
    {
      key: 'new-user',
      from: 'user',
      versions: [{ id: '4', content: 'Try again' }],
    },
  ]
  const payload = buildChatCompletionPayload(
    messages,
    { ...DEFAULT_CONFIG, model: 'available-model' },
    DEFAULT_PARAMETER_ENABLED
  )
  assert.equal(payload.model, 'available-model')
  assert.deepEqual(payload.messages, [
    { role: 'user', content: 'Hello' },
    { role: 'assistant', content: 'Hi' },
    { role: 'user', content: 'Try again' },
  ])
})
