CREATE TYPE agency_source AS ENUM ('seed', 'wikidata');

-- Woher ein Eintrag stammt, entscheidet, wer ihn überschreiben darf: Die von Hand
-- gepflegte Liste ist geprüft, der Import ist es nicht. Ein Import darf deshalb
-- ergänzen, aber nichts Geprüftes übermalen.
ALTER TABLE agencies ADD COLUMN source agency_source NOT NULL DEFAULT 'seed';

-- Die Kennung bei der Quelle, damit ein Eintrag nachvollziehbar bleibt und ein
-- zweiter Import ihn wiedererkennt, auch wenn sich der Name ändert.
ALTER TABLE agencies ADD COLUMN external_id text;
CREATE UNIQUE INDEX agencies_external_id_idx ON agencies (source, external_id)
    WHERE external_id IS NOT NULL;
