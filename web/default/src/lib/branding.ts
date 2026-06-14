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
import { DEFAULT_SYSTEM_NAME } from './constants'

const LEGACY_SYSTEM_NAMES = new Set(['new api', 'newapi'])

export function normalizeSystemName(name?: string | null): string {
  const trimmed = name?.trim()
  if (!trimmed) return DEFAULT_SYSTEM_NAME
  if (LEGACY_SYSTEM_NAMES.has(trimmed.toLowerCase())) {
    return DEFAULT_SYSTEM_NAME
  }
  return trimmed
}

export function normalizeDocsLink(link?: string | null): string {
  const trimmed = link?.trim()
  if (!trimmed) return '/docs'

  try {
    const url = new URL(trimmed)
    if (url.hostname === 'docs.newapi.pro') return '/docs'
  } catch {
    // Relative links are handled below.
  }

  if (trimmed.includes('docs.newapi.pro')) return '/docs'
  return trimmed
}

export function isInternalLink(href: string): boolean {
  return href.startsWith('/')
}
