<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'
import type { UserStats, GameRecordItem } from '../types'

const auth = useAuthStore()
const roomStore = useRoomStore()
const router = useRouter()

const joinCode = ref('')
const maxPlayers = ref(8)
const loading = ref(false)

const stats = ref<UserStats | null>(null)
const history = ref<GameRecordItem[]>([])
const showHistory = ref(false)
const historyPage = ref(1)
const historyTotal = ref(0)
const historyLoading = ref(false)

async function fetchStats() {
  try {
    const res = await auth.apiFetch('/api/user/stats')
    stats.value = await res.json()
  } catch { /* ignore */ }
}

async function fetchHistory(page = 1) {
  historyLoading.value = true
  try {
    const res = await auth.apiFetch(`/api/user/history?page=${page}&size=10`)
    const data = await res.json()
    history.value = data.records || []
    historyPage.value = data.page
    historyTotal.value = data.total
  } catch { /* ignore */ }
  historyLoading.value = false
}

function toggleHistory() {
  showHistory.value = !showHistory.value
  if (showHistory.value && history.value.length === 0) {
    fetchHistory()
  }
}

onMounted(() => {
  fetchStats()
})

function winRate(wins: number, total: number): string {
  if (total === 0) return '-'
  return Math.round((wins / total) * 100) + '%'
}

async function handleCreate() {
  loading.value = true
  const room = await roomStore.createRoom(maxPlayers.value)
  loading.value = false
  if (room) router.push(`/room/${room.code}`)
}

async function handleJoin() {
  if (joinCode.value.length !== 6) {
    roomStore.error = '房间码必须是6位'
    return
  }
  loading.value = true
  const room = await roomStore.joinRoom(joinCode.value)
  loading.value = false
  if (room) router.push(`/room/${room.code}`)
}

function logout() {
  auth.logout()
  router.push('/login')
}

function fmtTime(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth()+1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<template>
  <div class="flex-1 flex flex-col items-center justify-center p-4 gap-8">
    <div class="flex items-center justify-between w-full max-w-md">
      <h1 class="text-2xl font-bold text-primary-light">WitchHunt</h1>
      <div class="flex items-center gap-3">
        <span class="text-sm text-gray-400">{{ auth.user?.username }}</span>
        <button class="text-sm text-gray-500 hover:text-red-400 transition" @click="logout">退出</button>
      </div>
    </div>

    <div class="w-full max-w-md space-y-6">
      <!-- 创建房间 -->
      <div class="bg-surface-light rounded-xl p-6 space-y-4">
        <h2 class="text-lg font-semibold">创建房间</h2>
        <div class="flex items-center gap-3">
          <label class="text-sm text-gray-400 shrink-0">最大人数</label>
          <input
            v-model.number="maxPlayers"
            type="number"
            min="2"
            max="20"
            class="flex-1 px-3 py-2 bg-surface rounded-lg border border-gray-600 focus:border-primary-light focus:outline-none transition"
          />
        </div>
        <button
          :disabled="loading"
          class="w-full py-3 bg-primary hover:bg-primary-light rounded-lg font-medium transition disabled:opacity-50"
          @click="handleCreate"
        >
          {{ loading ? '...' : '创建新房间' }}
        </button>
      </div>

      <!-- 加入房间 -->
      <div class="bg-surface-light rounded-xl p-6 space-y-4">
        <h2 class="text-lg font-semibold">加入房间</h2>
        <input
          v-model="joinCode"
          type="text"
          placeholder="输入6位房间码"
          maxlength="6"
          class="w-full px-4 py-3 bg-surface rounded-lg border border-gray-600 focus:border-primary-light focus:outline-none transition uppercase tracking-widest text-center text-lg font-mono"
          @input="joinCode = joinCode.toUpperCase().replace(/[^A-Z0-9]/g, '')"
        />
        <button
          :disabled="loading || joinCode.length !== 6"
          class="w-full py-3 bg-accent hover:bg-amber-400 text-surface rounded-lg font-medium transition disabled:opacity-50"
          @click="handleJoin"
        >
          {{ loading ? '...' : '加入房间' }}
        </button>
      </div>

      <p v-if="roomStore.error" class="text-red-400 text-sm text-center">{{ roomStore.error }}</p>

      <!-- Stats -->
      <div v-if="stats" class="bg-surface-light rounded-xl p-6 space-y-4">
        <h2 class="text-lg font-semibold">个人战绩</h2>
        <div class="grid grid-cols-3 gap-3 text-center">
          <div class="bg-surface rounded-lg py-3">
            <p class="text-2xl font-bold text-primary-light">{{ stats.total_games }}</p>
            <p class="text-xs text-gray-400 mt-1">总场次</p>
          </div>
          <div class="bg-surface rounded-lg py-3">
            <p class="text-2xl font-bold text-green-400">{{ stats.wins }}</p>
            <p class="text-xs text-gray-400 mt-1">胜利</p>
          </div>
          <div class="bg-surface rounded-lg py-3">
            <p class="text-2xl font-bold text-red-400">{{ stats.losses }}</p>
            <p class="text-xs text-gray-400 mt-1">失败</p>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div class="flex items-center justify-between bg-surface rounded-lg px-4 py-2.5">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full bg-blue-400"></span>
              <span class="text-xs text-gray-300">村民</span>
            </div>
            <span class="text-xs font-mono">
              {{ stats.villager_wins }}/{{ stats.villager_games }}
              <span class="text-gray-500 ml-1">{{ winRate(stats.villager_wins, stats.villager_games) }}</span>
            </span>
          </div>
          <div class="flex items-center justify-between bg-surface rounded-lg px-4 py-2.5">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full bg-purple-400"></span>
              <span class="text-xs text-gray-300">女巫</span>
            </div>
            <span class="text-xs font-mono">
              {{ stats.witch_wins }}/{{ stats.witch_games }}
              <span class="text-gray-500 ml-1">{{ winRate(stats.witch_wins, stats.witch_games) }}</span>
            </span>
          </div>
        </div>
        <button
          class="w-full text-xs text-gray-400 hover:text-gray-200 transition py-1"
          @click="toggleHistory"
        >
          {{ showHistory ? '收起对局历史' : '查看对局历史' }}
        </button>
      </div>

      <!-- History -->
      <div v-if="showHistory" class="bg-surface-light rounded-xl p-4 space-y-2">
        <p v-if="historyLoading" class="text-xs text-gray-500 text-center py-4">加载中...</p>
        <p v-else-if="history.length === 0" class="text-xs text-gray-500 text-center py-4">暂无对局记录</p>
        <div v-for="r in history" :key="r.id"
          class="flex items-center justify-between bg-surface rounded-lg px-4 py-3">
          <div class="flex items-center gap-3 min-w-0">
            <span class="text-lg shrink-0">{{ r.won ? '🏆' : '💔' }}</span>
            <div class="min-w-0">
              <p class="text-xs font-medium truncate">
                <span :class="r.your_role === 'witch' ? 'text-purple-300' : 'text-blue-300'">
                  {{ r.your_role === 'witch' ? '女巫' : '村民' }}
                </span>
                <span class="text-gray-500 mx-1">·</span>
                <span :class="r.won ? 'text-green-400' : 'text-red-400'">
                  {{ r.won ? '胜利' : '失败' }}
                </span>
                <span v-if="!r.alive" class="text-gray-600 ml-1">(阵亡)</span>
              </p>
              <p class="text-[10px] text-gray-500 mt-0.5">
                {{ r.player_count }}人 · 第{{ r.day_count }}天结束 ·
                {{ r.winner === 'villager' ? '村民胜' : '女巫胜' }}
              </p>
            </div>
          </div>
          <span class="text-[10px] text-gray-600 shrink-0 ml-2">{{ fmtTime(r.created_at) }}</span>
        </div>
        <div v-if="historyTotal > 10" class="flex items-center justify-center gap-4 pt-2">
          <button
            :disabled="historyPage <= 1"
            class="text-xs text-gray-400 hover:text-gray-200 disabled:opacity-30 transition"
            @click="fetchHistory(historyPage - 1)"
          >上一页</button>
          <span class="text-xs text-gray-500">{{ historyPage }} / {{ Math.ceil(historyTotal / 10) }}</span>
          <button
            :disabled="historyPage >= Math.ceil(historyTotal / 10)"
            class="text-xs text-gray-400 hover:text-gray-200 disabled:opacity-30 transition"
            @click="fetchHistory(historyPage + 1)"
          >下一页</button>
        </div>
      </div>
    </div>
  </div>
</template>
