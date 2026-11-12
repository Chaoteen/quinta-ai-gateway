import assert from 'node:assert/strict'
import { test } from 'node:test'
import { chatLinkRequiresApiKey, parseChatConfig } from './chat-links'

test('CC Switch is a key-dependent application, and invalid navigation targets are hidden', () => {
  assert.equal(chatLinkRequiresApiKey('ccswitch'), true)
  const items = parseChatConfig([
    { 'CC Switch': 'ccswitch' },
    { bad: 'javascript:alert(1)' },
    { relative: '/missing-client' },
    { client: 'aionui://provider/add?data={aionuiConfig}' },
  ])
  assert.deepEqual(
    items.map((item) => item.name),
    ['CC Switch', 'client']
  )
  assert.equal(items[1].id, '3') // filtering never changes persisted route identities
})
