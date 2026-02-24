<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'
import { useWebSocket } from '../composables/useWebSocket'
import type { Player, WSMessage } from '../types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const roomStore = useRoomStore()

const roomCode = route.params.code as string
const token = auth.user?.token ?? ''

if (!token) {
  router.push('/login')
}

const { connected, messages, send } = useWebSocket(roomCode, token)

const players = computed(() => {
  const base: Player[] = roomStore.currentRoom?.players ? [...roomStore.currentRoom.players] : []

  for (const msg of messages.value) {
    if (msg.type === 'player_join' && msg.payload) {
      const id = msg.payload.user_id as number
      if (!base.find((p) => p.id === id)) {
        base.push({ id, username: msg.payload.username as string, ready: false })
      }
    }
    if (msg.type === 'player_leave' && msg.payload) {
      const idx = base.findIndex((p) => p.id === (msg.payload!.user_id as number))
      if (idx !== -1) base.splice(idx, 1)
    }
  }
  return base
})

const isHost = computed(() => roomStore.currentRoom?.host_id === auth.user?.id)

function leaveRoom() {
  roomStore.clear()
  router.push('/lobby')
}

function sendAction(action: string) {
  send({ type: 'game_action', payload: { action } })
}

watch(messages, (msgs) => {
  const last = msgs[msgs.length - 1] as WSMessage | undefined
  if (last?.type === 'game_action') {
    // TODO: handle game state updates from engine
  }
}, { deep: true })
</script>

<template>
  <div class="flex-1 flex flex-col p-4 max-w-2xl mx-auto w-full gap-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-primary-light">房间 {{ roomCode }}</h1>
        <p class="text-xs mt-1" :class="connected ? 'text-green-400' : 'text-red-400'">
          {{ connected ? '已连接' : '连接中...' }}
        </p>
      </div>
      <button
        class="px-4 py-2 text-sm bg-surface-light hover:bg-red-500/20 text-gray-400 hover:text-red-400 rounded-lg transition"
        @click="leaveRoom"
      >
        离开房间
      </button>
    </div>

    <!-- 玩家列表 -->
    <div class="bg-surface-light rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-400 mb-3">
        玩家 ({{ players.length }}{{ roomStore.currentRoom?.max_players ? `/${roomStore.currentRoom.max_players}` : '' }})
      </h2>
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
        <div
          v-for="player in players"
          :key="player.id"
          class="flex items-center gap-2 px-3 py-2 bg-surface rounded-lg"
        >
          <div class="w-8 h-8 rounded-full bg-primary/30 flex items-center justify-center text-sm font-bold text-primary-light">
            {{ player.username.charAt(0).toUpperCase() }}
          </div>
          <div class="min-w-0">
            <p class="text-sm truncate">{{ player.username }}</p>
            <p v-if="roomStore.currentRoom?.host_id === player.id" class="text-[10px] text-accent">房主</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 游戏区域占位 -->
    <div class="flex-1 bg-surface-light rounded-xl p-6 flex flex-col items-center justify-center text-gray-500 gap-4">
      <p class="text-lg">游戏区域</p>
      <p class="text-sm">等待游戏逻辑接入...</p>

      <button
        v-if="isHost"
        class="mt-4 px-6 py-3 bg-primary hover:bg-primary-light rounded-lg font-medium transition"
        @click="sendAction('start_game')"
      >
        开始游戏
      </button>
    </div>

    <!-- 消息日志 -->
    <div class="bg-surface-light rounded-xl p-4 max-h-40 overflow-y-auto">
      <p class="text-xs text-gray-500 mb-2">消息日志</p>
      <div v-for="(msg, i) in messages" :key="i" class="text-xs text-gray-400 py-0.5">
        <span class="text-primary-light">[{{ msg.type }}]</span>
        {{ msg.payload ? JSON.stringify(msg.payload) : '' }}
      </div>
      <p v-if="!messages.length" class="text-xs text-gray-600">暂无消息</p>
    </div>
  </div>
</template>
