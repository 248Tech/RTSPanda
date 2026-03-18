const BASE = '/api/v1'

export interface SystemStats {
  uptime_seconds: number
  goroutines: number
  heap_alloc_bytes: number
  heap_sys_bytes: number
  rss_bytes: number
  network_bytes_in: number
  network_bytes_out: number
  http_requests_total: number
  goos: string
  goarch: string
  num_cpu: number
}

export async function getSystemStats(): Promise<SystemStats> {
  const res = await fetch(`${BASE}/system/stats`)
  if (!res.ok) throw new Error(`getSystemStats: ${res.status}`)
  return res.json()
}
