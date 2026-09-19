import type { AgencyDetail, AgencyList, Scan, Stats } from '../api/types'

export const agencyList: AgencyList = {
  items: [
    {
      slug: 'bundesregierung',
      name: 'Bundesregierung',
      url: 'https://www.bundesregierung.de/',
      level: 'bund',
      score: 75.4,
      grade: 'C',
      delta: 4.9,
      pages: 4,
      scanned_at: '2026-09-18T09:00:00Z',
    },
    {
      slug: 'stadt-kiel',
      name: 'Kiel',
      url: 'https://www.kiel.de/',
      level: 'kommune',
      state: 'Schleswig-Holstein',
      score: null,
      pages: 0,
    },
  ],
  total: 2,
  page: 1,
  per_page: 50,
}

export const agencyDetail: AgencyDetail = {
  ...agencyList.items[0],
  subscores: { perceivable: 68, operable: 80, understandable: 90, robust: 85 },
  trend: {
    direction: 'improved',
    delta_last: 4.9,
    points_per_month: 5,
    months_to_grade_a: 3,
    scans: 3,
  },
  history: [
    { at: '2026-07-18T09:00:00Z', score: 60, grade: 'D' },
    { at: '2026-08-18T09:00:00Z', score: 70.5, grade: 'C' },
    { at: '2026-09-18T09:00:00Z', score: 75.4, grade: 'C' },
  ],
  latest_scan_id: 99,
}

export const latestScan: Scan = {
  id: 99,
  agency_slug: 'bundesregierung',
  agency_name: 'Bundesregierung',
  status: 'done',
  started_at: '2026-09-18T08:50:00Z',
  finished_at: '2026-09-18T09:00:00Z',
  score: 75.4,
  grade: 'C',
  subscores: { perceivable: 68, operable: 80, understandable: 90, robust: 85 },
  pages_scanned: 4,
  pages_failed: 1,
  pages_blocked: 0,
  rules: [
    {
      rule_id: 'color-contrast',
      impact: 'serious',
      principle: 'perceivable',
      help: 'Elemente müssen den Mindestkontrast erreichen',
      help_url: 'https://dequeuniversity.com/rules/axe/4.10/color-contrast',
      pages: 4,
      nodes: 12,
      sample_html: '<a href="/x">Datenschutz</a>',
    },
  ],
  pages: [
    {
      url: 'https://www.bundesregierung.de/',
      title: 'Startseite',
      depth: 0,
      is_entry: true,
      priority: false,
      dom_nodes: 900,
      score: 72,
      violations: 1,
    },
    {
      url: 'https://www.bundesregierung.de/kaputt',
      depth: 1,
      is_entry: false,
      priority: false,
      dom_nodes: 0,
      score: null,
      error: 'timeout',
      violations: 0,
    },
  ],
  changes: {
    fixed: [
      {
        rule_id: 'image-alt',
        impact: 'critical',
        principle: 'perceivable',
        help: 'Bilder brauchen eine Alternative',
        pages: 2,
        nodes: 3,
      },
    ],
    introduced: [],
    remaining: [],
  },
}

export const stats: Stats = {
  agencies: 92,
  scanned: 2,
  avg_score: 68.5,
  grades: { C: 1, D: 1 },
  by_level: [{ name: 'bund', agencies: 36, avg_score: 68.5 }],
  by_state: [{ name: 'Schleswig-Holstein', agencies: 4, avg_score: null }],
  top_rules: [{ rule_id: 'color-contrast', impact: 'serious', agencies: 2, pages: 8 }],
  states: ['Bayern', 'Schleswig-Holstein'],
  updated_at: '2026-09-18T09:00:00Z',
}
