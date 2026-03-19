import { useMemo, useState } from 'react'

interface LoginViewProps {
  mode: string
  error: string | null
  isSubmitting: boolean
  onSubmit: (token: string) => Promise<void>
}

export function LoginView({ mode, error, isSubmitting, onSubmit }: LoginViewProps) {
  const [token, setToken] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  const message = useMemo(() => error ?? localError, [error, localError])

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const normalized = token.trim()
    if (normalized === '') {
      setLocalError('Token is required.')
      return
    }

    setLocalError(null)
    try {
      await onSubmit(normalized)
    } catch {
      // Error message is surfaced by auth context.
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-base px-4 py-8 font-sans text-text-primary">
      <div className="w-full max-w-md rounded-xl border border-border-muted bg-surface p-6 shadow-modal">
        <h1 className="text-2xl font-semibold">Sign in</h1>
        <p className="mt-2 text-sm text-text-muted">
          Enter your RTSPanda auth token to access protected API routes.
        </p>
        <p className="mt-1 text-xs uppercase tracking-wide text-text-subtle">Mode: {mode}</p>

        <form className="mt-6 space-y-3" onSubmit={submit}>
          <label className="block text-sm font-medium text-text-primary" htmlFor="auth-token">
            Auth token
          </label>
          <input
            id="auth-token"
            type="password"
            autoComplete="current-password"
            value={token}
            onChange={(event) => setToken(event.target.value)}
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-sm text-text-primary outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/40"
            placeholder="Paste AUTH_TOKEN"
            disabled={isSubmitting}
          />

          {message && (
            <p className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {message}
            </p>
          )}

          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex w-full items-center justify-center rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-70"
          >
            {isSubmitting ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}
