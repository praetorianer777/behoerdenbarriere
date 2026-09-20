-- Googles Lighthouse-Score neben dem eigenen: Er ist der einzige etablierte, offen
-- dokumentierte Wert und rechnet bewusst anders — jedes Audit besteht ganz oder gar
-- nicht. Wo beide Zahlen auseinanderlaufen, ist das ein Befund.
ALTER TABLE scans ADD COLUMN lighthouse_score numeric(5, 2);
ALTER TABLE scans ADD COLUMN lighthouse_failed text[];
