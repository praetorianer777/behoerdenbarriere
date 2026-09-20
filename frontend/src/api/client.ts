import type { AgencyDetail, AgencyList, Scan, Stats, Usage } from './types'

const base = import.meta.env.VITE_API_URL ?? '/api/v1'

async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${base}${path}`, {
    signal,
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    // Der Text der Antwort ist für Menschen gedacht, die Statuszeile für uns.
    throw new Error(`${response.status} ${response.statusText}`)
  }
  return (await response.json()) as T
}

export interface AgencyQuery {
  q?: string
  level?: string
  state?: string
  grade?: string
  sort?: string
  page?: number
  per_page?: number
}

export function agencyQueryString(query: AgencyQuery): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== '' && value !== null) {
      params.set(key, String(value))
    }
  }
  const search = params.toString()
  return search ? `?${search}` : ''
}

export const api = {
  agencies: (query: AgencyQuery, signal?: AbortSignal) =>
    get<AgencyList>(`/agencies${agencyQueryString(query)}`, signal),
  agency: (slug: string, signal?: AbortSignal) => get<AgencyDetail>(`/agencies/${slug}`, signal),
  latestScan: (slug: string, signal?: AbortSignal) =>
    get<Scan>(`/agencies/${slug}/scans/latest`, signal),
  stats: (signal?: AbortSignal) => get<Stats>('/stats', signal),
  usage: (days: number, signal?: AbortSignal) => get<Usage>(`/usage?days=${days}`, signal),

  /**
   * Meldet der eigenen API, welche Seite aufgerufen wurde. Gezählt wird dort — ohne
   * Cookie, ohne Kennung, ohne fremden Dienst. Schlägt der Aufruf fehl, fehlt eine
   * Zahl in der Statistik und sonst nichts, deshalb wird der Fehler verschluckt.
   */
  view: (path: string) =>
    fetch(`${base}/view`, {
      method: 'POST',
      headers: { 'X-Page': path },
      keepalive: true,
    }).catch(() => undefined),
}
