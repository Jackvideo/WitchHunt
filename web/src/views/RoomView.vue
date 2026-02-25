<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'
import { useWebSocket } from '../composables/useWebSocket'
import type { Card, GameState, PublicPlayer } from '../types'
import { cardNames, identityNames, phaseNames, cardColorClass, identityColorClass, needsTwoTargets } from '../types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const roomStore = useRoomStore()

const roomCode = route.params.code as string
const token = auth.user?.token ?? ''
if (!token) router.push('/login')

const { connected, messages, send } = useWebSocket(roomCode, token)

const gameState = ref<GameState | null>(null)
const errorMsg = ref('')

const selectedCard = ref<Card | null>(null)
const targetStep = ref<'none' | 'first' | 'second'>('none')
const firstTargetId = ref<number | null>(null)

const logEl = ref<HTMLElement | null>(null)

watch(messages, (msgs) => {
  const last = msgs[msgs.length - 1]
  if (!last) return
  if (last.type === 'state_update' && last.payload) {
    gameState.value = last.payload as unknown as GameState
    nextTick(() => logEl.value?.scrollTo(0, logEl.value.scrollHeight))
  }
  if (last.type === 'error' && last.payload) {
    errorMsg.value = (last.payload as { message: string }).message
    setTimeout(() => (errorMsg.value = ''), 3000)
  }
}, { deep: true })

const gs = computed(() => gameState.value)
const actions = computed(() => gs.value?.actions ?? [])
const isTarget = (id: number) => gs.value?.valid_targets?.includes(id) ?? false

const players = computed(() => {
  if (gs.value) return gs.value.players
  return roomStore.currentRoom?.players?.map(p => ({
    user_id: p.id, username: p.username, alive: true,
    identity_count: 0, unrevealed_count: 0, revealed_identities: null,
    equipment: null, accuse_total: 0, detained: 0, has_hammer: false,
  } as PublicPlayer)) ?? []
})
const isHost = computed(() => roomStore.currentRoom?.host_id === auth.user?.id)

function sendAction(type_: string, data: Record<string, unknown> = {}) {
  send({ type: 'game_action', payload: { type: type_, ...data } })
}

function startGame() {
  send({ type: 'start_game' })
}

function leaveRoom() {
  roomStore.clear()
  router.push('/lobby')
}

function selectCard(card: Card) {
  clearSelection()
  selectedCard.value = card
  targetStep.value = 'first'
}

function clickPlayer(p: PublicPlayer) {
  if (!selectedCard.value || targetStep.value === 'none') {
    if (gs.value?.phase === 'night_witch' && actions.value.includes('witch_kill') && isTarget(p.user_id)) {
      sendAction('witch_kill', { target_id: p.user_id })
    }
    if (gs.value?.phase === 'night_sheriff' && actions.value.includes('sheriff_protect') && isTarget(p.user_id)) {
      sendAction('sheriff_protect', { target_id: p.user_id })
    }
    return
  }

  if (targetStep.value === 'first') {
    if (p.user_id === gs.value?.your_id || !p.alive) return
    if (needsTwoTargets(selectedCard.value)) {
      firstTargetId.value = p.user_id
      targetStep.value = 'second'
    } else {
      sendAction('play_card', { card_id: selectedCard.value.id, target_id: p.user_id })
      clearSelection()
    }
  } else if (targetStep.value === 'second') {
    if (p.user_id === gs.value?.your_id || !p.alive || p.user_id === firstTargetId.value) return
    sendAction('play_card', {
      card_id: selectedCard.value.id,
      target_id: firstTargetId.value,
      extra_target_id: p.user_id,
    })
    clearSelection()
  }
}

function clearSelection() {
  selectedCard.value = null
  targetStep.value = 'none'
  firstTargetId.value = null
}

function flipIdentity(index: number) {
  sendAction('flip_identity', { card_index: index })
}
</script>

<template>
  <div class="flex-1 flex flex-col max-w-3xl mx-auto w-full p-3 gap-3 text-sm">

    <!-- Error toast -->
    <Transition name="fade">
      <div v-if="errorMsg" class="fixed top-4 left-1/2 -translate-x-1/2 z-50 bg-red-500/90 text-white px-4 py-2 rounded-lg text-sm">
        {{ errorMsg }}
      </div>
    </Transition>

    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-lg font-bold text-primary-light">
          房间 {{ roomCode }}
          <span class="text-xs ml-2" :class="connected ? 'text-green-400' : 'text-red-400'">
            {{ connected ? '●' : '○' }}
          </span>
        </h1>
        <p v-if="gs" class="text-xs text-gray-400">
          {{ phaseNames[gs.phase] || gs.phase }}
          <span v-if="gs.is_your_turn" class="text-accent ml-1">你的回合</span>
        </p>
      </div>
      <button class="px-3 py-1.5 text-xs bg-surface-light hover:bg-red-500/20 text-gray-400 hover:text-red-400 rounded-lg transition" @click="leaveRoom">
        离开
      </button>
    </div>

    <!-- Game Over -->
    <div v-if="gs?.phase === 'game_over'" class="bg-surface-light rounded-xl p-6 text-center space-y-3">
      <p class="text-2xl font-bold" :class="gs.winner === 'villager' ? 'text-blue-400' : 'text-purple-400'">
        {{ gs.winner === 'villager' ? '村民阵营获胜！' : '女巫阵营获胜！' }}
      </p>
      <div class="flex flex-wrap justify-center gap-2 mt-3">
        <div v-for="p in gs.players" :key="p.user_id"
          class="px-3 py-1 rounded-lg text-xs"
          :class="p.is_witch ? 'bg-purple-500/20 text-purple-300' : 'bg-blue-500/20 text-blue-300'">
          {{ p.username }} - {{ p.is_witch ? '女巫' : '村民' }}{{ p.alive ? '' : '(死亡)' }}
        </div>
      </div>
      <button class="mt-4 px-4 py-2 bg-primary hover:bg-primary-light rounded-lg transition" @click="leaveRoom">返回大厅</button>
    </div>

    <!-- Player Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
      <div
        v-for="p in players" :key="p.user_id"
        class="relative px-3 py-2.5 rounded-xl border transition cursor-pointer"
        :class="[
          !p.alive ? 'opacity-40 border-gray-700 bg-surface-light/50' :
          selectedCard && targetStep !== 'none' && isTarget(p.user_id) ? 'border-accent bg-amber-500/10 hover:bg-amber-500/20' :
          (gs?.phase === 'night_witch' || gs?.phase === 'night_sheriff') && isTarget(p.user_id) ? 'border-accent bg-amber-500/10 hover:bg-amber-500/20' :
          'border-gray-700 bg-surface-light hover:border-gray-500',
          gs?.phase === 'day' && gs.players[gs.players.findIndex(x => x.user_id === gs!.players[0]?.user_id)]?.user_id === p.user_id ? '' : '',
        ]"
        @click="clickPlayer(p)"
      >
        <!-- Name -->
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0"
            :class="p.alive ? 'bg-primary/30 text-primary-light' : 'bg-gray-700 text-gray-500'">
            {{ p.username.charAt(0).toUpperCase() }}
          </div>
          <div class="min-w-0">
            <p class="truncate text-xs font-medium">{{ p.username }}</p>
            <p v-if="roomStore.currentRoom?.host_id === p.user_id && !gs" class="text-[10px] text-accent">房主</p>
            <p v-if="gs?.fellow_witches?.includes(p.user_id)" class="text-[10px] text-purple-400">女巫同伴</p>
          </div>
        </div>
        <!-- Stats -->
        <div v-if="gs" class="mt-1.5 flex flex-wrap gap-1">
          <span v-if="p.accuse_total > 0" class="text-[10px] px-1.5 py-0.5 rounded bg-red-500/20 text-red-300">
            指控 {{ p.accuse_total }}/7
          </span>
          <span v-if="p.has_hammer" class="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300">锤子</span>
          <span v-if="p.detained" class="text-[10px] px-1.5 py-0.5 rounded bg-gray-500/20 text-gray-400">拘留</span>
          <span v-for="eq in (p.equipment || [])" :key="eq.id" class="text-[10px] px-1.5 py-0.5 rounded bg-sky-500/20 text-sky-300">
            {{ cardNames[eq.type] || eq.type }}
          </span>
          <span v-if="p.unrevealed_count > 0" class="text-[10px] px-1.5 py-0.5 rounded bg-gray-600/30 text-gray-400">
            身份 {{ p.unrevealed_count }}张
          </span>
          <span v-for="ri in (p.revealed_identities || [])" :key="ri"
            class="text-[10px] px-1.5 py-0.5 rounded border"
            :class="identityColorClass[ri] || 'bg-gray-600/30 text-gray-400'">
            {{ identityNames[ri] || ri }}
          </span>
        </div>
        <div v-if="!p.alive" class="absolute inset-0 flex items-center justify-center">
          <span class="text-xl text-gray-600 font-bold">死亡</span>
        </div>
      </div>
    </div>

    <!-- Target hint -->
    <p v-if="selectedCard && targetStep === 'first'" class="text-xs text-accent text-center">
      选择目标玩家使用 [{{ cardNames[selectedCard.type] || selectedCard.type }}]
      <button class="ml-2 text-gray-400 hover:text-gray-200" @click="clearSelection">取消</button>
    </p>
    <p v-if="selectedCard && targetStep === 'second'" class="text-xs text-accent text-center">
      选择第二个目标（接收方）
      <button class="ml-2 text-gray-400 hover:text-gray-200" @click="clearSelection">取消</button>
    </p>

    <!-- Your identities -->
    <div v-if="gs && gs.identities" class="bg-surface-light rounded-xl p-3">
      <p class="text-[10px] text-gray-500 mb-1.5">你的身份牌</p>
      <div class="flex gap-1.5 flex-wrap">
        <div v-for="(id, i) in gs.identities" :key="i"
          class="px-3 py-1.5 rounded-lg border text-xs font-medium"
          :class="id.revealed
            ? (identityColorClass[id.type] || 'bg-gray-600/30 text-gray-400 border-gray-600')
            : 'bg-gray-600/30 text-gray-400 border-gray-600'">
          {{ id.revealed ? (identityNames[id.type] || id.type) : '?' }}
        </div>
      </div>
      <p v-if="gs.is_witch" class="text-[10px] text-purple-400 mt-1">你是女巫阵营</p>
    </div>

    <!-- Your hand -->
    <div v-if="gs && gs.hand && gs.hand.length > 0" class="bg-surface-light rounded-xl p-3">
      <p class="text-[10px] text-gray-500 mb-1.5">手牌 ({{ gs.hand.length }})</p>
      <div class="flex gap-1.5 flex-wrap">
        <button v-for="card in gs.hand" :key="card.id"
          class="px-3 py-1.5 rounded-lg border text-xs font-medium transition hover:scale-105"
          :class="[
            cardColorClass[card.color] || 'bg-gray-600/30 text-gray-400 border-gray-600',
            selectedCard?.id === card.id ? 'ring-2 ring-accent' : ''
          ]"
          :disabled="!actions.includes('play_card')"
          @click="actions.includes('play_card') && selectCard(card)">
          {{ cardNames[card.type] || card.type }}
        </button>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex flex-wrap gap-2 justify-center">
      <!-- Lobby: start game -->
      <button v-if="!gs && isHost && players.length >= 4"
        class="px-5 py-2.5 bg-primary hover:bg-primary-light rounded-lg font-medium transition"
        @click="startGame">
        开始游戏 ({{ players.length }}人)
      </button>
      <p v-if="!gs && !isHost" class="text-xs text-gray-500">等待房主开始游戏...</p>
      <p v-if="!gs && isHost && players.length < 4" class="text-xs text-gray-500">至少需要4人才能开始 (当前{{ players.length }}人)</p>

      <!-- Day actions -->
      <button v-if="actions.includes('draw')"
        class="px-4 py-2 bg-primary hover:bg-primary-light rounded-lg text-xs font-medium transition"
        @click="sendAction('draw')">
        抽牌 (剩{{ gs?.draw_pile_count }}张)
      </button>
      <button v-if="actions.includes('end_turn')"
        class="px-4 py-2 bg-surface-light hover:bg-gray-600 border border-gray-600 rounded-lg text-xs font-medium transition"
        @click="sendAction('end_turn')">
        结束回合
      </button>

      <!-- Night result -->
      <button v-if="actions.includes('confess')"
        class="px-4 py-2 bg-red-500/80 hover:bg-red-500 rounded-lg text-xs font-medium transition"
        @click="sendAction('confess')">
        自首（翻开一张身份牌）
      </button>
      <button v-if="actions.includes('pass')"
        class="px-4 py-2 bg-surface-light hover:bg-gray-600 border border-gray-600 rounded-lg text-xs font-medium transition"
        @click="sendAction('pass')">
        跳过
      </button>

      <!-- Trial -->
      <template v-if="actions.includes('flip_identity') && gs?.trial">
        <p class="w-full text-xs text-center text-accent">选择翻开 {{ gs.players.find(p => p.user_id === gs!.trial!.accused_id)?.username }} 的哪张身份牌：</p>
        <button v-for="i in (gs.players.find(p => p.user_id === gs!.trial!.accused_id)?.unrevealed_count || 0)" :key="i"
          class="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 rounded-lg text-xs font-medium transition"
          @click="flipIdentity(i - 1)">
          第 {{ i }} 张
        </button>
      </template>

      <!-- Waiting hints -->
      <p v-if="gs?.phase === 'night_witch' && !gs.is_witch" class="text-xs text-gray-500">女巫正在商议...</p>
      <p v-if="gs?.phase === 'night_witch' && gs.is_witch && !actions.includes('witch_kill')" class="text-xs text-gray-500">等待其他女巫投票...</p>
      <p v-if="gs?.phase === 'night_sheriff' && !actions.includes('sheriff_protect')" class="text-xs text-gray-500">警长正在行动...</p>
      <p v-if="gs?.phase === 'night_result' && !actions.includes('confess')" class="text-xs text-gray-500">等待其他玩家决定...</p>
      <p v-if="gs?.phase === 'trial' && !actions.includes('flip_identity')" class="text-xs text-gray-500">等待审判者翻牌...</p>
      <p v-if="gs?.phase === 'day' && !gs.is_your_turn" class="text-xs text-gray-500">
        等待 {{ gs.players.find((_, i) => i === gs!.players.findIndex(p => gs!.is_your_turn ? false : p.alive))?.username || '...' }} 行动
      </p>
    </div>

    <!-- Event log -->
    <div ref="logEl" class="bg-surface-light rounded-xl p-3 max-h-48 overflow-y-auto">
      <p class="text-[10px] text-gray-500 mb-1">事件日志</p>
      <div v-for="(evt, i) in (gs?.events || [])" :key="i" class="text-xs text-gray-400 py-0.5 border-b border-gray-700/30 last:border-0">
        {{ evt.message }}
      </div>
      <p v-if="!gs?.events?.length && !gs" class="text-xs text-gray-600">等待游戏开始...</p>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.3s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
