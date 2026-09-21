import { laender } from '../laender'

/**
 * Die sechzehn Länder als Kacheln, ungefähr dort, wo sie auf der Karte liegen. Ein
 * Kartogramm ist eine Leseordnung, keine Landkarte: Es bildet keine Flächen ab, und
 * eine verzerrte Karte würde mehr behaupten, als die Daten hergeben.
 *
 * Die Kacheln wiederholen nur, was eine Tabelle daneben sagt. Für Screenreader zählt
 * die Tabelle — das Raster ist ausgeblendet, und wer es einsetzt, hat die Tabelle
 * mitzuliefern. Eine Grafik, die niemand vorlesen kann, ist Schmuck, und Schmuck darf
 * nicht die einzige Fassung einer Zahl sein.
 */

interface Props<T> {
  werte: Map<string, T>
  /** Was in der Kachel steht — kurz, es ist wenig Platz. */
  text: (wert: T | undefined) => string
  /** Farbe und Schriftfarbe der Kachel. Die Farbe ordnet nur; die Zahl trägt die Aussage. */
  klasse: (wert: T | undefined) => string
}

export function Kartogramm<T>({ werte, text, klasse }: Props<T>) {
  return (
    <div
      aria-hidden="true"
      className="mt-4 grid grid-cols-6 gap-1 sm:gap-2"
      style={{ gridTemplateRows: 'repeat(5, minmax(0, 1fr))' }}
    >
      {laender.map((land) => {
        const wert = werte.get(land.state)
        return (
          <div
            key={land.short}
            style={{ gridColumn: land.col, gridRow: land.row }}
            className={`rounded-md border border-slate-300 p-2 text-center ${klasse(wert)}`}
          >
            <span className="block text-xs font-medium">{land.short}</span>
            <span className="tabular block text-sm font-semibold">{text(wert)}</span>
          </div>
        )
      })}
    </div>
  )
}
