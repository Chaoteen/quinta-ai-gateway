import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { isBusinessRole } from '@/lib/rbac'

// Server authorization is live on every request. This refresh updates the
// presentation after policy changes; it is never the authorization boundary.
export function RbacSync() {
  const actor = useAuthStore((state) => state.auth.user)
  const query = useQuery({
    queryKey: ['rbac-state', actor?.id],
    enabled: !!actor && isBusinessRole(actor),
    refetchInterval: 30000,
    refetchOnWindowFocus: true,
    queryFn: async () => {
      const { data } = await api.get('/api/rbac/self')
      if (!data.success) throw new Error('Authorization unavailable')
      return data.data as {
        grants: AuthUser['resource_permissions']
        can_delegate: boolean
        role_id: number
      }
    },
  })
  useEffect(() => {
    const current = useAuthStore.getState().auth.user
    if (!current || current.id !== actor?.id || !query.data) return
    const fields = {
      resource_permissions: query.data.grants,
      can_delegate_roles: query.data.can_delegate,
      scoped_role_id: query.data.role_id,
    }
    if (
      JSON.stringify([
        current.resource_permissions,
        current.can_delegate_roles,
        current.scoped_role_id,
      ]) !==
      JSON.stringify([
        fields.resource_permissions,
        fields.can_delegate_roles,
        fields.scoped_role_id,
      ])
    ) {
      useAuthStore.getState().auth.setUser({ ...current, ...fields })
    }
  }, [actor?.id, query.data])
  return null
}
