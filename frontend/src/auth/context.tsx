import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { fetchAuthConfig, fetchAuthSession, loginWithToken, logoutSession } from './api'
import { setUnauthorizedHandler } from './fetch'
import { AuthContext, type AuthContextValue } from './AuthContext'

function toErrorMessage(err: unknown): string {
  if (err instanceof Error && err.message.trim() !== '') {
    return err.message
  }
  return 'Unexpected auth error'
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true)
  const [enabled, setEnabled] = useState(true)
  const [authenticated, setAuthenticated] = useState(false)
  const [mode, setMode] = useState('unknown')
  const [error, setError] = useState<string | null>(null)
  const [isLoggingIn, setIsLoggingIn] = useState(false)

  useEffect(() => {
    let active = true

    const bootstrap = async () => {
      setLoading(true)
      setError(null)
      try {
        const config = await fetchAuthConfig()
        if (!active) return

        setEnabled(config.enabled)
        setMode(config.mode)

        if (!config.enabled) {
          setAuthenticated(true)
          setLoading(false)
          return
        }

        const session = await fetchAuthSession()
        if (!active) return
        setAuthenticated(session.authenticated)
      } catch (err) {
        if (!active) return
        setEnabled(true)
        setAuthenticated(false)
        setError(toErrorMessage(err))
      } finally {
        if (active) {
          setLoading(false)
        }
      }
    }

    void bootstrap()
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setAuthenticated(false)
      setError('Session expired. Sign in again.')
    })
    return () => setUnauthorizedHandler(null)
  }, [])

  const login = useCallback(async (token: string) => {
    setIsLoggingIn(true)
    setError(null)
    try {
      await loginWithToken(token)
      setAuthenticated(true)
    } catch (err) {
      setAuthenticated(false)
      const message = toErrorMessage(err)
      setError(message)
      throw new Error(message)
    } finally {
      setIsLoggingIn(false)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await logoutSession()
    } finally {
      if (enabled) {
        setAuthenticated(false)
      } else {
        setAuthenticated(true)
      }
      setError(null)
    }
  }, [enabled])

  const value = useMemo<AuthContextValue>(
    () => ({
      loading,
      enabled,
      authenticated,
      mode,
      error,
      isLoggingIn,
      login,
      logout,
    }),
    [authenticated, enabled, error, isLoggingIn, loading, login, logout, mode],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
