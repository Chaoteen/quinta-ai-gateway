import { afterEach, expect, test } from 'bun:test'
import { useAuthStore } from '../src/stores/auth-store'
import { requirePathAllowed, requirePermission } from '../src/lib/route-guards'
import { RBAC_PERMISSION, ADMIN_SIDEBAR_PERMISSION_BY_URL } from '../src/lib/rbac'

afterEach(() => useAuthStore.getState().auth.reset())

test('ordinary users can open self logs without receiving administrative log access', () => {
  useAuthStore.getState().auth.setUser({ id: 7, username: 'fixture', role: 1, role_key: 'user' })
  for (const section of ['common', 'task', 'drawing']) {
    expect(() => requirePathAllowed(`/usage-logs/${section}`)).not.toThrow()
  }
  expect(ADMIN_SIDEBAR_PERMISSION_BY_URL['/usage-logs/common']).toBeUndefined()
  expect(() => requirePermission(RBAC_PERMISSION.USAGE_LOGS)).toThrow()
})

test('anonymous visitors cannot pass the log page access guard', () => {
  expect(() => requirePathAllowed('/usage-logs/common')).toThrow()
})
