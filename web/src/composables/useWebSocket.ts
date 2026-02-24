import { ref, onUnmounted } from 'vue'
import type { WSMessage } from '../types'

export function useWebSocket(roomCode: string, token: string) {
  const connected = ref(false)
  const messages = ref<WSMessage[]>([])

  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let retryCount = 0
  const maxRetries = 5
  let closed = false

  function connect() {
    if (closed) return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/api/ws?token=${encodeURIComponent(token)}&room=${encodeURIComponent(roomCode)}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      retryCount = 0
    }

    ws.onclose = () => {
      connected.value = false
      if (!closed && retryCount < maxRetries) {
        const delay = Math.min(1000 * 2 ** retryCount, 16000)
        retryCount++
        reconnectTimer = setTimeout(connect, delay)
      }
    }

    ws.onerror = () => {
      ws?.close()
    }

    ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data)
        messages.value.push(msg)
      } catch { /* ignore malformed */ }
    }
  }

  function send(msg: WSMessage) {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg))
    }
  }

  function close() {
    closed = true
    if (reconnectTimer) clearTimeout(reconnectTimer)
    ws?.close()
  }

  connect()

  onUnmounted(close)

  return { connected, messages, send, close }
}
