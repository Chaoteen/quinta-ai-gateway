import { normalizeBasePath } from './deployment-path'

export { normalizeBasePath } from './deployment-path'

export const APP_BASE_PATH = normalizeBasePath(
  import.meta.env?.VITE_BASE_PATH ?? ''
)

export function withBasePath(path: string, basePath = APP_BASE_PATH): string {
  if (!path.startsWith('/') || path.startsWith('//')) {
    throw new Error('Expected an application-relative absolute path')
  }
  if (basePath && (path === basePath || path.startsWith(`${basePath}/`))) {
    return path
  }
  return `${basePath}${path}`
}
