import type { Page } from '@playwright/test'

/**
 * Die E2E-Tests prüfen die Oberfläche, nicht die API. Antworten kommen deshalb aus
 * festen Daten: ein Lauf, der von einem echten Scan abhängt, schlägt irgendwann fehl,
 * weil sich eine Behörde geändert hat — und sagt dann nichts über den Code aus.
 */
const agencies = {
  items: [
    {
      slug: 'bmwsb',
      name: 'Bundesministerium für Wohnen, Stadtentwicklung und Bauwesen',
      url: 'https://www.bmwsb.bund.de/',
      level: 'bund',
      score: 74.5,
      grade: 'C',
      delta: 4.9,
      pages: 4,
      scanned_at: '2026-09-18T09:00:00Z',
      lighthouse_score: 97,
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

const agencyDetail = {
  ...agencies.items[0],
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
    { at: '2026-09-18T09:00:00Z', score: 74.5, grade: 'C' },
  ],
  latest_scan_id: 99,
}

const latestScan = {
  id: 99,
  agency_slug: 'bmwsb',
  agency_name: agencies.items[0].name,
  status: 'done',
  started_at: '2026-09-18T08:50:00Z',
  finished_at: '2026-09-18T09:00:00Z',
  score: 74.5,
  grade: 'C',
  subscores: { perceivable: 68, operable: 80, understandable: 90, robust: 85 },
  pages_scanned: 4,
  pages_failed: 1,
  pages_blocked: 1,
  lighthouse_score: 97,
  lighthouse_failed: ['color-contrast'],
  rules: [
    {
      rule_id: 'color-contrast',
      impact: 'serious',
      principle: 'perceivable',
      help: 'Elemente müssen den Mindestkontrast erreichen',
      help_url: 'https://dequeuniversity.com/rules/axe/4.10/color-contrast',
      pages: 4,
      nodes: 12,
      sample_html:
        '<a href="/portal/DE/datenschutzerklaerung/datenschutzerklaerung_node.html">Datenschutzerklärung</a>',
    },
  ],
  pages: [
    {
      url: 'https://www.bmwsb.bund.de/',
      title: 'Startseite',
      depth: 0,
      is_entry: true,
      priority: false,
      dom_nodes: 900,
      score: 72,
      violations: 1,
      consent: 'declined',
    },
    {
      url: 'https://www.bmwsb.bund.de/SharedDocs/faqs/Webs/BMWSB/DE/barrierefreiheit/erklaerung-zur-barrierefreiheit.html',
      depth: 1,
      is_entry: false,
      priority: true,
      dom_nodes: 400,
      score: null,
      violations: 0,
      consent: 'blocked',
    },
  ],
  explanation: {
    score: 74.5,
    grade: 'C',
    pages: [
      {
        url: 'https://www.bundesregierung.de/',
        title: 'Startseite',
        score: 72,
        dom_nodes: 900,
        density: 4.2,
        weight: 3,
        is_entry: true,
        priority: false,
        reasons: [
          {
            rule_id: 'color-contrast',
            impact: 'serious',
            principle: 'perceivable',
            help: 'Elemente müssen den Mindestkontrast erreichen',
            help_url: 'https://dequeuniversity.com/rules/axe/4.10/color-contrast',
            nodes: 12,
            weight: 6,
            penalty: 20.9,
            share: 0.78,
            points_if_fixed: 21.4,
          },
          {
            rule_id: 'region',
            impact: 'moderate',
            principle: 'robust',
            help: 'Inhalte müssen in Landmarken liegen',
            nodes: 1,
            weight: 3,
            penalty: 3,
            share: 0.22,
            points_if_fixed: 3.1,
          },
        ],
      },
    ],
    improvements: [
      {
        rule_id: 'color-contrast',
        impact: 'serious',
        principle: 'perceivable',
        help: 'Elemente müssen den Mindestkontrast erreichen',
        help_url: 'https://dequeuniversity.com/rules/axe/4.10/color-contrast',
        pages: 4,
        nodes: 12,
        points_if_fixed: 16.1,
      },
    ],
  },
  statement: {
    state: 'found' as const,
    url: 'https://www.bmwsb.bund.de/erklaerung-zur-barrierefreiheit',
    findings: [
      { requirement: 'reachable', met: true },
      {
        requirement: 'conformance',
        met: true,
        evidence: '… ist mit der BITV teilweise vereinbar …',
      },
      { requirement: 'shortcomings', met: true, evidence: '… nicht barrierefreie Inhalte: PDF …' },
      { requirement: 'date', met: true, evidence: '… erstellt am 14.03.2026 …' },
      { requirement: 'feedback', met: true, evidence: '… Barrieren melden …' },
      { requirement: 'enforcement', met: false },
    ],
  },
  changes: { fixed: [], introduced: [], remaining: [] },
}

const stats = {
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

const usage = {
  days: [{ day: '2026-09-19', visitors: 12, views: 34 }],
  pages: [{ key: '/', count: 20 }],
  agencies: [{ key: 'bmwsb', count: 8 }],
  endpoints: [{ key: 'GET /api/v1/agencies', count: 30 }],
  visitors: 12,
  views: 34,
}

export async function stubApi(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const json = (body: unknown) => route.fulfill({ json: body })

    if (path.endsWith('/stats')) return json(stats)
    if (path.endsWith('/usage')) return json(usage)
    if (path.endsWith('/view')) return route.fulfill({ status: 204, body: '' })
    if (path.endsWith('/scans/latest')) return json(latestScan)
    if (path.match(/\/agencies\/[^/]+$/)) return json(agencyDetail)
    if (path.endsWith('/agencies')) return json(agencies)
    return json({})
  })
}

export const routes = ['/', '/dashboard', '/methodik', '/statistik', '/behoerde/bmwsb']
