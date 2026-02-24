export interface User {
  id: number
  username: string
  token: string
}

export interface Player {
  id: number
  username: string
  ready: boolean
}

export interface Room {
  code: string
  host_id: number
  players: Player[]
  max_players: number
  status: string
}

export interface WSMessage {
  type: string
  payload?: Record<string, unknown>
}
