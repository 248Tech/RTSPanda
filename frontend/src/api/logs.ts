const BASE = '/api/v1'

export interface LogsResponse {
  lines: string[]
}

export async function getLogs(): Promise<LogsResponse> {
  const res = await fetch(`${BASE}/logs`)
  if (!res.ok) throw new Error(`getLogs: ${res.status}`)
  return res.json()
}
