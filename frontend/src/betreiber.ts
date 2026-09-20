/**
 * Angaben zum Betreiber für Impressum und Datenschutzerklärung.
 *
 * VOR DEM ÖFFENTLICHEN BETRIEB AUSFÜLLEN. Ein Impressum ist nach § 5 DDG Pflicht,
 * und ein erfundenes wäre schlimmer als keins. Solange hier Platzhalter stehen,
 * weisen die Seiten sichtbar darauf hin — das ist Absicht und soll stören.
 */
export const betreiber = {
  name: 'BITTE AUSFÜLLEN — Name der Betreiberin oder des Betreibers',
  strasse: 'BITTE AUSFÜLLEN — Straße und Hausnummer',
  ort: 'BITTE AUSFÜLLEN — Postleitzahl und Ort',
  land: 'Deutschland',
  email: 'BITTE AUSFÜLLEN — E-Mail-Adresse',
  telefon: '',
  /** Optional: Umsatzsteuer-Identifikationsnummer, falls vorhanden. */
  ustId: '',
  /** Wer die Seite hostet — gehört in die Datenschutzerklärung. */
  hosting: 'BITTE AUSFÜLLEN — Hosting-Anbieter und Standort',
  /** Datum der letzten Selbstprüfung auf Barrierefreiheit, ISO-Format. */
  barrierefreiheitGeprueftAm: '2026-09-20',
}

const platzhalter = 'BITTE AUSFÜLLEN'

/** Sagt, ob noch Platzhalter in den Angaben stehen. */
export function angabenUnvollstaendig(): boolean {
  return Object.values(betreiber).some((wert) => wert.startsWith(platzhalter))
}
