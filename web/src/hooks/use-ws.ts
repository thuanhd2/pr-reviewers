import { useEffect } from 'react'
import { Centrifuge } from 'centrifuge'
import { useQueryClient } from '@tanstack/react-query'

type WSEvent =
  | { type: 'pr.updated'; payload: any }
  | { type: 'review.created'; payload: any }
  | { type: 'review.posted'; payload: any }
  | { type: 'scheduler.tick'; payload: { lastRun: string; nextRun: string } }

export function useWebSocket() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const centrifuge = new Centrifuge(
      `${protocol}://${window.location.host}/connection/websocket`
    )

    const sub = centrifuge.newSubscription('pr-updates')

    sub.on('publication', (ctx) => {
      const event = ctx.data as WSEvent
      switch (event.type) {
        case 'pr.updated':
          queryClient.setQueryData(['prs', event.payload.id], event.payload)
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'review.created':
          queryClient.setQueryData(
            ['prs', event.payload.pull_request_id, 'reviews'],
            (old: any[]) => (old ? [...old, event.payload] : [event.payload])
          )
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'review.posted':
          queryClient.setQueryData(['reviews', event.payload.id], event.payload)
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'scheduler.tick':
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          queryClient.invalidateQueries({ queryKey: ['dashboard'] })
          break
      }
    })

    sub.subscribe()
    centrifuge.connect()

    return () => {
      centrifuge.disconnect()
    }
  }, [queryClient])
}
