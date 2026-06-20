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
import type { AxiosRequestConfig } from 'axios'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'
import { api, getSelf } from '@/lib/api'
import { getOAuthState } from '../api'
import {
  buildGitHubOAuthUrl,
  buildDiscordOAuthUrl,
  buildOIDCOAuthUrl,
  buildLinuxDOOAuthUrl,
} from '../lib/oauth'
import type { SystemStatus, CustomOAuthProviderInfo } from '../types'

type LogoutRequestConfig = AxiosRequestConfig & {
  skipErrorHandler?: boolean
}

type MofangJwtMessage = {
  type?: string
  jwt?: unknown
}

/**
 * Hook for managing OAuth login
 */
export function useOAuthLogin(status: SystemStatus | null) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(false)
  const [githubButtonText, setGithubButtonText] = useState('')
  const [githubButtonDisabled, setGithubButtonDisabled] = useState(false)
  const githubTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const { auth } = useAuthStore()

  useEffect(() => {
    setGithubButtonText(t('Continue with GitHub'))

    return () => {
      if (githubTimeoutRef.current) {
        clearTimeout(githubTimeoutRef.current)
      }
    }
  }, [t])

  const resetSession = async () => {
    try {
      auth.reset()
    } catch (_error) {
      // ignore store reset errors
    }
    try {
      await api.get('/api/user/logout', {
        skipErrorHandler: true,
      } as LogoutRequestConfig)
    } catch (_error) {
      // ignore logout errors
    }
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
      const state = await getOAuthState()
      if (!state) {
        toast.error(t('Failed to initialize OAuth'))
        if (githubTimeoutRef.current) {
          clearTimeout(githubTimeoutRef.current)
        }
        setIsLoading(false)
        setGithubButtonText(t('Continue with GitHub'))
        setGithubButtonDisabled(false)
        return
      }

      const url = buildGitHubOAuthUrl(status.github_client_id, state)
      window.open(url, '_self')
    } catch (_error) {
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
      const state = await getOAuthState()
      if (!state) {
        toast.error(t('Failed to initialize OAuth'))
        return
      }

      const url = buildDiscordOAuthUrl(status.discord_client_id, state)
      window.open(url, '_self')
    } catch (_error) {
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
      const state = await getOAuthState()
      if (!state) {
        toast.error(t('Failed to initialize OAuth'))
        return
      }

      const url = buildOIDCOAuthUrl(
        status.oidc_authorization_endpoint,
        status.oidc_client_id,
        state
      )
      window.open(url, '_self')
    } catch (_error) {
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
      const state = await getOAuthState()
      if (!state) {
        toast.error(t('Failed to initialize OAuth'))
        return
      }

      const url = buildLinuxDOOAuthUrl(status.linuxdo_client_id, state)
      window.open(url, '_self')
    } catch (_error) {
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
    } catch (_error) {
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

    setIsLoading(true)
    try {
      await resetSession()
    } catch (_error) {
      // resetSession already ignores expected failures
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
      setIsLoading(false)
    }

    const completeWithJWT = async (jwt: string) => {
      try {
        const res = await api.post('/api/oauth/mofang/session', { jwt })
        if (!res?.data?.success) {
          toast.error(res?.data?.message || t('Mofang login failed'))
          return
        }

        const loginUserId = res.data?.data?.id
        if (loginUserId != null) {
          window.localStorage.setItem('uid', String(loginUserId))
        }
        if (res.data?.data) {
          auth.setUser(res.data.data as AuthUser)
        }

        try {
          const self = await getSelf()
          if (self?.success && self.data) {
            auth.setUser(self.data)
            if (self.data.id != null) {
              window.localStorage.setItem('uid', String(self.data.id))
            }
          }
        } catch (_error) {
          // The session is already established; the next route load can refresh user details.
        }

        toast.success(t('Signed in successfully!'))
        popup.close()
        window.location.replace('/dashboard')
      } catch (_error) {
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

    window.addEventListener('message', handleMessage)
    timeoutId = setTimeout(() => {
      if (finished) return
      toast.error(t('Mofang login timed out'))
      cleanup()
    }, 120000)
    closeCheckId = setInterval(() => {
      if (finished) return
      if (popup.closed) {
        cleanup()
      }
    }, 1000)

    popup.location.href = loginUrl.toString()
  }

  const handleTelegramLogin = () => {
    toast.info(t('Telegram login requires widget integration; coming soon'))
  }

  const handleCustomOAuthLogin = async (provider: CustomOAuthProviderInfo) => {
    if (!provider.authorization_endpoint || !provider.client_id) return

    setIsLoading(true)
    try {
      await resetSession()
      const state = await getOAuthState()
      if (!state) {
        toast.error(t('Failed to initialize OAuth'))
        return
      }

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
    } catch (_error) {
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
    handleGitHubLogin,
    handleDiscordLogin,
    handleOIDCLogin,
    handleLinuxDOLogin,
    handleMofangLogin,
    handleTelegramLogin,
    handleCustomOAuthLogin,
  }
}
