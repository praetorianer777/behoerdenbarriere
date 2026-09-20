-- Welche fremden Hosts eine Seite kontaktiert, ist eine andere Frage über dieselbe
-- Website — wie die Erklärung zur Barrierefreiheit. Die Tabelle steht deshalb neben
-- den Verstößen und fließt nicht in den Score ein.
CREATE TYPE contact_phase AS ENUM ('before_consent', 'after_declined', 'after_accepted');

-- Gespeichert wird der beobachtete Hostname, nicht seine Einordnung: Wem die Daten am
-- Ende gehören, sagt der Name nicht, und unsere Zuordnung darf sich verbessern können,
-- ohne dass dafür jede Behörde neu gecrawlt werden muss.
CREATE TABLE page_contacts (
    id        bigserial PRIMARY KEY,
    page_id   bigint NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    host      text NOT NULL,
    phase     contact_phase NOT NULL,
    requests  integer NOT NULL DEFAULT 1,
    UNIQUE (page_id, host, phase)
);

CREATE INDEX page_contacts_page ON page_contacts (page_id);
CREATE INDEX page_contacts_host ON page_contacts (host);
