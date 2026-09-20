import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router'

import './index.css'
import { Layout } from './components/Layout'
import { Ranking } from './pages/Ranking'
import { AgencyPage } from './pages/Agency'
import { Dashboard } from './pages/Dashboard'
import { Methodology } from './pages/Methodology'
import { Impressum } from './pages/Impressum'
import { Datenschutz } from './pages/Datenschutz'
import { Barrierefreiheit } from './pages/Barrierefreiheit'
import { Statistics } from './pages/Statistics'
import { Drittanbieter } from './pages/Drittanbieter'
import { NotFound } from './pages/NotFound'

const queryClient = new QueryClient({
  defaultOptions: {
    // Die Daten ändern sich höchstens einmal pro Woche; häufiger nachzufragen
    // belastet nur die eigene API.
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Ranking />} />
            <Route path="behoerde/:slug" element={<AgencyPage />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="methodik" element={<Methodology />} />
            <Route path="impressum" element={<Impressum />} />
            <Route path="datenschutz" element={<Datenschutz />} />
            <Route path="barrierefreiheit" element={<Barrierefreiheit />} />
            <Route path="statistik" element={<Statistics />} />
            <Route path="drittanbieter" element={<Drittanbieter />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
