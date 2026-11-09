export function normalizeBasePath(value: string): string {
  const normalized = value.trim().replace(/\/+$/, '')
  if (normalized && !/^(\/[A-Za-z0-9_-]+)+$/.test(normalized)) {
    throw new Error(
      'VITE_BASE_PATH must be an absolute path with simple segments'
    )
  }
  return normalized
}
