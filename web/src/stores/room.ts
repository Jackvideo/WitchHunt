import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Room } from '../types'
import { useAuthStore } from './auth'

export const useRoomStore = defineStore('room', () => {
  const currentRoom = ref<Room | null>(null)
  const error = ref('')

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

  return { currentRoom, error, createRoom, joinRoom, clear }
})
