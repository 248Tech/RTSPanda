export interface AuthConfig {
  enabled: boolean
  mode: string
}

export interface AuthSession {
  enabled: boolean
  authenticated: boolean
}

interface ErrorResponse {
  error?: string
}

async function parseJSON<T>(res: Response): Promise<T | null> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text) as T
  } catch {
    return null
  }
}

function errorMessageFrom(res: Response, payload: ErrorResponse | null, fallback: string): string {
  if (payload?.error && payload.error.trim() !== '') {
    return payload.error
  }
  return `${fallback} (${res.status})`
}

export async function fetchAuthConfig(): Promise<AuthConfig> {
  const res = await fetch('/api/v1/auth/config', { credentials: 'same-origin' })
  const payload = await parseJSON<AuthConfig & ErrorResponse>(res)
  if (!res.ok || payload === null) {
    throw new Error(errorMessageFrom(res, payload, 'failed to load auth config'))
  }
  return {
    enabled: payload.enabled,
    mode: payload.mode ?? 'unknown',
  }
}

export async function fetchAuthSession(): Promise<AuthSession> {
  const res = await fetch('/api/v1/auth/session', { credentials: 'same-origin' })
  const payload = await parseJSON<AuthSession & ErrorResponse>(res)

  if (res.status === 401) {
    return {
      enabled: true,
      authenticated: false,
    }
  }

  if (!res.ok || payload === null) {
    throw new Error(errorMessageFrom(res, payload, 'failed to load auth session'))
  }

  return {
    enabled: payload.enabled,
    authenticated: payload.authenticated,
  }
}

export async function loginWithToken(token: string): Promise<void> {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify({ token }),
  })
  const payload = await parseJSON<ErrorResponse>(res)
  if (!res.ok) {
    throw new Error(errorMessageFrom(res, payload, 'login failed'))
  }
}

export async function logoutSession(): Promise<void> {
  await fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'same-origin',
  })
}
