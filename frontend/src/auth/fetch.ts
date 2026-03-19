type UnauthorizedHandler = () => void

let installed = false
let unauthorizedHandler: UnauthorizedHandler | null = null

const ignoredUnauthorizedPaths = new Set([
  '/api/v1/auth/config',
  '/api/v1/auth/session',
  '/api/v1/auth/login',
  '/api/v1/auth/logout',
  '/api/v1/health',
  '/api/v1/health/ready',
])

function resolvePath(input: RequestInfo | URL): string | null {
  try {
    if (typeof input === 'string') {
      return new URL(input, window.location.origin).pathname
    }
    if (input instanceof URL) {
      return input.pathname
    }
    return new URL(input.url, window.location.origin).pathname
  } catch {
    return null
  }
}

export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler
}

export function installAuthFetch(): void {
  if (installed) {
    return
  }

  const nativeFetch = window.fetch.bind(window)
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = resolvePath(input)
    const requestInit: RequestInit = {
      ...init,
      credentials: init?.credentials ?? 'same-origin',
    }

    const res = await nativeFetch(input, requestInit)
    if (
      path !== null &&
      path.startsWith('/api/') &&
      !ignoredUnauthorizedPaths.has(path) &&
      res.status === 401
    ) {
      unauthorizedHandler?.()
    }
    return res
  }

  installed = true
}
