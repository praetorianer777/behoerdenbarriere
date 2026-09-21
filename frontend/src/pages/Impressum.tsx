import { useBetreiber } from '../betreiber'
import { Loading, LoadError } from '../components/Loading'
import { Platzhalterhinweis } from '../components/Platzhalterhinweis'

export function Impressum() {
  const angaben = useBetreiber()
  if (angaben.isPending) return <Loading what="Die Angaben zum Betreiber" />
  if (angaben.isError) return <LoadError what="Die Angaben zum Betreiber" error={angaben.error} />
  const betreiber = angaben.data

  return (
    <div className="max-w-3xl">
      <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Impressum</h1>
      <Platzhalterhinweis betreiber={betreiber} />

      <h2 className="mt-8 text-xl font-semibold">Angaben nach § 5 DDG</h2>
      <address className="mt-2 not-italic">
        {betreiber.name}
        <br />
        {betreiber.street}
        <br />
        {betreiber.city}
        <br />
        {betreiber.country}
      </address>

      <h2 className="mt-8 text-xl font-semibold">Kontakt</h2>
      <p className="mt-2">
        E-Mail:{' '}
        <a href={`mailto:${betreiber.email}`} className="break-all underline">
          {betreiber.email}
        </a>
        {betreiber.phone && (
          <>
            <br />
            Telefon: {betreiber.phone}
          </>
        )}
      </p>

      {betreiber.vat_id && (
        <>
          <h2 className="mt-8 text-xl font-semibold">Umsatzsteuer-Identifikationsnummer</h2>
          <p className="mt-2">{betreiber.vat_id}</p>
        </>
      )}

      <h2 className="mt-8 text-xl font-semibold">Verantwortlich für den Inhalt</h2>
      <p className="mt-2">{betreiber.name}, Anschrift wie oben.</p>

      <h2 className="mt-8 text-xl font-semibold">Zu den veröffentlichten Daten</h2>
      <p className="mt-2">
        Diese Website veröffentlicht Messergebnisse zu öffentlich erreichbaren Websites deutscher
        Behörden. Die Ergebnisse stammen aus automatisierten Prüfungen und werden ohne Gewähr
        veröffentlicht; wie sie zustande kommen und was sie nicht zeigen, steht auf der Seite{' '}
        <a href="/methodik" className="underline">
          Methodik
        </a>
        .
      </p>
      <p className="mt-4">
        Wer einen Fehler in einem Ergebnis sieht, möge sich melden — an die oben genannte Adresse.
        Behörden, die nicht geprüft werden möchten, werden auf Zuruf aus der Liste genommen.
      </p>
    </div>
  )
}
