import { describe, expect, test } from 'bun:test'
import { normalizeBasePath, withBasePath } from '../src/lib/base-path.ts'

describe('isolated path deployment', () => {
  test('normalizes root and nested mount paths', () => {
    expect(normalizeBasePath('/')).toBe('')
    expect(normalizeBasePath(' /quinta/ ')).toBe('/quinta')
    expect(normalizeBasePath('/apps/quinta')).toBe('/apps/quinta')
  })
  test('rejects origins, traversal and malformed mount paths', () => {
    for (const value of ['https://example.com', '//example.com', '/a/../b', '/a?b', '/a#b', '/a\\b']) {
      expect(() => normalizeBasePath(value)).toThrow()
    }
  })
  test('keeps API and redirects inside the application without double prefixes', () => {
    expect(withBasePath('/api/status', '/quinta')).toBe('/quinta/api/status')
    expect(withBasePath('/sign-in?redirect=%2Finvoices', '/quinta')).toBe('/quinta/sign-in?redirect=%2Finvoices')
    expect(withBasePath('/quinta/invoices', '/quinta')).toBe('/quinta/invoices')
    expect(withBasePath('/invoices', '')).toBe('/invoices')
    expect(() => withBasePath('//example.com', '/quinta')).toThrow()
  })
})
