<script setup>
import { computed } from 'vue'
import { useSessionStore, Phase } from '../stores/session'

const store = useSessionStore()

// Boru hattı adımları — mevcut duruma göre işaretlenir.
const steps = [
  { key: 'processing', label: 'Ses birleştiriliyor' },
  { key: 'transcribing', label: 'Metne dökülüyor (ASR)' },
  { key: 'summarizing', label: 'Özet çıkarılıyor (LLM)' },
  { key: 'pending_send', label: 'Gönderim bekliyor' },
  { key: 'sending', label: 'E-posta gönderiliyor' },
  { key: 'sent', label: 'Gönderildi' },
]

const order = ['processing', 'transcribing', 'summarizing', 'pending_send', 'sending', 'sent']
const currentIdx = computed(() => order.indexOf(store.status))

function stateFor(key) {
  const idx = order.indexOf(key)
  if (idx < currentIdx.value) return 'done'
  if (idx === currentIdx.value) return 'active'
  return 'todo'
}

const isFailed = computed(() => store.phase === Phase.Failed)
</script>

<template>
  <div class="pt-4">
    <h2 class="text-lg font-semibold mb-1">İşleniyor</h2>
    <p class="text-sm text-slate-400 mb-6">{{ store.title }}</p>

    <div v-if="isFailed" class="rounded-lg border border-red-500/40 bg-red-500/10 p-4">
      <div class="font-medium text-red-300">Bir hata oluştu</div>
      <p class="text-sm text-red-200/80 mt-1">{{ store.error }}</p>
    </div>

    <ol v-else class="space-y-3">
      <li v-for="s in steps" :key="s.key" class="flex items-center gap-3">
        <span
          class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs"
          :class="{
            'bg-emerald-500 text-slate-900': stateFor(s.key) === 'done',
            'bg-indigo-500 text-white': stateFor(s.key) === 'active',
            'bg-slate-800 text-slate-500 border border-slate-700': stateFor(s.key) === 'todo',
          }"
        >
          <span v-if="stateFor(s.key) === 'done'">✓</span>
          <span v-else-if="stateFor(s.key) === 'active'" class="h-2.5 w-2.5 rounded-full bg-white animate-pulse"></span>
          <span v-else>○</span>
        </span>
        <span
          class="text-sm"
          :class="{
            'text-slate-200': stateFor(s.key) !== 'todo',
            'text-slate-500': stateFor(s.key) === 'todo',
            'font-medium': stateFor(s.key) === 'active',
          }"
          >{{ s.label }}</span
        >
      </li>
    </ol>

    <p v-if="!isFailed && store.progressMessage" class="mt-6 text-sm text-slate-400">
      {{ store.progressMessage }}
    </p>
  </div>
</template>
