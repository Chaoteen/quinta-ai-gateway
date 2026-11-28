import { z } from 'zod'

const grantSchema = z.object({
  scope: z.string(),
  view: z.boolean(),
  use: z.boolean(),
  create: z.boolean(),
  operate: z.boolean(),
  update: z.boolean(),
  delete: z.boolean(),
})
const grantsSchema = z.record(z.string(), grantSchema)
export const policySchema = z.object({
  delegate: z.boolean(),
  grants: grantsSchema,
})
export const scopedRoleSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1)
    .refine((name) => new TextEncoder().encode(name).length <= 128),
  active: z.boolean(),
  grants: grantsSchema,
})
export const memberSchema = z.object({
  target: z
    .string()
    .regex(/^\d+$/)
    .refine(
      (value) => Number.isSafeInteger(Number(value)) && Number(value) > 0
    ),
  roleId: z.string().regex(/^\d*$/),
})
export type PolicyValues = z.infer<typeof policySchema>
export type ScopedRoleValues = z.infer<typeof scopedRoleSchema>
export type MemberValues = z.infer<typeof memberSchema>
