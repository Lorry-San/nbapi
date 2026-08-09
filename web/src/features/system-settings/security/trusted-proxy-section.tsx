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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const trustedProxySchema = z.object({
  TrustedProxiesEnabled: z.boolean(),
})

type TrustedProxyFormValues = z.infer<typeof trustedProxySchema>

type TrustedProxySectionProps = {
  defaultValues: TrustedProxyFormValues
}

export function TrustedProxySection({
  defaultValues,
}: TrustedProxySectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm({
    resolver: zodResolver(trustedProxySchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (data: TrustedProxyFormValues) => {
    if (data.TrustedProxiesEnabled === defaultValues.TrustedProxiesEnabled) {
      return
    }
    await updateOption.mutateAsync({
      key: 'TrustedProxiesEnabled',
      value: data.TrustedProxiesEnabled,
    })
  }

  return (
    <SettingsSection title={t('Trusted Proxies')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save trusted proxy settings'
          />
          <FormField
            control={form.control}
            name='TrustedProxiesEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Validate Trusted Proxy Sources')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When enabled, only client IP headers from proxies allowed by TRUSTED_PROXIES are accepted'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <FormDescription>
            {t(
              'When disabled, proxy source validation is skipped and CF-Connecting-IP, X-Forwarded-For, and X-Real-IP are trusted directly'
            )}
          </FormDescription>
          <FormDescription>
            {t(
              'Only disable validation when direct access to NBAPI is blocked or all callers are controlled, because clients can otherwise spoof these headers'
            )}
          </FormDescription>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
