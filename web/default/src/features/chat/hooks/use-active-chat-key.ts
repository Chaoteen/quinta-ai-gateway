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
import { useQuery } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth-store'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import { API_KEY_STATUS } from '@/features/keys/constants'

export async function fetchActiveChatKey() {
  let activeId: number | undefined
  const size = 50
  for (let page = 1; ; page++) {
    const result = await getApiKeys({ p: page, size })
    if (!result.success) {
      throw new Error(result.message || 'Failed to load API keys')
    }
    const items = result.data?.items ?? []
    const now = Math.floor(Date.now() / 1000)
    const active = items.find(
      (item) =>
        item.status === API_KEY_STATUS.ENABLED &&
        (item.expired_time === -1 || item.expired_time > now) &&
        (item.unlimited_quota || item.remain_quota > 0)
    )
    if (active) {
      activeId = active.id
      break
    }
    if (items.length < size || page * size >= (result.data?.total ?? 0)) break
  }
  if (activeId === undefined) {
    throw new Error(
      'No usable API key found. Create a key, or check its status, expiry and quota in API Keys.'
    )
  }

  const keyResult = await fetchTokenKey(activeId)
  if (!keyResult.success || !keyResult.data?.key) {
    throw new Error(keyResult.message || 'Failed to load API key')
  }

  return keyResult.data.key.startsWith('sk-')
    ? keyResult.data.key
    : `sk-${keyResult.data.key}`
}

/**
 * Get the currently active API key for chat links
 */
export function useActiveChatKey(enabled: boolean) {
  const userId = useAuthStore((state) => state.auth.user?.id)

  return useQuery({
    queryKey: ['chat-active-key', userId],
    queryFn: fetchActiveChatKey,
    enabled: enabled && Boolean(userId),
    staleTime: 0,
    gcTime: 10 * 60 * 1000,
  })
}
