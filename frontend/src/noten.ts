export const noten = ['A', 'B', 'C', 'D', 'E', 'F']

/**
 * Der Satz über der Verteilung, aus den Zahlen gerechnet statt geschrieben: Ein Text,
 * der „die häufigste Note ist F" behauptet, muss aufhören, das zu behaupten, sobald es
 * nicht mehr stimmt.
 */
export function notenSatz(grades: Record<string, number>, scanned: number): string {
  if (scanned === 0) return 'Noch keine Behörde ist geprüft.'
  const meiste = Math.max(...noten.map((note) => grades[note] ?? 0))
  const spitze = noten.filter((note) => (grades[note] ?? 0) === meiste)
  const anteil = Math.round((meiste / scanned) * 100)
  if (spitze.length === 1) {
    return `Die häufigste Note ist ${spitze[0]}: ${meiste} von ${scanned} geprüften Behörden, ${anteil} Prozent.`
  }
  return `Die häufigsten Noten sind ${spitze.join(' und ')}, je ${meiste} von ${scanned} geprüften Behörden.`
}
