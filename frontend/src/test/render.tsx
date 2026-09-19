import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router'

/**
 * Rendert eine Seite mit Router und Query-Client, aber ohne Wiederholungen: ein
 * fehlschlagender Aufruf soll im Test sofort als Fehler ankommen, nicht nach Sekunden.
 */
export function renderPage(element: ReactElement, { path = '/', route = '/' } = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path={path} element={element} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
