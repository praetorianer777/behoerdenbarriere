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
  lighthouse_score?: number
  // Der Wert stützt sich auf zu wenige Seiten, um die Website zu beschreiben — etwa
  // weil die robots.txt drei Minuten Pause zwischen zwei Anfragen verlangt.
  provisional?: boolean
  /** Gemessen wurde überwiegend die Einwilligungsabfrage, nicht die Seite dahinter. */
  obscured?: boolean
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

export interface Failure {
  reason: string
  at: string
}

export interface AgencyDetail extends Agency {
  // Warum die letzte Prüfung nichts ergeben hat. Kein Urteil über die Behörde.
  failure?: Failure
  subscores: Subscores
  mail?: MailRecord
  trend: TrendSummary
  // Eine ältere API schickt hier null statt einer leeren Liste. Der Typ sagt das,
  // damit niemand wieder blind eine Länge darauf liest — das hat die ganze Seite
  // weiß werden lassen.
  history?: TrendPoint[] | null
  latest_scan_id?: number
}

export type MailProvider =
  | 'microsoft365'
  | 'google'
  | 'proofpoint'
  | 'barracuda'
  | 'cloudflare'
  | 'mimecast'
  | 'hornetsecurity'
  | 'sophos'
  | 'symantec'
  | 'telekom'
  | 'ionos'
  | 'strato'
  | 'hetzner'
  | 'netcup'
  | 'mailbox-org'
  | 'retarus'
  | 'public-it'
  | 'self'
  | 'none'
  | 'unknown'

export interface MailExchanger {
  host: string
  preference: number
  provider: MailProvider
}

export interface MailRecord {
  domain: string
  mx?: MailExchanger[]
  provider: MailProvider
  spf?: string
  spf_includes?: string[]
  spf_senders?: MailProvider[]
  dmarc?: string
  dmarc_policy?: string
  checked_at: string
  error?: string
  us_based?: boolean
  // filter: Der MX filtert, er verwahrt nicht. Wo das zutrifft, sagt der Eintrag, wo
  // Post geprüft wird, und nichts darüber, wo sie liegt.
  filter?: boolean
}

export interface MailCount {
  name?: string
  provider: MailProvider
  agencies: number
  us_based: boolean
  filter?: boolean
}

export interface MailSummary {
  total: number
  checked_at?: string
  by_provider: MailCount[]
  by_state: MailCount[]
  by_level: MailCount[]
}

export interface AgencyList {
  items: Agency[]
  total: number
  /** Wie viele der gefundenen Behörden ein Ergebnis haben. */
  scanned: number
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
  consent?: 'none' | 'declined' | 'accepted' | 'blocked'
  violations: number
}

export interface Reason {
  rule_id: string
  impact: Impact
  principle: string
  help?: string
  help_url?: string
  nodes: number
  weight: number
  penalty: number
  share: number
  points_if_fixed: number
}

export interface PageExplanation {
  url: string
  title?: string
  score: number
  dom_nodes: number
  density: number
  weight: number
  is_entry: boolean
  priority: boolean
  reasons: Reason[]
}

export interface Improvement {
  rule_id: string
  impact: Impact
  principle: string
  help?: string
  help_url?: string
  pages: number
  nodes: number
  points_if_fixed: number
}

export interface SiteExplanation {
  score: number
  grade: string
  pages: PageExplanation[]
  improvements: Improvement[]
}

export type Requirement =
  'reachable' | 'conformance' | 'shortcomings' | 'date' | 'feedback' | 'enforcement'

export interface StatementFinding {
  requirement: Requirement
  met: boolean
  evidence?: string
}

export type StatementState = 'missing' | 'unreadable' | 'found'

export interface StatementResult {
  state: StatementState
  url?: string
  findings: StatementFinding[]
}

export type ContactPhase = 'before_consent' | 'after_declined' | 'after_accepted'

export type ContactGroup =
  | 'google-fonts'
  | 'google-analytics'
  | 'google-maps'
  | 'google-ads'
  | 'google-other'
  | 'youtube'
  | 'vimeo'
  | 'meta'
  | 'x'
  | 'linkedin'
  | 'matomo'
  | 'etracker'
  | 'consent-tool'
  | 'cdn'
  | 'unknown'

export interface ThirdParty {
  host: string
  domain: string
  group: ContactGroup
  public_body: boolean
  self_hosted?: boolean
  phase: ContactPhase
  requests?: number
  pages: number
}

export interface ThirdPartyReach {
  host: string
  domain: string
  group: ContactGroup
  public_body: boolean
  phase: ContactPhase
  agencies: number
  pages: number
}

export interface ThirdPartyList {
  items: ThirdPartyReach[]
  scanned: number
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
  pages_blocked: number
  lighthouse_score?: number
  lighthouse_failed?: string[]
  rules?: Rule[]
  pages?: Page[]
  changes?: RuleChange
  explanation?: SiteExplanation
  statement?: StatementResult
  third_parties?: ThirdParty[]
}

export interface Group {
  name: string
  agencies: number
  /** Wie viele davon geprüft sind — der Durchschnitt gilt nur für diese. */
  scanned: number
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

export interface UsageDay {
  day: string
  visitors: number
  views: number
}

export interface UsageKey {
  key: string
  count: number
}

export interface Usage {
  since: string
  until: string
  visitors: number
  views: number
  days: UsageDay[]
  pages: UsageKey[]
  agencies: UsageKey[]
  endpoints: UsageKey[]
}
