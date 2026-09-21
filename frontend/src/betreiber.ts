import { useQuery } from '@tanstack/react-query'

import { api } from './api/client'
import type { Betreiber } from './api/types'

/**
 * Die Angaben zum Betreiber für Impressum und Datenschutzerklärung. Sie kommen vom
 * Server, der sie aus seiner Umgebung liest: Wer die veröffentlichten Images nutzt,
 * kann den Code nicht anfassen — und soll es für sein eigenes Impressum auch nicht
 * müssen. Solange etwas fehlt, weist die Seite sichtbar darauf hin; erfunden wird
 * nichts, ein erfundenes Impressum wäre schlimmer als ein fehlendes.
 */
export function useBetreiber() {
  return useQuery({
    queryKey: ['betreiber'],
    queryFn: ({ signal }) => api.betreiber(signal),
    staleTime: Infinity,
  })
}

export type { Betreiber }
