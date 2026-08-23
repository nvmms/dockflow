const API_BASE = '/api/v1'

export interface LogStream { close(): void }

export function openLogStream(
  path: string,
  handlers: { onLog: (chunk: string) => void; onDone?: (status: string) => void; onError?: () => void },
): LogStream {
  const source = new EventSource(`${API_BASE}${path}`)
  source.addEventListener('log', (event) => {
    const payload = JSON.parse((event as MessageEvent).data) as { chunk?: string }
    if (payload.chunk) handlers.onLog(payload.chunk)
  })
  source.addEventListener('done', (event) => {
    const payload = JSON.parse((event as MessageEvent).data) as { status?: string }
    handlers.onDone?.(payload.status || 'closed')
    source.close()
  })
  source.addEventListener('error', () => {
    handlers.onError?.()
    source.close()
  })
  return { close: () => source.close() }
}
