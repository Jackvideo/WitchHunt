<script setup lang="ts">
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'
import { useWebSocket } from '../composables/useWebSocket'
import type { Card, GameState, GameEvent, PublicPlayer } from '../types'
import { cardNames, identityNames, phaseNames, cardColorClass, identityColorClass, needsTwoTargets } from '../types'

const route = useRoute()
const router = useRouter()
  const auth = useAuthStore()
  const roomStore = useRoomStore()

  const roomCode = route.params.code as string

  if (roomCode) {
    roomStore.fetchCardDescriptions()
  }

  const token = auth.user?.token ?? ''
if (!token) router.push('/login')

const { connected, messages, send } = useWebSocket(roomCode, token)

const gameState = ref<GameState | null>(null)
const errorMsg = ref('')

const selectedCard = ref<Card | null>(null)
const targetStep = ref<'none' | 'first' | 'second'>('none')
const firstTargetId = ref<number | null>(null)

const logEl = ref<HTMLElement | null>(null)
const chatEl = ref<HTMLElement | null>(null)

interface ChatMsg {
  id: number
  userId: number
  username: string
  text: string
  ts: number
  system?: boolean
}
const MAX_CHAT_MSGS = 100
let chatIdCounter = 0
const chatMessages = reactive<ChatMsg[]>([])
const chatInput = ref('')
const chatOpen = ref(false)
const chatUnread = ref(0)
const chatCooldown = ref(false)

function addChatMessage(msg: Omit<ChatMsg, 'id'>) {
  chatMessages.push({ ...msg, id: chatIdCounter++ })
  if (chatMessages.length > MAX_CHAT_MSGS) {
    chatMessages.splice(0, chatMessages.length - MAX_CHAT_MSGS)
  }
  if (chatOpen.value) {
    nextTick(() => chatEl.value?.scrollTo(0, chatEl.value.scrollHeight))
  } else {
    chatUnread.value++
  }
}

function sendChat() {
  const text = chatInput.value.trim()
  if (!text || chatCooldown.value) return
  send({ type: 'chat_message', payload: { text } })
  chatInput.value = ''
  chatCooldown.value = true
  setTimeout(() => { chatCooldown.value = false }, 1000)
}

function toggleChat() {
  chatOpen.value = !chatOpen.value
  if (chatOpen.value) {
    chatUnread.value = 0
    nextTick(() => chatEl.value?.scrollTo(0, chatEl.value.scrollHeight))
  }
}

function chatTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const quickEmojis = ['👍', '👎', '🤔', '😂', '😱', '🔥', '💀', '🛡️', '🎯', '🤫', '👀', '❓']
const showEmojiPicker = ref(false)

interface FloatingEmoji {
  id: number
  userId: number
  emoji: string
}
let emojiIdCounter = 0
const floatingEmojis = reactive<FloatingEmoji[]>([])

function sendEmoji(emoji: string) {
  send({ type: 'send_emoji', payload: { emoji } })
  showEmojiPicker.value = false
}

function spawnFloatingEmoji(userId: number, emoji: string) {
  const id = emojiIdCounter++
  floatingEmojis.push({ id, userId, emoji })
  setTimeout(() => {
    const idx = floatingEmojis.findIndex(e => e.id === id)
    if (idx !== -1) floatingEmojis.splice(idx, 1)
  }, 2200)
}

const playerEffects = reactive<Record<number, string>>({})
const effectTimeouts = new Map<number, ReturnType<typeof setTimeout>>()
const prevEventCount = ref(0)

function setPlayerEffect(playerId: number, type: string, durationMs: number) {
  const existing = effectTimeouts.get(playerId)
  if (existing) clearTimeout(existing)
  playerEffects[playerId] = type
  const timeout = setTimeout(() => {
    delete playerEffects[playerId]
    effectTimeouts.delete(playerId)
  }, durationMs)
  effectTimeouts.set(playerId, timeout)
}

function processEventEffect(evt: GameEvent) {
  if (evt.type === 'kill' && evt.data?.player_id) {
    setPlayerEffect(Number(evt.data.player_id), 'kill', 3000)
  }
  if (evt.type === 'sheriff_protect' && evt.data?.sheriff_id) {
    setPlayerEffect(Number(evt.data.sheriff_id), 'sheriff', 3000)
  }
  if (evt.type === 'contagion_receive' && evt.data?.player_ids) {
    const ids = evt.data.player_ids as number[]
    for (const id of ids) {
      setPlayerEffect(id, 'contagion', 12000)
    }
  }
}

watch(messages, (msgs) => {
  const last = msgs[msgs.length - 1]
  if (!last) return
  if (last.type === 'state_update' && last.payload) {
    const newState = last.payload as unknown as GameState
    const events = newState.events || []
    if (events.length > prevEventCount.value) {
      const newEvents = events.slice(prevEventCount.value)
      for (const evt of newEvents) {
        processEventEffect(evt)
      }
    }
    prevEventCount.value = events.length
    gameState.value = newState
    nextTick(() => logEl.value?.scrollTo(0, logEl.value.scrollHeight))
  }
  if (last.type === 'error' && last.payload) {
    errorMsg.value = (last.payload as { message: string }).message
    setTimeout(() => (errorMsg.value = ''), 3000)
  }
  if (last.type === 'player_emoji' && last.payload) {
    const { user_id, emoji } = last.payload as { user_id: number; emoji: string }
    spawnFloatingEmoji(user_id, emoji)
  }
  if (last.type === 'chat_message' && last.payload) {
    const p = last.payload as { user_id: number; username: string; text: string; ts: number }
    addChatMessage({ userId: p.user_id, username: p.username, text: p.text, ts: p.ts })
  }
  if (last.type === 'player_join' && last.payload) {
    const name = String(last.payload.username)
    addChatMessage({ userId: 0, username: '', text: `${name} 加入了房间`, ts: Date.now(), system: true })
    if (roomStore.currentRoom) {
      const pid = Number(last.payload.user_id || last.payload.id)
      if (!roomStore.currentRoom.players.find(p => p.id === pid)) {
        roomStore.currentRoom.players.push({
          id: pid,
          username: name,
          ready: true
        })
      }
    }
  }
  if (last.type === 'player_leave' && last.payload) {
    const name = String(last.payload.username)
    addChatMessage({ userId: 0, username: '', text: `${name} 离开了房间`, ts: Date.now(), system: true })
    if (roomStore.currentRoom) {
      const pid = Number(last.payload.user_id || last.payload.id)
      roomStore.currentRoom.players = roomStore.currentRoom.players.filter(p => p.id !== pid)
    }
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

function addBot() {
  send({ type: 'add_bot' })
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
          第{{ gs.day_number }}天 · {{ phaseNames[gs.phase] || gs.phase }}
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
          !p.alive && playerEffects[p.user_id] !== 'kill' ? 'opacity-40 border-gray-700 bg-surface-light/50' :
          playerEffects[p.user_id] === 'kill' ? 'border-red-500 bg-red-500/15 shadow-[0_0_12px_rgba(239,68,68,0.4)] animate-shake' :
          playerEffects[p.user_id] === 'sheriff' ? 'border-yellow-400 bg-yellow-400/10 shadow-[0_0_12px_rgba(250,204,21,0.3)]' :
          playerEffects[p.user_id] === 'contagion' ? 'border-red-400 bg-red-500/10 ring-1 ring-red-400/40' :
          selectedCard && targetStep !== 'none' && isTarget(p.user_id) ? 'border-accent bg-amber-500/10 hover:bg-amber-500/20' :
          (gs?.phase === 'night_witch' || gs?.phase === 'night_sheriff') && isTarget(p.user_id) ? 'border-accent bg-amber-500/10 hover:bg-amber-500/20' :
          'border-gray-700 bg-surface-light hover:border-gray-500',
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
          <span v-if="playerEffects[p.user_id] === 'contagion'" class="text-[10px] px-1.5 py-0.5 rounded bg-red-500/30 text-red-300 animate-pulse">
            传染
          </span>
          <span v-if="p.unrevealed_count > 0 && p.user_id !== gs?.your_id"
            class="text-[10px] px-1.5 py-0.5 rounded transition-all duration-300"
            :class="playerEffects[p.user_id] === 'contagion'
              ? 'bg-red-500/30 text-red-300 border border-red-400/60 contagion-card-glow'
              : 'bg-gray-600/30 text-gray-400'">
            身份 {{ p.unrevealed_count }}张
          </span>
          <span v-for="(ri, idx) in (p.revealed_identities || [])" :key="'rev-'+idx"
            class="text-[10px] px-1.5 py-0.5 rounded border"
            :class="identityColorClass[ri] || 'bg-gray-600/30 text-gray-400'">
            {{ identityNames[ri] || ri }}
          </span>
          <span v-for="(ui, idx) in (p.user_id === gs?.your_id ? gs?.identities?.filter(i => !i.revealed) : [])" :key="'unrev-'+idx"
            class="text-[10px] px-1.5 py-0.5 rounded border transition-all duration-300"
            :class="playerEffects[p.user_id] === 'contagion'
              ? 'border-red-400/60 bg-red-500/25 text-red-200 contagion-card-glow'
              : 'border-gray-600 bg-gray-600/30 text-gray-300'">
            {{ identityNames[ui.type] || ui.type }}
          </span>
        </div>
        <div v-if="!p.alive" class="absolute inset-0 flex items-center justify-center">
          <span class="text-xl text-gray-600 font-bold">死亡</span>
        </div>
        <!-- Floating emojis -->
        <TransitionGroup name="emoji-float">
          <span
            v-for="fe in floatingEmojis.filter(e => e.userId === p.user_id)"
            :key="fe.id"
            class="emoji-bubble absolute -right-2 top-0 text-2xl pointer-events-none select-none"
          >{{ fe.emoji }}</span>
        </TransitionGroup>
      </div>
    </div>

    <!-- Target hint -->
    <div v-if="selectedCard" class="text-center space-y-1">
      <p class="text-xs text-gray-300 font-medium bg-gray-700/50 py-1 px-3 rounded-lg inline-block">
        {{ roomStore.cardDescriptions[selectedCard.type] || '暂无说明' }}
      </p>
      <p v-if="targetStep === 'first'" class="text-xs text-accent">
        选择目标玩家使用 [{{ cardNames[selectedCard.type] || selectedCard.type }}]
        <button class="ml-2 text-gray-400 hover:text-gray-200" @click="clearSelection">取消</button>
      </p>
      <p v-if="targetStep === 'second'" class="text-xs text-accent">
        选择第二个目标（接收方）
        <button class="ml-2 text-gray-400 hover:text-gray-200" @click="clearSelection">取消</button>
      </p>
    </div>

    <!-- Your identities -->
    <div v-if="gs && gs.identities && false" class="bg-surface-light rounded-xl p-3">
      <p class="text-[10px] text-gray-500 mb-1.5">你的身份牌</p>
      <div class="flex gap-1.5 flex-wrap">
        <div v-for="(id, i) in gs?.identities" :key="i"
          class="px-3 py-1.5 rounded-lg border text-xs font-medium"
          :class="id.revealed
            ? (identityColorClass[id.type] || 'bg-gray-600/30 text-gray-400 border-gray-600')
            : 'bg-gray-600/30 text-gray-400 border-gray-600'">
          {{ id.revealed ? (identityNames[id.type] || id.type) : '?' }}
        </div>
      </div>
      <p v-if="gs?.is_witch" class="text-[10px] text-purple-400 mt-1">你是女巫阵营</p>
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
      <div v-if="!gs && isHost" class="flex flex-col items-center gap-2">
        <div class="flex gap-2">
          <button
            class="px-5 py-2.5 bg-gray-600 hover:bg-gray-500 rounded-lg font-medium transition"
            @click="addBot">
            添加机器人
          </button>
          <button v-if="players.length >= 4"
            class="px-5 py-2.5 bg-primary hover:bg-primary-light rounded-lg font-medium transition"
            @click="startGame">
            开始游戏 ({{ players.length }}人)
          </button>
        </div>
        <p v-if="players.length < 4" class="text-xs text-gray-500">至少需要4人才能开始 (当前{{ players.length }}人)</p>
      </div>
      <p v-if="!gs && !isHost" class="text-xs text-gray-500">等待房主开始游戏...</p>

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

    <!-- Chat Panel -->
    <div class="bg-surface-light rounded-xl overflow-hidden border border-gray-700/50">
      <!-- Chat Header (toggle) -->
      <button
        class="w-full flex items-center justify-between px-4 py-2.5 hover:bg-gray-700/30 transition"
        @click="toggleChat"
      >
        <div class="flex items-center gap-2">
          <span class="text-xs font-medium text-gray-300">💬 聊天</span>
          <span v-if="chatUnread > 0 && !chatOpen"
            class="min-w-[18px] h-[18px] flex items-center justify-center text-[10px] font-bold bg-red-500 text-white rounded-full px-1 animate-bounce">
            {{ chatUnread > 99 ? '99+' : chatUnread }}
          </span>
        </div>
        <svg class="w-4 h-4 text-gray-500 transition-transform" :class="chatOpen ? 'rotate-180' : ''" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
        </svg>
      </button>

      <!-- Chat Body -->
      <Transition name="chat-slide">
        <div v-if="chatOpen" class="border-t border-gray-700/50">
          <!-- Messages -->
          <div ref="chatEl" class="h-52 overflow-y-auto px-3 py-2 space-y-1.5 chat-scroll">
            <p v-if="chatMessages.length === 0" class="text-xs text-gray-600 text-center py-8">暂无消息，说点什么吧</p>
            <div v-for="msg in chatMessages" :key="msg.id">
              <!-- System message -->
              <p v-if="msg.system" class="text-[10px] text-gray-500 text-center py-0.5">{{ msg.text }}</p>
              <!-- Player message -->
              <div v-else class="flex gap-2" :class="msg.userId === auth.user?.id ? 'flex-row-reverse' : ''">
                <div
                  class="w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-bold shrink-0 mt-0.5"
                  :class="msg.userId === auth.user?.id ? 'bg-primary/30 text-primary-light' : 'bg-gray-600 text-gray-300'"
                >{{ msg.username.charAt(0).toUpperCase() }}</div>
                <div class="max-w-[75%] min-w-0">
                  <div class="flex items-baseline gap-1.5 mb-0.5" :class="msg.userId === auth.user?.id ? 'flex-row-reverse' : ''">
                    <span class="text-[10px] font-medium text-gray-400">{{ msg.username }}</span>
                    <span class="text-[9px] text-gray-600">{{ chatTime(msg.ts) }}</span>
                  </div>
                  <div
                    class="px-2.5 py-1.5 rounded-xl text-xs leading-relaxed break-words"
                    :class="msg.userId === auth.user?.id
                      ? 'bg-primary/20 text-gray-200 rounded-tr-sm'
                      : 'bg-gray-700/60 text-gray-300 rounded-tl-sm'"
                  >{{ msg.text }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Input -->
          <div class="flex items-center gap-2 px-3 py-2 border-t border-gray-700/50">
            <div class="relative">
              <button
                class="w-8 h-8 flex items-center justify-center text-base rounded-lg hover:bg-gray-600/60 transition"
                @click="showEmojiPicker = !showEmojiPicker"
              >😊</button>
              <Transition name="fade">
                <div v-if="showEmojiPicker" class="absolute bottom-full left-0 mb-2 bg-surface border border-gray-700 rounded-xl p-2 shadow-xl z-40">
                  <div class="grid grid-cols-6 gap-1">
                    <button
                      v-for="em in quickEmojis" :key="em"
                      class="w-8 h-8 flex items-center justify-center text-lg rounded-lg hover:bg-gray-600/60 active:scale-90 transition-all"
                      @click="sendEmoji(em)"
                    >{{ em }}</button>
                  </div>
                </div>
              </Transition>
            </div>
            <input
              v-model="chatInput"
              type="text"
              maxlength="200"
              placeholder="发送消息..."
              class="flex-1 min-w-0 px-3 py-1.5 bg-surface rounded-lg border border-gray-600 text-xs
                     focus:border-primary-light focus:outline-none transition placeholder-gray-600"
              @keydown.enter.prevent="sendChat"
            />
            <button
              :disabled="!chatInput.trim() || chatCooldown"
              class="px-3 py-1.5 bg-primary hover:bg-primary-light rounded-lg text-xs font-medium transition
                     disabled:opacity-30 disabled:cursor-not-allowed shrink-0"
              @click="sendChat"
            >发送</button>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Event log -->
    <div ref="logEl" class="bg-surface-light rounded-xl p-3 max-h-48 overflow-y-auto">
      <p class="text-[10px] text-gray-500 mb-1">事件日志</p>
      <div v-for="(evt, i) in (gs?.events || [])" :key="i"
        class="text-xs py-0.5 border-b border-gray-700/30 last:border-0 transition-all duration-300"
        :class="[
          evt.type === 'contagion' ? 'text-red-400 font-bold text-sm py-1.5 bg-red-500/10 -mx-1 px-1 rounded' :
          evt.type === 'contagion_receive' ? 'text-red-400 font-semibold' :
          evt.type === 'kill' ? 'text-red-300 font-semibold' :
          evt.type === 'sheriff_protect' ? 'text-yellow-300 font-semibold' :
          'text-gray-400',
        ]">
        <span v-if="evt.type === 'contagion'" class="mr-1">⚠</span>
        <span v-if="evt.type === 'kill'" class="mr-1">💀</span>
        <span v-if="evt.type === 'sheriff_protect'" class="mr-1">🛡</span>
        {{ evt.message }}
      </div>
      <p v-if="!gs?.events?.length && !gs" class="text-xs text-gray-600">等待游戏开始...</p>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.3s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  10%, 30%, 50%, 70%, 90% { transform: translateX(-3px); }
  20%, 40%, 60%, 80% { transform: translateX(3px); }
}
.animate-shake {
  animation: shake 0.4s ease-in-out 6;
}

@keyframes emoji-rise {
  0% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
  50% {
    opacity: 1;
    transform: translateY(-28px) scale(1.3);
  }
  100% {
    opacity: 0;
    transform: translateY(-52px) scale(0.8);
  }
}
.emoji-bubble {
  animation: emoji-rise 2.2s ease-out forwards;
  filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.3));
}
.emoji-float-enter-from {
  opacity: 0;
  transform: translateY(8px) scale(0.5);
}
.emoji-float-leave-to {
  opacity: 0;
}

@keyframes contagion-glow {
  0%, 100% {
    box-shadow: 0 0 4px rgba(239, 68, 68, 0.3);
  }
  50% {
    box-shadow: 0 0 10px rgba(239, 68, 68, 0.6);
  }
}
.contagion-card-glow {
  animation: contagion-glow 1s ease-in-out infinite;
}

.chat-slide-enter-active { transition: all 0.25s ease-out; }
.chat-slide-leave-active { transition: all 0.2s ease-in; }
.chat-slide-enter-from,
.chat-slide-leave-to {
  max-height: 0;
  opacity: 0;
  overflow: hidden;
}
.chat-slide-enter-to,
.chat-slide-leave-from {
  max-height: 400px;
  opacity: 1;
}

.chat-scroll::-webkit-scrollbar { width: 4px; }
.chat-scroll::-webkit-scrollbar-track { background: transparent; }
.chat-scroll::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 2px; }
.chat-scroll::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
</style>
