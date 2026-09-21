import type {
  AgencyDetail,
  AgencyList,
  MailSummary,
  Scan,
  Stats,
  ThirdPartyList,
  Usage,
} from '../api/types'

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
  scanned: 1,
  page: 1,
  per_page: 50,
}

export const mailRecord = {
  domain: 'bundesregierung.de',
  provider: 'microsoft365' as const,
  mx: [
    {
      host: 'bundesregierung-de.mail.protection.outlook.com',
      preference: 10,
      provider: 'microsoft365' as const,
    },
  ],
  spf: 'v=spf1 include:spf.protection.outlook.com include:newsletter.example.net -all',
  spf_includes: ['spf.protection.outlook.com', 'newsletter.example.net'],
  spf_senders: ['microsoft365' as const],
  dmarc: 'v=DMARC1; p=reject',
  dmarc_policy: 'reject',
  checked_at: '2026-09-20T06:00:00Z',
}

export const agencyDetail: AgencyDetail = {
  ...agencyList.items[0],
  mail: mailRecord,
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
  lighthouse_score: 97,
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
  third_parties: [
    {
      host: 'fonts.gstatic.com',
      domain: 'gstatic.com',
      group: 'google-fonts' as const,
      public_body: false,
      phase: 'before_consent' as const,
      requests: 6,
      pages: 4,
    },
    {
      host: 'www.service.bund.de',
      domain: 'bund.de',
      group: 'unknown' as const,
      public_body: true,
      phase: 'before_consent' as const,
      requests: 1,
      pages: 1,
    },
    {
      host: 'www.youtube-nocookie.com',
      domain: 'youtube-nocookie.com',
      group: 'youtube' as const,
      public_body: false,
      phase: 'after_accepted' as const,
      requests: 2,
      pages: 1,
    },
  ],
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
  by_level: [{ name: 'bund', agencies: 36, scanned: 30, avg_score: 68.5 }],
  by_state: [{ name: 'Schleswig-Holstein', agencies: 4, scanned: 0, avg_score: null }],
  top_rules: [{ rule_id: 'color-contrast', impact: 'serious', agencies: 2, pages: 8 }],
  states: ['Bayern', 'Schleswig-Holstein'],
  updated_at: '2026-09-18T09:00:00Z',
}

export const usage: Usage = {
  since: '2026-09-13',
  until: '2026-09-19',
  visitors: 31,
  views: 84,
  days: [
    { day: '2026-09-18', visitors: 12, views: 30 },
    { day: '2026-09-19', visitors: 19, views: 54 },
  ],
  pages: [
    { key: '/', count: 50 },
    { key: '/behoerde/:slug', count: 24 },
    { key: '/methodik', count: 10 },
  ],
  agencies: [{ key: 'bundesregierung', count: 18 }],
  endpoints: [{ key: 'GET /api/v1/agencies', count: 120 }],
}

export const thirdParties: ThirdPartyList = {
  scanned: 120,
  items: [
    {
      host: 'fonts.gstatic.com',
      domain: 'gstatic.com',
      group: 'google-fonts',
      public_body: false,
      phase: 'before_consent',
      agencies: 48,
      pages: 900,
    },
    {
      host: 'fonts.googleapis.com',
      domain: 'googleapis.com',
      group: 'google-fonts',
      public_body: false,
      phase: 'before_consent',
      agencies: 44,
      pages: 870,
    },
    {
      host: 'www.google-analytics.com',
      domain: 'google-analytics.com',
      group: 'google-analytics',
      public_body: false,
      phase: 'before_consent',
      agencies: 12,
      pages: 60,
    },
    {
      host: 'www.youtube-nocookie.com',
      domain: 'youtube-nocookie.com',
      group: 'youtube',
      public_body: false,
      phase: 'after_accepted',
      agencies: 9,
      pages: 20,
    },
  ],
}

export const mailSummary: MailSummary = {
  total: 120,
  checked_at: '2026-09-20T06:00:00Z',
  by_provider: [
    { provider: 'self', agencies: 50, us_based: false },
    { provider: 'microsoft365', agencies: 36, us_based: true },
    { provider: 'public-it', agencies: 24, us_based: false },
    { provider: 'google', agencies: 10, us_based: true },
  ],
  by_state: [
    { name: 'Bayern', provider: 'microsoft365', agencies: 6, us_based: true },
    { name: 'Bayern', provider: 'self', agencies: 6, us_based: false },
    { name: 'Berlin', provider: 'self', agencies: 4, us_based: false },
  ],
  by_level: [
    { name: 'kommune', provider: 'microsoft365', agencies: 20, us_based: true },
    { name: 'bund', provider: 'public-it', agencies: 12, us_based: false },
  ],
}

export const betreiber = {
  name: 'Musterverein für digitale Teilhabe e. V.',
  street: 'Beispielweg 1',
  city: '12345 Musterstadt',
  country: 'Deutschland',
  email: 'post@example.org',
  hosting: 'Beispiel-Hoster GmbH, Falkenstein',
  accessibility_checked_at: '2026-09-20',
  complete: true,
}
