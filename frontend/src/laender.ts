/**
 * Die sechzehn Länder mit dem Platz ihrer Kachel im Kartogramm — Spalte und Zeile in
 * einem Raster von sechs mal fünf, ungefähr dort, wo das Land auf der Karte liegt.
 * Eigene Datei, damit der Baustein daneben nur Komponenten exportiert.
 */
export const laender: { state: string; short: string; col: number; row: number }[] = [
  { state: 'Schleswig-Holstein', short: 'SH', col: 3, row: 1 },
  { state: 'Mecklenburg-Vorpommern', short: 'MV', col: 5, row: 1 },
  { state: 'Bremen', short: 'HB', col: 2, row: 2 },
  { state: 'Hamburg', short: 'HH', col: 3, row: 2 },
  { state: 'Niedersachsen', short: 'NI', col: 4, row: 2 },
  { state: 'Brandenburg', short: 'BB', col: 5, row: 2 },
  { state: 'Berlin', short: 'BE', col: 6, row: 2 },
  { state: 'Nordrhein-Westfalen', short: 'NW', col: 3, row: 3 },
  { state: 'Sachsen-Anhalt', short: 'ST', col: 4, row: 3 },
  { state: 'Sachsen', short: 'SN', col: 5, row: 3 },
  { state: 'Rheinland-Pfalz', short: 'RP', col: 3, row: 4 },
  { state: 'Hessen', short: 'HE', col: 4, row: 4 },
  { state: 'Thüringen', short: 'TH', col: 5, row: 4 },
  { state: 'Saarland', short: 'SL', col: 2, row: 5 },
  { state: 'Baden-Württemberg', short: 'BW', col: 3, row: 5 },
  { state: 'Bayern', short: 'BY', col: 4, row: 5 },
]
