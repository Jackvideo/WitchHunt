<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'

const auth = useAuthStore()
const roomStore = useRoomStore()
const router = useRouter()

const joinCode = ref('')
const maxPlayers = ref(8)
const loading = ref(false)

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
    </div>
  </div>
</template>
