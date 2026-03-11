const BASE = '/api/v1'

export type AlertType = 'motion' | 'connectivity' | 'object_detection'

export interface AlertRule {
  id: string
  camera_id: string
  name: string
  type: AlertType
  enabled: boolean
  config: string // JSON string
  created_at: string
  updated_at: string
}

export interface AlertEvent {
  id: string
  rule_id: string
  camera_id: string
  triggered_at: string
  snapshot_path?: string
  metadata: string // JSON string
}

export interface CreateAlertRuleInput {
  name: string
  type: AlertType
  enabled?: boolean
  config?: string
}

export interface UpdateAlertRuleInput {
  name?: string
  type?: AlertType
  enabled?: boolean
  config?: string
}

export async function listAlertRules(cameraId: string): Promise<AlertRule[]> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/alerts`)
  if (!res.ok) throw new Error(`listAlertRules: ${res.status}`)
  return res.json()
}

export async function createAlertRule(cameraId: string, data: CreateAlertRuleInput): Promise<AlertRule> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/alerts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`createAlertRule: ${res.status}`)
  return res.json()
}

export async function updateAlertRule(id: string, data: UpdateAlertRuleInput): Promise<AlertRule> {
  const res = await fetch(`${BASE}/alerts/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`updateAlertRule: ${res.status}`)
  return res.json()
}

export async function deleteAlertRule(id: string): Promise<void> {
  const res = await fetch(`${BASE}/alerts/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteAlertRule: ${res.status}`)
}

export async function listAlertEvents(ruleId: string): Promise<AlertEvent[]> {
  const res = await fetch(`${BASE}/alerts/${ruleId}/events`)
  if (!res.ok) throw new Error(`listAlertEvents: ${res.status}`)
  return res.json()
}

export async function listCameraAlertEvents(cameraId: string): Promise<AlertEvent[]> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/alert-events`)
  if (!res.ok) throw new Error(`listCameraAlertEvents: ${res.status}`)
  return res.json()
}
