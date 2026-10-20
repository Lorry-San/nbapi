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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  getGroups,
  getLightcodeModels,
  saveLightcodeModels,
} from '@/features/subscriptions/api'
import type { LightcodeGroupModel } from '@/features/subscriptions/types'
import { cn } from '@/lib/utils'

const EFFORTS = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max'] as const

const emptyModel = (id: number): LightcodeGroupModel => ({
  id,
  model: '',
  display_name: '',
  efforts: [],
  default_effort: '',
  context_window: 200000,
  max_output_tokens: 32000,
  vision: false,
  hidden: false,
  multiplier: 1,
})

/** Rows edit only display and policy; routing still comes from channel abilities. */
function ModelRow(props: {
  value: LightcodeGroupModel
  position: number
  onChange: (value: LightcodeGroupModel) => void
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
  canMoveUp: boolean
  canMoveDown: boolean
  dragging: boolean
  onDragStart: () => void
  onDragEnter: () => void
  onDragEnd: () => void
}) {
  const { t } = useTranslation()
  const toggleEffort = (effort: string, on: boolean) => {
    const efforts = on
      ? EFFORTS.filter((e) => e === effort || props.value.efforts.includes(e))
      : props.value.efforts.filter((e) => e !== effort)
    const fallback = efforts.includes(props.value.default_effort)
      ? props.value.default_effort
      : (efforts[0] ?? '')
    props.onChange({ ...props.value, efforts, default_effort: fallback })
  }
  return (
    <div
      draggable
      onDragStart={(event) => {
        event.dataTransfer.effectAllowed = 'move'
        // Some browsers cancel a drag without any payload.
        event.dataTransfer.setData('text/plain', props.value.model)
        props.onDragStart()
      }}
      onDragEnter={props.onDragEnter}
      onDragOver={(event) => {
        event.preventDefault()
        event.dataTransfer.dropEffect = 'move'
      }}
      onDragEnd={props.onDragEnd}
      aria-label={t('Model {{position}}: {{name}}', {
        position: props.position,
        name: props.value.display_name || props.value.model,
      })}
      className={cn(
        'flex flex-col gap-3 rounded-md border p-3',
        props.dragging && 'opacity-50'
      )}
    >
      <div className='flex items-center justify-between gap-2'>
        <span className='flex items-center gap-2 text-sm font-medium'>
          <span
            aria-hidden='true'
            className='text-muted-foreground cursor-grab select-none'
          >
            ⠿
          </span>
          <span className='text-muted-foreground'>#{props.position}</span>
          {props.value.display_name || props.value.model}
        </span>
        <span className='flex gap-1'>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Move up')}
            disabled={!props.canMoveUp}
            onClick={() => props.onMove(-1)}
          >
            ↑
          </Button>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Move down')}
            disabled={!props.canMoveDown}
            onClick={() => props.onMove(1)}
          >
            ↓
          </Button>
        </span>
      </div>
      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Model ID')}
          <Input
            value={props.value.model}
            onChange={(e) =>
              props.onChange({ ...props.value, model: e.target.value.trim() })
            }
          />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Display name in Lightcode')}
          <Input
            value={props.value.display_name}
            onChange={(e) =>
              props.onChange({ ...props.value, display_name: e.target.value })
            }
          />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Context window')}
          <Input
            type='number'
            min={1}
            value={props.value.context_window}
            onChange={(e) =>
              props.onChange({
                ...props.value,
                context_window: Number.parseInt(e.target.value, 10) || 0,
              })
            }
          />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Max output tokens')}
          <Input
            type='number'
            min={1}
            value={props.value.max_output_tokens}
            onChange={(e) =>
              props.onChange({
                ...props.value,
                max_output_tokens: Number.parseInt(e.target.value, 10) || 0,
              })
            }
          />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Consumption multiplier')}
          <Input
            type='number'
            min={0.01}
            max={100}
            step={0.1}
            value={props.value.multiplier}
            onChange={(e) =>
              props.onChange({
                ...props.value,
                multiplier: Number.parseFloat(e.target.value) || 0,
              })
            }
          />
          <span className='text-muted-foreground text-xs'>
            {t('Applies to Lightcode usage windows only')}
          </span>
        </label>
      </div>
      <fieldset className='flex flex-col gap-2'>
        <legend className='text-sm'>{t('Supported reasoning efforts')}</legend>
        <div className='flex flex-wrap gap-3'>
          {EFFORTS.map((effort) => (
            <label key={effort} className='flex items-center gap-2 text-sm'>
              <Switch
                checked={props.value.efforts.includes(effort)}
                onCheckedChange={(on) => toggleEffort(effort, on)}
                aria-label={effort}
              />
              {effort}
            </label>
          ))}
        </div>
      </fieldset>
      {props.value.efforts.length > 0 ? (
        <label className='flex flex-col gap-1 text-sm'>
          {t('Default reasoning effort')}
          <select
            className='border-input bg-background h-9 rounded-md border px-2'
            value={props.value.default_effort}
            onChange={(e) =>
              props.onChange({ ...props.value, default_effort: e.target.value })
            }
          >
            {props.value.efforts.map((effort) => (
              <option key={effort} value={effort}>
                {effort}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      <div className='flex flex-col gap-2'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex flex-wrap items-center gap-4'>
            <label className='flex items-center gap-2 text-sm'>
              <Switch
                checked={props.value.vision}
                onCheckedChange={(vision) =>
                  props.onChange({ ...props.value, vision })
                }
                aria-label={t('Image input')}
              />
              {t('Image input')}
            </label>
            <label className='flex items-center gap-2 text-sm'>
              <Switch
                checked={props.value.hidden}
                onCheckedChange={(hidden) =>
                  props.onChange({ ...props.value, hidden })
                }
                aria-label={t('Hide from Lightcode model list')}
              />
              {t('Hide from Lightcode model list')}
            </label>
          </div>
          <Button variant='outline' size='sm' onClick={props.onRemove}>
            {t('Remove')}
          </Button>
        </div>
        <p className='text-muted-foreground text-xs'>
          {t(
            'Hidden models remain available to explicitly configured internal routes.'
          )}
        </p>
      </div>
    </div>
  )
}

export function LightcodeGroupModels() {
  const { t } = useTranslation()
  const groups = useQuery({
    queryKey: ['lightcode', 'groups'],
    queryFn: getGroups,
  })
  const models = useQuery({
    queryKey: ['lightcode', 'models'],
    queryFn: getLightcodeModels,
  })
  const [group, setGroup] = useState('')
  const [rows, setRows] = useState<LightcodeGroupModel[]>([])
  const [saving, setSaving] = useState(false)
  // Index of the row being dragged; drop targets swap as the pointer enters them.
  const [draggingIndex, setDraggingIndex] = useState<number | null>(null)
  const nextRowId = useRef(-1)

  useEffect(() => {
    setRows((models.data?.data ?? []).filter((m) => m.routing_group === group))
  }, [group, models.data])

  const moveRow = (from: number, to: number) => {
    if (to < 0 || to >= rows.length || from === to) return
    const next = [...rows]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    setRows(next)
  }

  const save = async () => {
    setSaving(true)
    try {
      const res = await saveLightcodeModels(
        group,
        rows.map((r) => ({ ...r, display_name: r.display_name || r.model }))
      )
      if (res.success) {
        toast.success(t('Update succeeded'))
        void models.refetch()
      } else {
        toast.error(res.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setSaving(false)
    }
  }

  const groupOptions = (groups.data?.data ?? []).filter((g) => g !== 'auto')
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Lightcode models')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex max-w-3xl flex-col gap-4'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Choose a channel group, then list the models Lightcode shows for it. Subscriptions select which of these models they support.'
            )}
          </p>
          <label className='flex flex-col gap-1 text-sm'>
            {t('Channel group')}
            <select
              className='border-input bg-background h-9 rounded-md border px-2'
              value={group}
              onChange={(e) => setGroup(e.target.value)}
            >
              <option value=''>{t('Select a group')}</option>
              {groupOptions.map((g) => (
                <option key={g} value={g}>
                  {g}
                </option>
              ))}
            </select>
          </label>
          {group ? (
            <>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Drag a row by its handle, or use the arrows, to set the order Lightcode shows these models in.'
                )}
              </p>
              {rows.map((row, index) => (
                <ModelRow
                  key={row.id ?? `${row.model}-${index}`}
                  value={row}
                  position={index + 1}
                  dragging={draggingIndex === index}
                  canMoveUp={index > 0}
                  canMoveDown={index < rows.length - 1}
                  onChange={(value) =>
                    setRows(rows.map((r, i) => (i === index ? value : r)))
                  }
                  onRemove={() => setRows(rows.filter((_, i) => i !== index))}
                  onMove={(direction) => moveRow(index, index + direction)}
                  onDragStart={() => setDraggingIndex(index)}
                  onDragEnter={() => {
                    if (draggingIndex !== null && draggingIndex !== index) {
                      moveRow(draggingIndex, index)
                      setDraggingIndex(index)
                    }
                  }}
                  onDragEnd={() => setDraggingIndex(null)}
                />
              ))}
              <div className='flex gap-3'>
                <Button
                  variant='outline'
                  onClick={() =>
                    setRows([...rows, emptyModel(nextRowId.current--)])
                  }
                >
                  {t('Add model')}
                </Button>
                <Button
                  onClick={save}
                  disabled={saving || rows.some((r) => !r.model)}
                >
                  {t('Save')}
                </Button>
              </div>
            </>
          ) : null}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
