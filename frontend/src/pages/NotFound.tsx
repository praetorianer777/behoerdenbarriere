import { Link } from 'react-router'

export function NotFound() {
  return (
    <>
      <h1 className="text-3xl font-bold">Seite nicht gefunden</h1>
      <p className="mt-2">
        Diese Adresse gibt es nicht.{' '}
        <Link to="/" className="underline">
          Zum Ranking
        </Link>
      </p>
    </>
  )
}
