import { useEffect, useRef, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getAccessToken } from '@/lib/auth'

interface HeartbeatEvent {
  edge_id: string
  timestamp: string
  cpu_pct: number
  mem_pct: number
  disk_pct: number
}

interface WsMessage {
  type: string
  payload: HeartbeatEvent
}

export function useEdgeWebSocket() {
  const qc = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const mountedRef = useRef(true)

  const connect = useCallback(() => {
    const token = getAccessToken()
    if (!token) return

    const wsUrl = window.location.origin
      .replace('http://', 'ws://')
      .replace('https://', 'wss://')

    const ws = new WebSocket(`${wsUrl}/api/v1/ws/edges?access_token=${encodeURIComponent(token)}`)
    wsRef.current = ws

    ws.onopen = () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
    }

    ws.onmessage = (event) => {
      try {
        const msg: WsMessage = JSON.parse(event.data as string)
        if (msg.type === 'edge.heartbeat') {
          qc.invalidateQueries({ queryKey: ['edges'] })
        }
      } catch {
        // ignore parse errors
      }
    }

    ws.onclose = () => {
      if (!mountedRef.current) return  // don't reconnect after unmount
      reconnectTimerRef.current = setTimeout(connect, 5000)
    }

    ws.onerror = () => {
      ws.close()
    }
  }, [qc])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
      wsRef.current?.close()
    }
  }, [connect])
}
