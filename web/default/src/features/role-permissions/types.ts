export type Action = 'view' | 'use' | 'create' | 'operate' | 'update' | 'delete'
export type Grant = Record<Action, boolean> & { scope: string }
export type Grants = Record<string, Grant>
export type Policy = {
  family: string
  delegate: boolean
  version: number
  grants: Grants
  ceiling: Grants
}
export type Role = {
  id: number
  tenant_id: number
  family: string
  name: string
  active: boolean
  version: number
  grants: Grants
}
export type Workspace = {
  state: {
    family: string
    tenant_id: number
    role_id: number
    can_delegate: boolean
    grants: Grants
  }
  policies: Policy[]
  roles: Role[]
  audits: {
    id: number
    actor_id: number
    target_user_id: number
    role_id: number
    action: string
    created_at: number
  }[]
  truncated: boolean
}

export type WriteRoleConfiguration = (
  path: string,
  body: unknown,
  method?: 'post' | 'put' | 'delete'
) => void
