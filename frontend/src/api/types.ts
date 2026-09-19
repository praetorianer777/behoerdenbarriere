export type Level = 'bund' | 'land' | 'kreis' | 'kommune'

export interface Agency {
  slug: string
  name: string
  url: string
  level: Level
  state?: string
  category?: string
  score: number | null
  grade?: string
  delta?: number
  pages: number
  scanned_at?: string
}

export interface Subscores {
  perceivable: number | null
  operable: number | null
  understandable: number | null
  robust: number | null
}

export type Direction = 'improved' | 'declined' | 'unchanged' | 'unknown'

export interface TrendPoint {
  at: string
  score: number
  grade: string
}

export interface TrendSummary {
  latest?: TrendPoint
  previous?: TrendPoint
  direction: Direction
  delta_last?: number
  delta_30d?: number
  delta_90d?: number
  delta_365d?: number
  points_per_month?: number
  months_to_grade_a?: number
  scans: number
}

export interface AgencyDetail extends Agency {
  subscores: Subscores
  trend: TrendSummary
  history: TrendPoint[]
  latest_scan_id?: number
}

export interface AgencyList {
  items: Agency[]
  total: number
  page: number
  per_page: number
}

export type Impact = 'critical' | 'serious' | 'moderate' | 'minor'

export interface Rule {
  rule_id: string
  impact: Impact
  principle: string
  help?: string
  help_url?: string
  pages: number
  nodes: number
  sample_html?: string
  sample_target?: string
}

export interface Page {
  url: string
  title?: string
  depth: number
  is_entry: boolean
  priority: boolean
  http_status?: number
  dom_nodes: number
  score: number | null
  load_ms?: number
  error?: string
  violations: number
}

export interface RuleChange {
  fixed: Rule[]
  introduced: Rule[]
  remaining: Rule[]
}

export interface Scan {
  id: number
  agency_slug: string
  agency_name: string
  status: string
  started_at: string
  finished_at?: string
  error?: string
  score: number | null
  grade?: string
  subscores: Subscores
  pages_scanned: number
  pages_failed: number
  rules?: Rule[]
  pages?: Page[]
  changes?: RuleChange
}

export interface Group {
  name: string
  agencies: number
  avg_score: number | null
}

export interface RuleCount {
  rule_id: string
  impact: Impact
  agencies: number
  pages: number
}

export interface Stats {
  agencies: number
  scanned: number
  avg_score: number | null
  grades: Record<string, number>
  by_level: Group[]
  by_state: Group[]
  top_rules: RuleCount[]
  states: string[]
  updated_at?: string
}
