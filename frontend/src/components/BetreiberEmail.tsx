/**
 * Die E-Mail-Adresse des Betreibers als Link — oder, solange sie fehlt, ein Wort
 * dazu. Ein `mailto:` ohne Adresse und ohne Text ist ein Link ohne Namen: genau der
 * Befund, den wir bei Behörden zählen, und die eigene Seite fiel damit im eigenen
 * Test durch.
 */
export function BetreiberEmail({ email }: { email: string }) {
  if (!email) return <span className="text-slate-700">noch nicht angegeben</span>
  return (
    <a href={`mailto:${email}`} className="break-all underline">
      {email}
    </a>
  )
}
