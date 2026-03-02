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

// ---- Game types ----

export interface Card {
  id: string
  type: string
  color: string
  value?: number
}

export interface IdentityCard {
  type: string
  revealed: boolean
}

export interface PublicPlayer {
  user_id: number
  username: string
  alive: boolean
  identity_count: number
  unrevealed_count: number
  revealed_identities: string[] | null
  equipment: Card[] | null
  accuse_total: number
  detained: number
  has_hammer: boolean
  is_witch?: boolean
}

export interface TrialInfo {
  accused_id: number
  flipper_id: number
}

export interface GameEvent {
  message: string
  type?: string
  data?: Record<string, unknown>
}

export interface GameState {
  phase: string
  day_number: number
  your_id: number
  is_your_turn: boolean
  current_player_id: number
  hand: Card[] | null
  identities: IdentityCard[] | null
  is_witch: boolean
  players: PublicPlayer[]
  actions: string[] | null
  events: GameEvent[]
  draw_pile_count: number
  winner?: string
  fellow_witches?: number[]
  trial?: TrialInfo
  valid_targets?: number[]
}

export const cardNames: Record<string, string> = {
  accuse_1: '指控(1)', accuse_2: '指控(2)', accuse_3: '指控(3)',
  night: '夜晚', contagion: '传染',
  black_cat: '黑猫', sanctuary: '避难所', devotee: '信徒',
  frame: '嫁祸', arson: '纵火', detention: '拘留',
  defense: '辩护', robbery: '抢劫', curse: '诅咒',
}

export const identityNames: Record<string, string> = {
  villager: '村民', witch: '女巫', sheriff: '警长',
}

export const phaseNames: Record<string, string> = {
  day: '白天', night_witch: '夜晚·女巫行动', night_sheriff: '夜晚·警长行动',
  night_result: '天亮·自首阶段', trial: '审判', game_over: '游戏结束',
}

export const cardColorClass: Record<string, string> = {
  red: 'bg-red-500/20 text-red-300 border-red-500/40',
  green: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
  blue: 'bg-sky-500/20 text-sky-300 border-sky-500/40',
  black: 'bg-gray-700/50 text-gray-300 border-gray-500/40',
}

export const identityColorClass: Record<string, string> = {
  villager: 'bg-blue-500/20 text-blue-300 border-blue-500/40',
  witch: 'bg-purple-500/20 text-purple-300 border-purple-500/40',
  sheriff: 'bg-amber-500/20 text-amber-300 border-amber-500/40',
}

export function needsTwoTargets(card: Card): boolean {
  return card.type === 'frame' || card.type === 'robbery'
}

// ---- Stats types ----

export interface UserStats {
  total_games: number
  wins: number
  losses: number
  witch_games: number
  witch_wins: number
  villager_games: number
  villager_wins: number
}

export interface GameParticipant {
  user_id: number
  username: string
  is_witch: boolean
  alive: boolean
  won: boolean
  is_bot: boolean
}

export interface GameRecordItem {
  id: number
  room_code: string
  winner: string
  player_count: number
  day_count: number
  created_at: string
  participants: GameParticipant[]
  your_role: string
  won: boolean
  alive: boolean
}

export interface HistoryResponse {
  records: GameRecordItem[]
  page: number
  size: number
  total: number
}
