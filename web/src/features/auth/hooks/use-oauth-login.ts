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
import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { clearAuthentication, isAuthBundle } from '@/lib/api'

import { createOAuthFlow, logout, mofangLogin, telegramLogin } from '../api'
import {
  buildGitHubOAuthUrl,
  buildDiscordOAuthUrl,
  buildOIDCOAuthUrl,
  buildLinuxDOOAuthUrl,
} from '../lib/oauth'
import { MOFANG_CALLBACK_STORAGE_KEY } from '../lib/mofang-callback'
import { pickTelegramAuthorization } from '../lib/telegram-login'
import type { SystemStatus, CustomOAuthProviderInfo } from '../types'
import { useAuthRedirect } from './use-auth-redirect'

type MofangJwtMessage = {
  type?: string
  jwt?: unknown
}

function parseStoredMofangJwt(value: string | null, maxAgeMs: number): string {
  if (!value) return ''
  try {
    const payload = JSON.parse(value) as MofangJwtMessage & {
      timestamp?: unknown
    }
    if (payload?.type !== 'mofang-jwt') return ''
    if (typeof payload.jwt !== 'string' || !payload.jwt.trim()) return ''
    if (
      typeof payload.timestamp === 'number' &&
      Date.now() - payload.timestamp > maxAgeMs
    ) {
      return ''
    }
    return payload.jwt.trim()
  } catch {
    return ''
  }
}

/**
 * Hook for managing OAuth login
 */
export function useOAuthLogin(
  status: SystemStatus | null,
  redirectTo?: string
) {
  const { t } = useTranslation()
  const { handleLoginSuccess } = useAuthRedirect()
  const [isLoading, setIsLoading] = useState(false)
  const [isTelegramDialogOpen, setIsTelegramDialogOpen] = useState(false)
  const [isTelegramPending, setIsTelegramPending] = useState(false)
  const [githubButtonText, setGithubButtonText] = useState('')
  const [githubButtonDisabled, setGithubButtonDisabled] = useState(false)
  const githubTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    setGithubButtonText(t('Continue with GitHub'))

    return () => {
      if (githubTimeoutRef.current) {
        clearTimeout(githubTimeoutRef.current)
      }
    }
  }, [t])

  const resetSession = async () => {
    const response = await logout()
    if (!response.success) {
      throw new Error(response.message || t('Failed to sign out session'))
    }
    clearAuthentication()
  }

  const handleGitHubLogin = async () => {
    if (!status?.github_client_id) return
    if (githubButtonDisabled) return

    setIsLoading(true)
    setGithubButtonDisabled(true)
    setGithubButtonText(t('Redirecting to GitHub...'))

    if (githubTimeoutRef.current) {
      clearTimeout(githubTimeoutRef.current)
    }

    githubTimeoutRef.current = setTimeout(() => {
      setIsLoading(false)
      setGithubButtonText(
        t('Request timed out, please refresh and restart GitHub login')
      )
      setGithubButtonDisabled(true)
    }, 20000)

    try {
      await resetSession()
      const state = await createOAuthFlow('github', 'login')

      const url = buildGitHubOAuthUrl(status.github_client_id, state)
      window.open(url, '_self')
    } catch {
      toast.error(t('Failed to start GitHub login'))
      if (githubTimeoutRef.current) {
        clearTimeout(githubTimeoutRef.current)
      }
      setIsLoading(false)
      setGithubButtonText(t('Continue with GitHub'))
      setGithubButtonDisabled(false)
    }
  }

  const handleDiscordLogin = async () => {
    if (!status?.discord_client_id) return

    setIsLoading(true)
    try {
      await resetSession()
      const state = await createOAuthFlow('discord', 'login')

      const url = buildDiscordOAuthUrl(status.discord_client_id, state)
      window.open(url, '_self')
    } catch {
      toast.error(t('Failed to start Discord login'))
    } finally {
      setIsLoading(false)
    }
  }

  const handleOIDCLogin = async () => {
    if (!status?.oidc_authorization_endpoint || !status?.oidc_client_id) return

    setIsLoading(true)
    try {
      await resetSession()
      const state = await createOAuthFlow('oidc', 'login')

      const url = buildOIDCOAuthUrl(
        status.oidc_authorization_endpoint,
        status.oidc_client_id,
        state
      )
      window.open(url, '_self')
    } catch {
      toast.error(t('Failed to start OIDC login'))
    } finally {
      setIsLoading(false)
    }
  }

  const handleLinuxDOLogin = async () => {
    if (!status?.linuxdo_client_id) return

    setIsLoading(true)
    try {
      await resetSession()
      const state = await createOAuthFlow('linuxdo', 'login')

      const url = buildLinuxDOOAuthUrl(status.linuxdo_client_id, state)
      window.open(url, '_self')
    } catch {
      toast.error(t('Failed to start LinuxDO login'))
    } finally {
      setIsLoading(false)
    }
  }

  const handleMofangLogin = async () => {
    if (!status?.mofang_login_url) return

    let loginUrl: URL
    try {
      loginUrl = new URL(status.mofang_login_url)
    } catch {
      toast.error(t('Mofang login failed'))
      return
    }
    loginUrl.searchParams.set('redirect_url', window.location.origin)
    loginUrl.searchParams.set('origin', window.location.origin)

    const popup = window.open(
      'about:blank',
      'mofang-oauth',
      'popup=yes,width=480,height=680,menubar=no,toolbar=no,location=no,status=no'
    )
    if (!popup) {
      toast.error(t('Failed to open Mofang login window'))
      return
    }

    const timeoutMs = 120000
    setIsLoading(true)
    try {
      window.localStorage.removeItem(MOFANG_CALLBACK_STORAGE_KEY)
    } catch {
      // Storage may be unavailable in hardened browser contexts.
    }

    try {
      await resetSession()
    } catch {
      popup.close()
      setIsLoading(false)
      toast.error(t('Mofang login failed'))
      return
    }

    const allowedOrigins = new Set([loginUrl.origin, window.location.origin])
    let finished = false
    let timeoutId: ReturnType<typeof setTimeout> | undefined
    let closeCheckId: ReturnType<typeof setInterval> | undefined

    const cleanup = () => {
      finished = true
      if (timeoutId) clearTimeout(timeoutId)
      if (closeCheckId) clearInterval(closeCheckId)
      window.removeEventListener('message', handleMessage)
      window.removeEventListener('storage', handleStorage)
      setIsLoading(false)
    }

    const completeWithJWT = async (jwt: string) => {
      try {
        const response = await mofangLogin(jwt)
        if (!response.success || !isAuthBundle(response.data)) {
          toast.error(response.message || t('Mofang login failed'))
          return
        }
        popup.close()
        await handleLoginSuccess(response.data, redirectTo)
        toast.success(t('Signed in successfully!'))
      } catch {
        toast.error(t('Mofang login failed'))
      } finally {
        cleanup()
      }
    }

    function handleMessage(event: MessageEvent<MofangJwtMessage>) {
      if (finished || !allowedOrigins.has(event.origin)) return
      if (!event.data || event.data.type !== 'mofang-jwt') return
      const jwt = event.data.jwt
      if (typeof jwt !== 'string' || !jwt.trim()) {
        toast.error(t('Mofang login failed'))
        cleanup()
        return
      }
      void completeWithJWT(jwt.trim())
    }

    function consumeStoredToken() {
      if (finished) return
      const jwt = parseStoredMofangJwt(
        window.localStorage.getItem(MOFANG_CALLBACK_STORAGE_KEY),
        timeoutMs
      )
      if (!jwt) return
      try {
        window.localStorage.removeItem(MOFANG_CALLBACK_STORAGE_KEY)
      } catch {
        // Storage may be unavailable in hardened browser contexts.
      }
      void completeWithJWT(jwt)
    }

    function handleStorage(event: StorageEvent) {
      if (event.key !== MOFANG_CALLBACK_STORAGE_KEY) return
      const jwt = parseStoredMofangJwt(event.newValue, timeoutMs)
      if (!jwt) return
      try {
        window.localStorage.removeItem(MOFANG_CALLBACK_STORAGE_KEY)
      } catch {
        // Storage may be unavailable in hardened browser contexts.
      }
      void completeWithJWT(jwt)
    }

    window.addEventListener('message', handleMessage)
    window.addEventListener('storage', handleStorage)
    timeoutId = setTimeout(() => {
      if (finished) return
      toast.error(t('Mofang login timed out'))
      cleanup()
    }, timeoutMs)
    closeCheckId = setInterval(() => {
      if (finished) return
      consumeStoredToken()
      if (popup.closed) cleanup()
    }, 1000)

    popup.location.href = loginUrl.toString()
  }

  const handleTelegramLogin = async () => {
    if (!status?.telegram_bot_name?.trim()) {
      toast.error(t('Login failed'))
      return
    }

    setIsLoading(true)
    try {
      await resetSession()
      setIsTelegramDialogOpen(true)
    } catch {
      toast.error(
        t('Failed to start {{provider}} login', { provider: 'Telegram' })
      )
    } finally {
      setIsLoading(false)
    }
  }

  const handleTelegramAuthorization = async (value: unknown) => {
    const authorization = pickTelegramAuthorization(value)
    if (!authorization) {
      toast.error(t('Login failed'))
      return
    }

    setIsTelegramPending(true)
    try {
      const response = await telegramLogin(authorization)
      if (!response.success || !isAuthBundle(response.data)) {
        toast.error(t('Login failed'))
        return
      }

      setIsTelegramDialogOpen(false)
      await handleLoginSuccess(response.data, redirectTo)
      toast.success(t('Welcome back!'))
    } catch {
      toast.error(t('Login failed'))
    } finally {
      setIsTelegramPending(false)
    }
  }

  const handleCustomOAuthLogin = async (provider: CustomOAuthProviderInfo) => {
    if (!provider.authorization_endpoint || !provider.client_id) return

    setIsLoading(true)
    try {
      await resetSession()
      const state = await createOAuthFlow(provider.slug, 'login')

      const redirectUri = `${window.location.origin}/oauth/${provider.slug}`
      const url = new URL(provider.authorization_endpoint)
      url.searchParams.set('client_id', provider.client_id)
      url.searchParams.set('redirect_uri', redirectUri)
      url.searchParams.set('response_type', 'code')
      url.searchParams.set('state', state)
      if (provider.scopes) {
        url.searchParams.set('scope', provider.scopes)
      }

      window.open(url.toString(), '_self')
    } catch {
      toast.error(
        t('Failed to start {{provider}} login', { provider: provider.name })
      )
    } finally {
      setIsLoading(false)
    }
  }

  return {
    isLoading,
    githubButtonText,
    githubButtonDisabled,
    isTelegramDialogOpen,
    isTelegramPending,
    handleGitHubLogin,
    handleDiscordLogin,
    handleOIDCLogin,
    handleLinuxDOLogin,
    handleMofangLogin,
    handleTelegramLogin,
    handleTelegramAuthorization,
    setIsTelegramDialogOpen,
    handleCustomOAuthLogin,
  }
}
