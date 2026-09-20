-- Was eine Domain über ihre E-Mail veröffentlicht: MX, SPF, DMARC. Öffentliches DNS,
-- nichts wird abgeklopft. Die Frage steht neben dem Score und nicht darin.
CREATE TABLE agency_mail (
    agency_id  bigint PRIMARY KEY REFERENCES agencies(id) ON DELETE CASCADE,
    checked_at timestamptz NOT NULL,
    -- Der Betreiber des bevorzugten MX. Abgeleitet, für die Auswertung; maßgeblich
    -- ist immer der Rohdatensatz daneben.
    provider   text NOT NULL,
    -- Der vollständige Datensatz, wie er abgefragt wurde. Ein MX-Eintrag sagt, wer die
    -- Post annimmt, nicht wer sie liest — wer unsere Einordnung prüfen will, muss den
    -- Rohwert sehen.
    record     jsonb NOT NULL,
    error      text
);

CREATE INDEX agency_mail_provider ON agency_mail (provider);
