import { useEffect } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router'

import { api } from '../api/client'
import { Fehlergrenze } from './Fehlergrenze'

const navigation = [
  { to: '/', label: 'Ranking', end: true },
  { to: '/dashboard', label: 'Überblick', end: false },
  { to: '/methodik', label: 'Methodik', end: false },
  { to: '/drittanbieter', label: 'Drittanbieter', end: false },
  { to: '/e-mail', label: 'E-Mail', end: false },
  { to: '/statistik', label: 'Statistik', end: false },
]

export function Layout() {
  const { pathname } = useLocation()

  useEffect(() => {
    void api.view(pathname)
  }, [pathname])

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      {/* Der Sprunglink ist das erste fokussierbare Element der Seite und wird
          sichtbar, sobald er den Fokus hat. */}
      <a
        href="#inhalt"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-slate-900 focus:px-4 focus:py-2 focus:text-white"
      >
        Zum Inhalt springen
      </a>

      {/* Der Balken oben ist die einzige Fläche in der Akzentfarbe: Er macht die Seite
          wiedererkennbar, ohne dass irgendwo Text auf Farbe stehen müsste. */}
      <header className="border-t-4 border-t-akzent border-b border-b-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-4">
          <NavLink
            to="/"
            className="flex items-center gap-2 text-xl font-bold tracking-tight text-slate-900"
          >
            <span aria-hidden="true" className="inline-block h-6 w-1.5 rounded-sm bg-akzent" />
            Behördenbarriere
          </NavLink>
          <nav aria-label="Hauptnavigation">
            <ul className="flex flex-wrap gap-1">
              {navigation.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    end={item.end}
                    className={({ isActive }) =>
                      `flex min-h-11 items-center rounded-md px-3 py-2 text-base font-medium ${
                        isActive ? 'bg-akzent text-white' : 'text-slate-700 hover:bg-akzent-hell'
                      }`
                    }
                  >
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </header>

      <main id="inhalt" className="mx-auto max-w-6xl px-4 py-8">
        {/* Die Grenze steht hier und nicht nur ganz außen: Bricht eine Seite ab,
            bleiben Navigation und Fuß stehen, und man kommt weiter. */}
        <Fehlergrenze>
          <Outlet />
        </Fehlergrenze>
      </main>

      <footer className="border-t border-slate-200 bg-white">
        <div className="mx-auto max-w-6xl px-4 py-6 text-sm text-slate-700">
          <p>
            Automatisierte Prüfung nach WCAG 2.1 AA. Automatische Tests erfassen einen Teil der
            Kriterien — der Wert ist ein Hinweis, kein BITV-Prüfbericht.{' '}
            <NavLink to="/methodik" className="underline">
              Wie wir prüfen
            </NavLink>
          </p>
          {/* Die Erklärung zur Barrierefreiheit steht im Fuß und damit auf jeder Seite
              — genau das verlangen wir von den geprüften Behörden. */}
          <nav aria-label="Rechtliches" className="mt-4">
            <ul className="flex flex-wrap gap-x-6 gap-y-2">
              <li>
                <NavLink to="/impressum" className="inline-flex min-h-11 items-center underline">
                  Impressum
                </NavLink>
              </li>
              <li>
                <NavLink to="/datenschutz" className="inline-flex min-h-11 items-center underline">
                  Datenschutz
                </NavLink>
              </li>
              <li>
                <NavLink
                  to="/barrierefreiheit"
                  className="inline-flex min-h-11 items-center underline"
                >
                  Erklärung zur Barrierefreiheit
                </NavLink>
              </li>
              <li>
                <NavLink to="/statistik" className="inline-flex min-h-11 items-center underline">
                  Statistik
                </NavLink>
              </li>
            </ul>
          </nav>
        </div>
      </footer>
    </div>
  )
}
