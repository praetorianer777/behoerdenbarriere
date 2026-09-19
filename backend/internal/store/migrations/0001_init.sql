CREATE TYPE agency_level AS ENUM ('bund', 'land', 'kreis', 'kommune');
CREATE TYPE scan_status AS ENUM ('queued', 'running', 'done', 'failed');
CREATE TYPE violation_impact AS ENUM ('critical', 'serious', 'moderate', 'minor');
CREATE TYPE wcag_principle AS ENUM ('perceivable', 'operable', 'understandable', 'robust');

CREATE TABLE agencies (
    id         bigserial PRIMARY KEY,
    slug       text NOT NULL UNIQUE,
    name       text NOT NULL,
    url        text NOT NULL,
    level      agency_level NOT NULL,
    state      text,
    category   text,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX agencies_level_idx ON agencies (level);
CREATE INDEX agencies_state_idx ON agencies (state);

CREATE TABLE scans (
    id                     bigserial PRIMARY KEY,
    agency_id              bigint NOT NULL REFERENCES agencies (id) ON DELETE CASCADE,
    status                 scan_status NOT NULL DEFAULT 'queued',
    started_at             timestamptz NOT NULL DEFAULT now(),
    finished_at            timestamptz,
    error                  text,
    score                  numeric(5, 2),
    grade                  char(1),
    score_perceivable      numeric(5, 2),
    score_operable         numeric(5, 2),
    score_understandable   numeric(5, 2),
    score_robust           numeric(5, 2),
    pages_scanned          integer NOT NULL DEFAULT 0,
    pages_failed           integer NOT NULL DEFAULT 0,
    config                 jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX scans_agency_started_idx ON scans (agency_id, started_at DESC);
CREATE INDEX scans_status_idx ON scans (status);

-- Only one scan per agency may be in flight; without this lock a manual rescan and the
-- scheduler would hit the same website twice at once.
CREATE UNIQUE INDEX scans_one_active_per_agency_idx
    ON scans (agency_id)
    WHERE status IN ('queued', 'running');

CREATE TABLE pages (
    id          bigserial PRIMARY KEY,
    scan_id     bigint NOT NULL REFERENCES scans (id) ON DELETE CASCADE,
    url         text NOT NULL,
    depth       integer NOT NULL DEFAULT 0,
    is_entry    boolean NOT NULL DEFAULT false,
    priority    boolean NOT NULL DEFAULT false,
    title       text,
    http_status integer,
    dom_nodes   integer NOT NULL DEFAULT 0,
    page_score  numeric(5, 2),
    load_ms     integer,
    error       text,
    scanned_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (scan_id, url)
);
CREATE INDEX pages_scan_idx ON pages (scan_id);

CREATE TABLE violations (
    id            bigserial PRIMARY KEY,
    page_id       bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    rule_id       text NOT NULL,
    impact        violation_impact NOT NULL,
    description   text,
    help          text,
    help_url      text,
    wcag_tags     text[] NOT NULL DEFAULT '{}',
    principle     wcag_principle NOT NULL,
    node_count    integer NOT NULL DEFAULT 1,
    sample_html   text,
    sample_target text
);
CREATE INDEX violations_page_idx ON violations (page_id);
CREATE INDEX violations_rule_idx ON violations (rule_id);

CREATE TABLE jobs (
    id         bigserial PRIMARY KEY,
    agency_id  bigint NOT NULL REFERENCES agencies (id) ON DELETE CASCADE,
    kind       text NOT NULL DEFAULT 'scan',
    run_after  timestamptz NOT NULL DEFAULT now(),
    attempts   integer NOT NULL DEFAULT 0,
    last_error text,
    locked_at  timestamptz,
    locked_by  text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX jobs_runnable_idx ON jobs (run_after) WHERE locked_at IS NULL;
CREATE UNIQUE INDEX jobs_one_per_agency_idx ON jobs (agency_id, kind);
