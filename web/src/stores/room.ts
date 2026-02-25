import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Room } from '../types'
import { useAuthStore } from './auth'

export const useRoomStore = defineStore('room', () => {
  const currentRoom = ref<Room | null>(null)
  const cardDescriptions = ref<Record<string, string>>({})
  const error = ref('')

  async function fetchCardDescriptions() {
    // If already fetched, don't fetch again
    if (Object.keys(cardDescriptions.value).length > 0) return

    const auth = useAuthStore()
    try {
      const res = await auth.apiFetch('/api/cards')
      if (res.ok) {
        const data = await res.json()
        data.forEach((d: any) => {
          cardDescriptions.value[d.type] = d.description
        })
      }
    } catch {
      // ignore
    }
  }

  async function createRoom(maxPlayers: number) {
    error.value = ''
    const auth = useAuthStore()
    try {
      const res = await auth.apiFetch('/api/room/create', {
        method: 'POST',
        body: JSON.stringify({ max_players: maxPlayers }),
      })
      const data = await res.json()
      if (!res.ok) {
        error.value = data.error || 'failed to create room'
        return null
      }
      currentRoom.value = data
      return data as Room
    } catch {
      error.value = 'network error'
      return null
    }
  }

  async function joinRoom(code: string) {
    error.value = ''
    const auth = useAuthStore()
    try {
      const res = await auth.apiFetch('/api/room/join', {
        method: 'POST',
        body: JSON.stringify({ code: code.toUpperCase() }),
      })
      const data = await res.json()
      if (!res.ok) {
        error.value = data.error || 'failed to join room'
        return null
      }
      currentRoom.value = data
      return data as Room
    } catch {
      error.value = 'network error'
      return null
    }
  }

  function clear() {
    currentRoom.value = null
    error.value = ''
  }

  return { currentRoom, cardDescriptions, error, createRoom, joinRoom, fetchCardDescriptions, clear }
})
