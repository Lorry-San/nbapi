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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { getGroups, getLightcodeModels, saveLightcodeModels } from '@/features/subscriptions/api'
import type { LightcodeGroupModel } from '@/features/subscriptions/types'

const EFFORTS = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max'] as const

const emptyModel = (): LightcodeGroupModel => ({
  model: '',
  display_name: '',
  efforts: [],
  default_effort: '',
  context_window: 200000,
  max_output_tokens: 32000,
  vision: false,
})

/** Rows edit only display and policy; routing still comes from channel abilities. */
function ModelRow(props: {
  value: LightcodeGroupModel
  onChange: (value: LightcodeGroupModel) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const toggleEffort = (effort: string, on: boolean) => {
    const efforts = on
      ? EFFORTS.filter((e) => e === effort || props.value.efforts.includes(e))
      : props.value.efforts.filter((e) => e !== effort)
    const fallback = efforts.includes(props.value.default_effort) ? props.value.default_effort : efforts[0] ?? ''
    props.onChange({ ...props.value, efforts, default_effort: fallback })
  }
  return (
    <div className='flex flex-col gap-3 rounded-md border p-3'>
      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Model ID')}
          <Input value={props.value.model} onChange={(e) => props.onChange({ ...props.value, model: e.target.value.trim() })} />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Display name in Lightcode')}
          <Input value={props.value.display_name} onChange={(e) => props.onChange({ ...props.value, display_name: e.target.value })} />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Context window')}
          <Input type='number' min={1} value={props.value.context_window} onChange={(e) => props.onChange({ ...props.value, context_window: Number.parseInt(e.target.value, 10) || 0 })} />
        </label>
        <label className='flex flex-col gap-1 text-sm'>
          {t('Max output tokens')}
          <Input type='number' min={1} value={props.value.max_output_tokens} onChange={(e) => props.onChange({ ...props.value, max_output_tokens: Number.parseInt(e.target.value, 10) || 0 })} />
        </label>
      </div>
      <fieldset className='flex flex-col gap-2'>
        <legend className='text-sm'>{t('Supported reasoning efforts')}</legend>
        <div className='flex flex-wrap gap-3'>
          {EFFORTS.map((effort) => (
            <label key={effort} className='flex items-center gap-2 text-sm'>
              <Switch checked={props.value.efforts.includes(effort)} onCheckedChange={(on) => toggleEffort(effort, on)} aria-label={effort} />
              {effort}
            </label>
          ))}
        </div>
      </fieldset>
      {props.value.efforts.length > 0 ? (
        <label className='flex flex-col gap-1 text-sm'>
          {t('Default reasoning effort')}
          <select className='border-input bg-background h-9 rounded-md border px-2' value={props.value.default_effort} onChange={(e) => props.onChange({ ...props.value, default_effort: e.target.value })}>
            {props.value.efforts.map((effort) => <option key={effort} value={effort}>{effort}</option>)}
          </select>
        </label>
      ) : null}
      <div className='flex items-center justify-between'>
        <label className='flex items-center gap-2 text-sm'>
          <Switch checked={props.value.vision} onCheckedChange={(vision) => props.onChange({ ...props.value, vision })} aria-label={t('Image input')} />
          {t('Image input')}
        </label>
        <Button variant='outline' size='sm' onClick={props.onRemove}>{t('Remove')}</Button>
      </div>
    </div>
  )
}

export function LightcodeGroupModels() {
  const { t } = useTranslation()
  const groups = useQuery({ queryKey: ['lightcode', 'groups'], queryFn: getGroups })
  const models = useQuery({ queryKey: ['lightcode', 'models'], queryFn: getLightcodeModels })
  const [group, setGroup] = useState('')
  const [rows, setRows] = useState<LightcodeGroupModel[]>([])
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setRows((models.data?.data ?? []).filter((m) => m.routing_group === group))
  }, [group, models.data])

  const save = async () => {
    setSaving(true)
    try {
      const res = await saveLightcodeModels(group, rows.map((r) => ({ ...r, display_name: r.display_name || r.model })))
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
            {t('Choose a channel group, then list the models Lightcode shows for it. Subscriptions select which of these models they support.')}
          </p>
          <label className='flex flex-col gap-1 text-sm'>
            {t('Channel group')}
            <select className='border-input bg-background h-9 rounded-md border px-2' value={group} onChange={(e) => setGroup(e.target.value)}>
              <option value=''>{t('Select a group')}</option>
              {groupOptions.map((g) => <option key={g} value={g}>{g}</option>)}
            </select>
          </label>
          {group ? (
            <>
              {rows.map((row, index) => (
                <ModelRow
                  key={index}
                  value={row}
                  onChange={(value) => setRows(rows.map((r, i) => (i === index ? value : r)))}
                  onRemove={() => setRows(rows.filter((_, i) => i !== index))}
                />
              ))}
              <div className='flex gap-3'>
                <Button variant='outline' onClick={() => setRows([...rows, emptyModel()])}>{t('Add model')}</Button>
                <Button onClick={save} disabled={saving || rows.some((r) => !r.model)}>{t('Save')}</Button>
              </div>
            </>
          ) : null}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
