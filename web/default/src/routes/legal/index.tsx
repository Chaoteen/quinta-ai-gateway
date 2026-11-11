import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useStatus } from '@/hooks/use-status'
import { PublicLayout } from '@/components/layout'

function LegalPage() {
  const { i18n } = useTranslation()
  const { status } = useStatus()
  const zh = i18n.language.startsWith('zh')
  const version = String(status?.version || '')
  const revision = /^[a-f0-9]{12,40}$/.test(version) ? version : 'main'
  const repo = 'https://github.com/Chaoteen/quinta-ai-gateway'
  return (
    <PublicLayout>
      <main className='mx-auto max-w-4xl space-y-6 px-4 py-24'>
        <h1 className='text-3xl font-semibold'>
          {zh ? '开源许可与对应源码' : 'License and corresponding source'}
        </h1>
        <p>
          {zh
            ? 'Quinta AI Gateway 基于 QuantumNous 的 new-api 进行修改，当前公开版本按 AGPLv3 提供。'
            : 'Quinta AI Gateway modifies QuantumNous new-api and is offered under AGPLv3.'}
        </p>
        <p>Frontend design and development by New API contributors.</p>
        <ul className='[&_a]:text-primary space-y-3 border-l-2 pl-6 [&_a]:underline'>
          <li>
            <a
              href={`${repo}/tree/${revision}`}
              target='_blank'
              rel='noopener noreferrer'
            >
              {zh
                ? '查看当前版本源码及构建说明'
                : 'Source and build instructions'}{' '}
              ({revision})
            </a>
          </li>
          <li>
            <a
              href={`${repo}/blob/${revision}/docs/product/build-from-source.md`}
              target='_blank'
              rel='noopener noreferrer'
            >
              {zh ? '构建与验证说明' : 'Build and verification instructions'}
            </a>
          </li>
          <li>
            <a
              href={`${repo}/blob/${revision}/LICENSE`}
              target='_blank'
              rel='noopener noreferrer'
            >
              AGPLv3 LICENSE
            </a>
          </li>
          <li>
            <a
              href={`${repo}/blob/${revision}/NOTICE`}
              target='_blank'
              rel='noopener noreferrer'
            >
              NOTICE
            </a>
          </li>
          <li>
            <a
              href={`${repo}/blob/${revision}/THIRD-PARTY-LICENSES.md`}
              target='_blank'
              rel='noopener noreferrer'
            >
              {zh ? '第三方许可与署名' : 'Third-party licenses and notices'}
            </a>
          </li>
          <li>
            <a
              href='https://github.com/QuantumNous/new-api'
              target='_blank'
              rel='noopener noreferrer'
            >
              QuantumNous / new-api
            </a>
          </li>
        </ul>
        <p className='text-muted-foreground'>
          {zh
            ? '当前未声明获得商业闭源授权。计划闭源交付时，需要另行核实上游及第三方授权。'
            : 'No proprietary commercial license is claimed. Closed-source delivery requires separate rights verification.'}
        </p>
      </main>
    </PublicLayout>
  )
}

export const Route = createFileRoute('/legal/')({ component: LegalPage })
