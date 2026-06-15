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
import axios from 'axios'
import i18next from 'i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

// ============================================================================
// Axios Instance Configuration
// ============================================================================

// Base URL: empty string for same-origin API requests
const baseURL = ''

// Create axios instance with default config
export const api = axios.create({
  baseURL,
  withCredentials: true, // Include cookies in cross-origin requests
  headers: {
    'Cache-Control': 'no-store', // Prevent caching
  },
})

// ============================================================================
// Request Deduplication
// ============================================================================

// Deduplicate concurrent GET requests to the same URL
// Prevents multiple identical requests from being sent simultaneously
const inFlightGet = new Map<string, Promise<unknown>>()
const originalGet = api.get.bind(api)

api.get = ((url: string, config = {}) => {
  const disableDuplicate = (config as unknown as Record<string, unknown>)
    ?.disableDuplicate
  if (disableDuplicate) return originalGet(url, config)

  const params = (config as unknown as Record<string, unknown>)?.params
    ? JSON.stringify((config as unknown as Record<string, unknown>).params)
    : '{}'
  const key = `${url}?${params}`

  // Return existing in-flight request if available
  if (inFlightGet.has(key)) return inFlightGet.get(key)!

  // Create new request and clean up after completion
  const req = originalGet(url, config).finally(() => inFlightGet.delete(key))
  inFlightGet.set(key, req)
  return req
}) as typeof api.get

// ============================================================================
// Response Interceptor
// ============================================================================

function navigateToErrorPage(path: string) {
  if (typeof window === 'undefined') return
  if (window.location.pathname === path) return
  window.location.assign(path)
}

function navigateToSignIn() {
  if (typeof window === 'undefined') return
  const current = `${window.location.pathname}${window.location.search}`
  const target = `/sign-in?redirect=${encodeURIComponent(current)}`
  if (window.location.pathname === '/sign-in') return
  window.location.assign(target)
}

// Handle business logic errors and HTTP errors globally
api.interceptors.response.use(
  (response) => {
    const skipBusiness = (response.config as unknown as Record<string, unknown>)
      ?.skipBusinessError

    // Unified business response format: { success, message, data }
    if (
      !skipBusiness &&
      response &&
      response.data &&
      typeof response.data.success === 'boolean'
    ) {
      if (!response.data.success) {
        // Show error toast for business failures
        const msg = response.data.message || 'Request failed'
        toast.error(msg)
      }
    }
    return response
  },
  (error) => {
    const skip = error?.config?.skipErrorHandler
    if (!skip) {
      const status = error?.response?.status

      if (status === 401) {
        // Unauthorized: clear auth state and show toast
        toast.error(i18next.t('Session expired!'))
        try {
          useAuthStore.getState().auth.reset()
        } catch {
          /* empty */
        }
        navigateToSignIn()
      } else if (status === 403) {
        toast.error(i18next.t("You don't have necessary permission"))
        navigateToErrorPage('/403')
      } else if (typeof status === 'number' && status >= 500) {
        const msg =
          error?.response?.data?.message || error?.message || 'Request error'
        toast.error(msg)
        navigateToErrorPage('/500')
      } else {
        // Other errors: show error message from response or default
        const msg =
          error?.response?.data?.message || error?.message || 'Request error'
        toast.error(msg)
      }
    }
    return Promise.reject(error)
  }
)

// ============================================================================
// Common Headers Utility
// ============================================================================

/**
 * Get user ID from localStorage
 */
function getUserId(): string | null {
  try {
    if (typeof window !== 'undefined') {
      return window.localStorage.getItem('uid')
    }
  } catch {
    /* empty */
  }
  return null
}

/**
 * Get common request headers (for both axios and SSE requests)
 */
export function getCommonHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const uid = getUserId()
  if (uid) {
    headers['New-Api-User'] = uid
  }

  return headers
}

// ============================================================================
// Request Interceptor
// ============================================================================

// Attach user ID header for all requests
api.interceptors.request.use((config) => {
  const uid = getUserId()
  if (uid) {
    // Custom header for user identification
    ;(config.headers as Record<string, string>)['New-Api-User'] = uid
  }
  return config
})

// ============================================================================
// Common API Functions
// ============================================================================

// ----------------------------------------------------------------------------
// User APIs
// ----------------------------------------------------------------------------

// Get current user info
export async function getSelf() {
  const res = await api.get('/api/user/self', {
    // Avoid global 401 toast during guards/preloads
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

// Get user available models
export async function getUserModels(): Promise<{
  success: boolean
  message?: string
  data?: string[]
}> {
  const res = await api.get('/api/user/models')
  return res.data
}

// Get user groups with descriptions and ratios
export async function getUserGroups(): Promise<{
  success: boolean
  message?: string
  data?: Record<string, { desc: string; ratio: number | string }>
}> {
  const res = await api.get('/api/user/self/groups')
  return res.data
}

// ----------------------------------------------------------------------------
// System APIs
// ----------------------------------------------------------------------------

const STATUS_CACHE_KEY = 'status'
const STATUS_CACHE_META_KEY = 'status_cache_meta'
const STATUS_CACHE_TTL_MS = 5 * 60 * 1000
const NOTICE_CACHE_KEY = 'notice_cache'
const NOTICE_CACHE_META_KEY = 'notice_cache_meta'
const NOTICE_CACHE_TTL_MS = 5 * 60 * 1000

let statusRequest: Promise<Record<string, unknown>> | null = null
let noticeRequest:
  | Promise<{
      success: boolean
      message?: string
      data?: string
    }>
  | null = null

function getCachedJson<T>(key: string, metaKey: string, ttlMs: number): T | null {
  try {
    if (typeof window === 'undefined') return null
    const meta = window.localStorage.getItem(metaKey)
    const saved = window.localStorage.getItem(key)
    if (!meta || !saved) return null
    const timestamp = Number(meta)
    if (!Number.isFinite(timestamp) || Date.now() - timestamp > ttlMs) {
      return null
    }
    return JSON.parse(saved) as T
  } catch {
    return null
  }
}

function setCachedJson(key: string, metaKey: string, value: unknown): void {
  try {
    if (typeof window === 'undefined') return
    window.localStorage.setItem(key, JSON.stringify(value))
    window.localStorage.setItem(metaKey, String(Date.now()))
  } catch {
    /* empty */
  }
}

// Get system status
export async function getStatus() {
  const cached = getCachedJson<Record<string, unknown>>(
    STATUS_CACHE_KEY,
    STATUS_CACHE_META_KEY,
    STATUS_CACHE_TTL_MS
  )
  if (cached) return cached

  if (!statusRequest) {
    statusRequest = api
      .get('/api/status', {
        disableDuplicate: true,
        skipErrorHandler: true,
      } as Record<string, unknown>)
      .then((res) => {
        const status = res.data?.data as Record<string, unknown>
        if (status) setCachedJson(STATUS_CACHE_KEY, STATUS_CACHE_META_KEY, status)
        return status
      })
      .catch((error) => {
        const fallback = getCachedJson<Record<string, unknown>>(
          STATUS_CACHE_KEY,
          STATUS_CACHE_META_KEY,
          Number.POSITIVE_INFINITY
        )
        if (fallback) return fallback
        throw error
      })
      .finally(() => {
        statusRequest = null
      })
  }

  return statusRequest
}

// Get system notice
export async function getNotice(): Promise<{
  success: boolean
  message?: string
  data?: string
}> {
  const cached = getCachedJson<{
    success: boolean
    message?: string
    data?: string
  }>(NOTICE_CACHE_KEY, NOTICE_CACHE_META_KEY, NOTICE_CACHE_TTL_MS)
  if (cached) return cached

  if (!noticeRequest) {
    noticeRequest = api
      .get('/api/notice', {
        disableDuplicate: true,
        skipErrorHandler: true,
      } as Record<string, unknown>)
      .then((res) => {
        const notice = res.data as {
          success: boolean
          message?: string
          data?: string
        }
        setCachedJson(NOTICE_CACHE_KEY, NOTICE_CACHE_META_KEY, notice)
        return notice
      })
      .catch((error) => {
        const fallback = getCachedJson<{
          success: boolean
          message?: string
          data?: string
        }>(NOTICE_CACHE_KEY, NOTICE_CACHE_META_KEY, Number.POSITIVE_INFINITY)
        if (fallback) return fallback
        throw error
      })
      .finally(() => {
        noticeRequest = null
      })
  }

  return noticeRequest
}

// ----------------------------------------------------------------------------
// 2FA Management APIs
// ----------------------------------------------------------------------------

// Get 2FA status
export async function get2FAStatus() {
  const res = await api.get('/api/user/2fa/status')
  return res.data
}

// Setup 2FA
export async function setup2FA() {
  const res = await api.post('/api/user/2fa/setup')
  return res.data
}

// Enable 2FA with verification code
export async function enable2FA(code: string) {
  const res = await api.post('/api/user/2fa/enable', { code })
  return res.data
}

// Disable 2FA with verification code
export async function disable2FA(code: string) {
  const res = await api.post('/api/user/2fa/disable', { code })
  return res.data
}

// Regenerate 2FA backup codes
export async function regenerate2FABackupCodes(code: string) {
  const res = await api.post('/api/user/2fa/backup_codes', { code })
  return res.data
}
