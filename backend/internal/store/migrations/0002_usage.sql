-- Usage counters, aggregated per day. No event log: a request only ever raises a
-- counter, so the table grows with the days and not with the traffic, and nothing
-- about a single visit is kept.
--
-- kind 'visitor' holds one row per daily hash of address and user agent. The salt
-- behind it lives in the API process, rotates daily and is never stored, so the rows
-- cannot be traced back to anybody and not joined across days either.
CREATE TABLE usage_counters (
    day   date NOT NULL,
    kind  text NOT NULL,
    key   text NOT NULL,
    count bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (day, kind, key)
);
