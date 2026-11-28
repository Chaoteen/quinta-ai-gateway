import { useTranslation } from 'react-i18next'
import type { Action, Grants } from '../types'

const actions: Action[] = [
  'view',
  'use',
  'create',
  'operate',
  'update',
  'delete',
]
const actionLabels = [
  'Visible',
  'Usable',
  'Create',
  'Operate',
  'Update',
  'Delete',
]
export function GrantsEditor({
  value,
  ceiling,
  onChange,
  disabled = false,
}: {
  value: Grants
  ceiling: Grants
  onChange: (v: Grants) => void
  disabled?: boolean
}) {
  const { t } = useTranslation()
  function toggle(resource: string, action: Action, checked: boolean) {
    const next = {
      ...(value[resource] || ceiling[resource]),
      [action]: checked,
    }
    if (action === 'view' && !checked)
      actions.forEach((a) => {
        next[a] = false
      })
    if (action !== 'view' && checked) next.view = true
    onChange({ ...value, [resource]: next })
  }
  return (
    <div className='overflow-x-auto rounded-lg border'>
      <table className='w-full text-sm'>
        <thead>
          <tr>
            <th className='p-3 text-start'>{t('Resource')}</th>
            <th className='p-3 text-start'>{t('Scope')}</th>
            {actionLabels.map((a) => (
              <th className='p-3 text-center' key={a}>
                {t(a)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Object.entries(ceiling).map(([resource, cap]) => (
            <tr className='border-t' key={resource}>
              <td className='p-3'>{t('rbac.resource.' + resource)}</td>
              <td className='p-3'>{t('rbac.scope.' + cap.scope)}</td>
              {actions.map((action, i) => (
                <td className='p-2 text-center' key={action}>
                  <input
                    type='checkbox'
                    className='accent-primary size-4'
                    aria-label={
                      t('rbac.resource.' + resource) +
                      ' / ' +
                      t(actionLabels[i])
                    }
                    disabled={disabled || !cap[action]}
                    checked={!!value[resource]?.[action] && cap[action]}
                    onChange={(e) => toggle(resource, action, e.target.checked)}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
