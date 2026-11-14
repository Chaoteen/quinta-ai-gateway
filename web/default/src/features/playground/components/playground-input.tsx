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
import { useState } from 'react'
import {
  GlobeIcon,
  SendIcon,
  SquareIcon,
  BarChartIcon,
  BoxIcon,
  NotepadTextIcon,
  CodeSquareIcon,
  GraduationCapIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  PromptInput,
  PromptInputButton,
  PromptInputFooter,
  PromptInputTextarea,
  PromptInputTools,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { Suggestion, Suggestions } from '@/components/ai-elements/suggestion'
import { ModelSelector, GroupSelector } from '@/components/model-group-selector'
import type { ModelOption, GroupOption } from '../types'

interface PlaygroundInputProps {
  webSearch: boolean
  onWebSearchChange: (enabled: boolean) => void
  onSubmit: (text: string) => void
  onStop?: () => void
  disabled?: boolean
  canSubmit?: boolean
  isGenerating?: boolean
  models: ModelOption[]
  modelValue: string
  onModelChange: (value: string) => void
  isModelLoading?: boolean
  groups: GroupOption[]
  groupValue: string
  onGroupChange: (value: string) => void
}

const suggestions = [
  {
    icon: BarChartIcon,
    text: 'How do I connect my application?',
    color: '#76d0eb',
  },
  { icon: BoxIcon, text: 'How do I redeem a voucher?', color: '#76d0eb' },
  {
    icon: NotepadTextIcon,
    text: 'How do I request an invoice?',
    color: '#ea8444',
  },
  { icon: CodeSquareIcon, text: 'Code', color: '#6c71ff' },
  { icon: GraduationCapIcon, text: 'Get advice', color: '#76d0eb' },
  { icon: null, text: 'More' },
]

export function PlaygroundInput({
  webSearch,
  onWebSearchChange,
  onSubmit,
  onStop,
  disabled,
  canSubmit = true,
  isGenerating,
  models,
  modelValue,
  onModelChange,
  isModelLoading = false,
  groups,
  groupValue,
  onGroupChange,
}: PlaygroundInputProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  const isInputDisabled = disabled || !canSubmit
  const isModelSelectDisabled =
    disabled || isModelLoading || models.length === 0
  const isGroupSelectDisabled = disabled || groups.length === 0

  const handleSubmit = (message: PromptInputMessage) => {
    if (!message.text?.trim() || isInputDisabled) return
    onSubmit(message.text)
    setText('')
  }

  const handleSuggestionClick = (suggestion: string) => {
    if (!isInputDisabled) setText(t(suggestion))
  }

  return (
    <div className='grid shrink-0 gap-4 px-1 md:pb-4'>
      <PromptInput groupClassName='rounded-xl' onSubmit={handleSubmit}>
        <PromptInputTextarea
          autoComplete='off'
          autoCorrect='off'
          autoCapitalize='off'
          spellCheck={false}
          className='px-5 md:text-base'
          disabled={isInputDisabled}
          onChange={(event) => setText(event.target.value)}
          placeholder={t('Ask anything')}
          value={text}
        />

        <PromptInputFooter className='flex-wrap gap-2 p-2.5'>
          <PromptInputTools>
            <PromptInputButton
              className='border font-medium'
              disabled={isInputDisabled}
              aria-pressed={webSearch}
              onClick={() => onWebSearchChange(!webSearch)}
              variant={webSearch ? 'secondary' : 'outline'}
            >
              <GlobeIcon size={16} />
              <span>{t(webSearch ? 'Web search on' : 'Web search off')}</span>
            </PromptInputButton>
          </PromptInputTools>

          <div className='flex items-center gap-1.5 md:gap-2'>
            <GroupSelector
              selectedGroup={groupValue}
              groups={groups}
              onGroupChange={onGroupChange}
              disabled={isGroupSelectDisabled}
            />
            <ModelSelector
              selectedModel={modelValue}
              models={models}
              onModelChange={onModelChange}
              disabled={isModelSelectDisabled}
            />

            {isGenerating && onStop ? (
              <PromptInputButton
                className='text-foreground font-medium'
                onClick={onStop}
                variant='secondary'
              >
                <SquareIcon className='fill-current' size={16} />
                <span className='hidden sm:inline'>{t('Stop')}</span>
                <span className='sr-only sm:hidden'>{t('Stop')}</span>
              </PromptInputButton>
            ) : (
              <PromptInputButton
                className='text-foreground font-medium'
                disabled={isInputDisabled || !text.trim()}
                type='submit'
                variant='secondary'
              >
                <SendIcon size={16} />
                <span className='hidden sm:inline'>{t('Send')}</span>
                <span className='sr-only sm:hidden'>{t('Send')}</span>
              </PromptInputButton>
            )}
          </div>
        </PromptInputFooter>
      </PromptInput>

      <p className='text-muted-foreground px-3 text-xs leading-relaxed'>
        {t(
          'Platform documents are retrieved automatically. Web search sends only your current question (up to 256 characters) to Bing for public snippets. Do not include credentials or private information.'
        )}
      </p>
      <Suggestions>
        {suggestions.map(({ icon: Icon, text, color }) => (
          <Suggestion
            className={`text-xs font-normal sm:text-sm ${
              text === 'More' ? 'hidden sm:flex' : ''
            }`}
            key={text}
            onClick={() => handleSuggestionClick(text)}
            suggestion={text}
            disabled={isInputDisabled}
          >
            {Icon && <Icon size={16} style={{ color }} />}
            {t(text)}
          </Suggestion>
        ))}
      </Suggestions>
    </div>
  )
}
