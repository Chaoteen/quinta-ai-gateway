import { expect, test } from 'bun:test'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { runInNewContext } from 'node:vm'
import { createRsbuild } from '@rsbuild/core'
import configFactory from '../rsbuild.config.ts'

// Exercise the real production config: source-level tests cannot catch a
// missing define that erases import.meta.env in the shipped JavaScript.
test('production bundle shares the HTML mount path with router and API', async () => {
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'quinta-base-build-'))
  const previous = process.env.VITE_BASE_PATH
  try {
    const entry = path.join(temporary, 'probe.js')
    const modulePath = path.resolve('src/lib/base-path.ts').replaceAll('\\', '/')
    await writeFile(entry, `import { APP_BASE_PATH, withBasePath } from ${JSON.stringify(modulePath)};
globalThis.mountProbe = { basePath: APP_BASE_PATH, statusURL: withBasePath('/api/status') };`)
    for (const mount of ['', '/quinta']) {
      process.env.VITE_BASE_PATH = mount
      const config = await configFactory({ envMode: 'production', command: 'build' })
      const output = path.join(temporary, mount ? 'subpath' : 'root')
      const build = await createRsbuild({
        rsbuildConfig: {
          ...config,
          plugins: [],
          source: { ...config.source, entry: { probe: entry } },
          output: { ...config.output, distPath: { root: output }, minify: false },
          performance: { buildCache: false },
          tools: {},
        },
      })
      const result = await build.build()
      try {
        const html = await readFile(path.join(output, 'probe.html'), 'utf8')
        const asset = html.match(/src="([^\"]+\.js)"/)[1]
        expect(asset.startsWith(`${mount}/`)).toBe(true)
        const emitted = await readFile(path.join(output, asset.slice(mount.length)), 'utf8')
        const context = {}
        runInNewContext(emitted, context)
        expect(context.mountProbe.basePath).toBe(mount)
        expect(context.mountProbe.statusURL).toBe(`${mount}/api/status`)
      } finally {
        await result.close()
      }
    }
  } finally {
    if (previous === undefined) delete process.env.VITE_BASE_PATH
    else process.env.VITE_BASE_PATH = previous
    await rm(temporary, { recursive: true, force: true })
  }
}, 60000)
