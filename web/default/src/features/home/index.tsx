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
import type { CSSProperties, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { Markdown } from '@/components/ui/markdown'
import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { CTA, Features, Hero, HowItWorks, Stats } from './components'
import { useHomePageContent } from './hooks'

function isLikelyHtml(value: string) {
  return /<\/?[a-z][\s\S]*>/i.test(value)
}

function backgroundStyle(background: string): CSSProperties | undefined {
  const trimmed = background.trim()
  if (!trimmed) return undefined
  const safeUrl = trimmed.replace(/["\\\n\r]/g, '')
  return {
    backgroundImage: `url("${safeUrl}")`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundAttachment: 'fixed',
  }
}

function HomeBackground({
  background,
  children,
}: {
  background: string
  children: ReactNode
}) {
  if (!background.trim()) return <>{children}</>

  return (
    <div className='relative isolate min-h-screen overflow-hidden'>
      <div
        aria-hidden
        className='absolute inset-0 -z-20'
        style={backgroundStyle(background)}
      />
      <div
        aria-hidden
        className='bg-background/78 absolute inset-0 -z-10 backdrop-blur-[1px] dark:bg-background/70'
      />
      {children}
    </div>
  )
}

export function Home() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, background, isLoaded, isUrl } = useHomePageContent()
  const isHtml = content && !isUrl && isLikelyHtml(content)

  if (!isLoaded) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-screen items-center justify-center'>
          <div className='text-muted-foreground'>{t('Loading...')}</div>
        </main>
      </PublicLayout>
    )
  }

  if (content) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='relative min-h-screen overflow-x-hidden'>
          {isUrl ? (
            <iframe
              src={content}
              className='h-screen w-full border-none'
              title={t('Custom Home Page')}
            />
          ) : isHtml ? (
            <HomeBackground background={background}>
              <div
                className='custom-home-html min-h-screen'
                dangerouslySetInnerHTML={{ __html: content }}
              />
            </HomeBackground>
          ) : (
            <HomeBackground background={background}>
              <div className='container mx-auto py-8'>
                <Markdown className='custom-home-content'>{content}</Markdown>
              </div>
            </HomeBackground>
          )}
        </main>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <HomeBackground background={background}>
        <Hero isAuthenticated={isAuthenticated} />
        <Stats />
        <Features />
        <HowItWorks />
        <CTA isAuthenticated={isAuthenticated} />
        <Footer />
      </HomeBackground>
    </PublicLayout>
  )
}
