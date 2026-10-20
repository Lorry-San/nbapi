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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import type { SubscriptionPlan, PlanPayload } from '../types'

export function getPlanFormSchema(t: TFunction) {
  return z.object({
    title: z.string().min(1, t('Please enter plan title')),
    subtitle: z.string().optional(),
    price_amount: z.coerce.number().min(0, t('Please enter amount')),
    duration_unit: z.enum(['year', 'month', 'day', 'hour', 'custom']),
    duration_value: z.coerce.number().min(1),
    custom_seconds: z.coerce.number().min(0).optional(),
    quota_reset_period: z.enum([
      'never',
      'daily',
      'weekly',
      'monthly',
      'custom',
    ]),
    quota_reset_custom_seconds: z.coerce.number().min(0).optional(),
    enabled: z.boolean(),
    sort_order: z.coerce.number(),
    recommended: z.boolean(),
    allow_balance_pay: z.boolean(),
    allow_wallet_overflow: z.boolean(),
    max_purchase_per_user: z.coerce.number().min(0),
    total_amount: z.coerce.number().min(0),
    upgrade_group: z.string().optional(),
    downgrade_group: z.string().optional(),
    stripe_price_id: z.string().optional(),
    creem_product_id: z.string().optional(),
    waffo_pancake_product_id: z.string().optional(),
    pool: z.enum(['api', 'lightcode']),
    routing_group: z.string().optional(),
    allowed_models: z.array(z.string()),
    window_short_hours: z.coerce.number().min(0),
    window_short_amount: z.coerce.number().min(0),
    window_long_days: z.coerce.number().min(0),
    window_long_amount: z.coerce.number().min(0),
  }).superRefine((value, ctx) => {
    if (value.pool !== 'lightcode') return
    if (!value.routing_group) {
      ctx.addIssue({ code: 'custom', path: ['routing_group'], message: t('Select a Lightcode model group') })
    }
    if (value.allowed_models.length === 0) {
      ctx.addIssue({ code: 'custom', path: ['allowed_models'], message: t('Select at least one supported model') })
    }
  })
}

export type PlanFormValues = z.infer<ReturnType<typeof getPlanFormSchema>>

export const PLAN_FORM_DEFAULTS: PlanFormValues = {
  title: '',
  subtitle: '',
  price_amount: 0,
  duration_unit: 'month',
  duration_value: 1,
  custom_seconds: 0,
  quota_reset_period: 'never',
  quota_reset_custom_seconds: 0,
  enabled: true,
  sort_order: 0,
  recommended: false,
  allow_balance_pay: true,
  allow_wallet_overflow: true,
  max_purchase_per_user: 0,
  total_amount: 0,
  upgrade_group: '',
  downgrade_group: '',
  stripe_price_id: '',
  creem_product_id: '',
  waffo_pancake_product_id: '',
  pool: 'api',
  routing_group: '',
  allowed_models: [],
  window_short_hours: 5,
  window_short_amount: 0,
  window_long_days: 7,
  window_long_amount: 0,
}

export function planToFormValues(plan: SubscriptionPlan): PlanFormValues {
  return {
    title: plan.title || '',
    subtitle: plan.subtitle || '',
    price_amount: Number(plan.price_amount || 0),
    duration_unit: plan.duration_unit || 'month',
    duration_value: Number(plan.duration_value || 1),
    custom_seconds: Number(plan.custom_seconds || 0),
    quota_reset_period: plan.quota_reset_period || 'never',
    quota_reset_custom_seconds: Number(plan.quota_reset_custom_seconds || 0),
    enabled: plan.enabled !== false,
    sort_order: Number(plan.sort_order || 0),
    recommended: plan.recommended === true,
    allow_balance_pay: plan.allow_balance_pay !== false,
    allow_wallet_overflow: plan.allow_wallet_overflow !== false,
    max_purchase_per_user: Number(plan.max_purchase_per_user || 0),
    total_amount: quotaUnitsToDollars(Number(plan.total_amount || 0)),
    upgrade_group: plan.upgrade_group || '',
    downgrade_group: plan.downgrade_group || '',
    stripe_price_id: plan.stripe_price_id || '',
    creem_product_id: plan.creem_product_id || '',
    waffo_pancake_product_id: plan.waffo_pancake_product_id || '',
    pool: plan.pool === 'lightcode' ? 'lightcode' : 'api',
    routing_group: plan.routing_group || '',
    allowed_models: plan.allowed_models ?? [],
    window_short_hours: plan.window_short_seconds ? plan.window_short_seconds / 3600 : 5,
    window_short_amount: quotaUnitsToDollars(Number(plan.window_short_amount || 0)),
    window_long_days: plan.window_long_seconds ? plan.window_long_seconds / 86400 : 7,
    window_long_amount: quotaUnitsToDollars(Number(plan.window_long_amount || 0)),
  }
}

export function formValuesToPlanPayload(values: PlanFormValues): PlanPayload {
  return {
    plan: {
      ...values,
      price_amount: Number(values.price_amount || 0),
      currency: 'USD',
      duration_value: Number(values.duration_value || 0),
      custom_seconds: Number(values.custom_seconds || 0),
      quota_reset_period: values.quota_reset_period || 'never',
      quota_reset_custom_seconds:
        values.quota_reset_period === 'custom'
          ? Number(values.quota_reset_custom_seconds || 0)
          : 0,
      sort_order: Number(values.sort_order || 0),
      max_purchase_per_user: Number(values.max_purchase_per_user || 0),
      total_amount: parseQuotaFromDollars(Number(values.total_amount || 0)),
      // A Lightcode entitlement never changes the API user group or spills into the wallet.
      upgrade_group: values.pool === 'lightcode' ? '' : values.upgrade_group || '',
      downgrade_group: values.pool === 'lightcode' ? '' : values.downgrade_group || '',
      allow_wallet_overflow: values.pool === 'lightcode' ? false : values.allow_wallet_overflow,
      routing_group: values.pool === 'lightcode' ? values.routing_group || '' : '',
      allowed_models: values.pool === 'lightcode' ? values.allowed_models : [],
      ...lightcodeWindowPayload(values),
    },
  }
}

/** A window with no amount is disabled; the server rejects a half-configured window. */
function lightcodeWindowPayload(values: PlanFormValues) {
  const lightcode = values.pool === 'lightcode'
  const short = lightcode ? parseQuotaFromDollars(Number(values.window_short_amount || 0)) : 0
  const long = lightcode ? parseQuotaFromDollars(Number(values.window_long_amount || 0)) : 0
  return {
    window_short_amount: short,
    window_short_seconds: short > 0 ? Math.round(Number(values.window_short_hours) * 3600) : 0,
    window_long_amount: long,
    window_long_seconds: long > 0 ? Math.round(Number(values.window_long_days) * 86400) : 0,
  }
}
