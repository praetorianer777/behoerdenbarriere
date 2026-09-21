import type { Betreiber } from '../betreiber'

/**
 * Solange die Angaben zum Betreiber fehlen, steht das sichtbar auf der Seite. Ein
 * erfundenes Impressum wäre schlimmer als ein fehlendes, und ein stiller Platzhalter
 * fällt niemandem auf, bevor es zu spät ist.
 *
 * Solange die Angaben noch geladen werden, steht hier nichts — eine Warnung, die kurz
 * aufblitzt und verschwindet, wäre eine falsche. Ließen sie sich nicht laden, fehlen
 * sie für die Leserin genauso, und dann steht es hier.
 */
export function Platzhalterhinweis({
  betreiber,
  fehlgeschlagen = false,
}: {
  betreiber: Betreiber | undefined
  fehlgeschlagen?: boolean
}) {
  if (!fehlgeschlagen && (!betreiber || betreiber.complete)) return null

  return (
    <p role="alert" className="mt-4 rounded-lg border-2 border-grade-f bg-white p-4 font-medium">
      Diese Seite ist noch nicht vollständig: Die Angaben zum Betreiber fehlen. Sie werden in der{' '}
      <code>.env</code> der Installation gesetzt (<code>OPERATOR_NAME</code>,{' '}
      <code>OPERATOR_STREET</code>, <code>OPERATOR_CITY</code>, <code>OPERATOR_EMAIL</code>,{' '}
      <code>OPERATOR_HOSTING</code>), bevor die Website öffentlich erreichbar ist.
    </p>
  )
}
