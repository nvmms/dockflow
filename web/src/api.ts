const API_BASE = '/api/v1'

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (options.body && !(options.body instanceof Blob) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(body.error || `请求失败 (${response.status})`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  get<T>(path: string) { return request<T>(path) },
  post<T>(path: string, data?: unknown) { return request<T>(path, { method: 'POST', body: data === undefined ? undefined : JSON.stringify(data) }) },
  put<T>(path: string, data: unknown) { return request<T>(path, { method: 'PUT', body: JSON.stringify(data) }) },
  delete(path: string) { return request<void>(path, { method: 'DELETE' }) },
}

export interface Namespace {
  name: string
  network: string
  network_id: string
  subnet: string
  gateway: string
  created_at: string
}

export interface AppRecord {
  name: string
  cpu: number
  memory: number
  repo: string
  trigger: { type: 'branch' | 'tag'; rule: string }
  env: Array<{ key: string; value: string }>
  url: Array<{ host: string; port: string }>
  deploy?: Array<{ containerId: string; version: string; url: string }>
}

export interface DatabaseRecord {
  name: string
  cpu: number
  memory: number
  username: string
  dbname: string
  dbtype: string
  container_id: string
  ip: string[]
  remote: boolean
}

export interface RedisRecord {
  name: string
  version: string
  cpu: number
  memory: number
  appendonly: boolean
  maxmemory_policy: string
  container_id: string
  ip: string[]
}
