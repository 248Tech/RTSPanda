import { createContext } from 'react'

export interface AuthContextValue {
  loading: boolean
  enabled: boolean
  authenticated: boolean
  mode: string
  error: string | null
  isLoggingIn: boolean
  login: (token: string) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)
