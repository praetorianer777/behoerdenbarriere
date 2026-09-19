interface Props {
  what: string
}

/**
 * Ladezustand und Fehler bekommen eine Live-Region: wer nicht sieht, dass sich die
 * Tabelle neu aufgebaut hat, muss es hören können.
 */
export function Loading({ what }: Props) {
  return (
    <p role="status" className="py-8 text-slate-700">
      {what} wird geladen …
    </p>
  )
}

export function LoadError({ what, error }: { what: string; error: unknown }) {
  const message = error instanceof Error ? error.message : String(error)
  return (
    <div role="alert" className="rounded-lg border border-grade-f bg-white p-4">
      <p className="font-semibold">{what} konnte nicht geladen werden.</p>
      <p className="mt-1 text-sm text-slate-700">{message}</p>
    </div>
  )
}
