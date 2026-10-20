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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { SubscriptionPlansCard } from '@/features/wallet/components/subscription-plans-card'
import { useTopupInfo } from '@/features/wallet/hooks'
import { getSelf } from '@/lib/api'

export function LightcodeSubscription() {
  const { t } = useTranslation()
  const { topupInfo } = useTopupInfo()
  const self = useQuery({ queryKey: ['self'], queryFn: getSelf })
  const quota = self.data?.success
    ? (self.data.data as { quota?: number } | undefined)?.quota
    : undefined

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Subscription')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto w-full max-w-6xl'>
          <SubscriptionPlansCard
            topupInfo={topupInfo}
            userQuota={quota}
            onPurchaseSuccess={() => {
              self.refetch()
            }}
            pool='lightcode'
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
