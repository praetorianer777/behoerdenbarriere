import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  error: Error | null
}

/**
 * Fängt einen Fehler beim Aufbauen der Seite ab.
 *
 * Ohne diese Grenze nimmt React bei jeder Ausnahme den gesamten Baum vom Bildschirm:
 * Die Seite wird weiß, ohne Meldung, ohne Hinweis, was fehlt. Genau das ist passiert,
 * als die API für eine noch ungeprüfte Behörde `null` statt einer leeren Liste
 * schickte — ein Feld, und die ganze Website war leer.
 *
 * Ein Fehler soll sichtbar sein und die Seite tragen, nicht sie verschlucken.
 */
export class Fehlergrenze extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // In der Konsole, damit beim Nachsehen etwas dasteht — gemeldet wird nichts nach
    // außen, wir schicken keine Daten an Dritte.
    console.error('Fehler beim Aufbau der Seite:', error, info.componentStack)
  }

  render() {
    if (!this.state.error) return this.props.children

    return (
      <div role="alert" className="mx-auto max-w-2xl rounded-lg border border-grade-f bg-white p-6">
        <h1 className="text-xl font-bold">Diese Seite konnte nicht aufgebaut werden</h1>
        <p className="mt-2 text-slate-700">
          Ein Fehler in der Anwendung hat die Darstellung abgebrochen. Das liegt an uns, nicht an
          Ihrem Gerät.
        </p>
        <p className="mt-2 text-slate-700">
          Laden Sie die Seite neu oder gehen Sie zurück zum{' '}
          <a href="/" className="underline">
            Ranking
          </a>
          . Bleibt es dabei, ist es ein Fehler, den wir beheben müssen.
        </p>
        {/* Die technische Meldung gehört dazu: Wer sie weitergibt, hilft beim
            Finden — und wer sie nicht braucht, liest sie nicht. */}
        <p className="mt-4 text-sm break-words text-slate-600">
          <code>{this.state.error.message}</code>
        </p>
      </div>
    )
  }
}
