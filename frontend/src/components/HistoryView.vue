<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../lib/api'
import { t, locale } from '../i18n'

const emit = defineEmits(['open'])

const sessions = ref([])
const loading = ref(true)
const error = ref('')

function statusLabel(status) {
  return t('status.' + status) === 'status.' + status ? status : t('status.' + status)
}

function badgeStyle(status) {
  if (status === 'sent') return 'background:var(--pastel-sage);color:var(--sent)'
  if (status === 'failed') return 'background:color-mix(in srgb,var(--danger) 12%,transparent);color:var(--danger)'
  if (status === 'cancelled') return 'background:var(--pastel-butter);color:var(--ink-2)'
  return 'background:var(--signal-soft);color:var(--signal)'
}

function fmtDate(iso) {
  try {
    return new Date(iso).toLocaleString(locale.value === 'en' ? 'en-US' : 'tr-TR', {
      day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
    })
  } catch {
    return iso
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    sessions.value = await api.listSessions(100)
  } catch (e) {
    error.value = e.message || t('hist.loadError')
  } finally {
    loading.value = false
  }
}

onMounted(load)
defineExpose({ reload: load })
</script>

<template>
  <div class="space-y-5 rise">
    <div class="flex items-end justify-between">
      <div>
        <div class="label mb-1">{{ t('app.archive') }}</div>
        <h2 class="serif-title text-[1.7rem]">{{ t('hist.pastMinutes') }}</h2>
      </div>
      <button @click="load" class="btn btn-ghost px-3 py-1.5 text-[13px]">{{ t('hist.refresh') }}</button>
    </div>

    <hr class="rule" />

    <p v-if="loading" class="text-sm py-6" style="color: var(--ink-3)">{{ t('hist.loading') }}</p>
    <p v-else-if="error" class="text-sm py-6" style="color: var(--danger)">{{ error }}</p>
    <p v-else-if="sessions.length === 0" class="text-sm py-6" style="color: var(--ink-3)">
      {{ t('hist.empty') }}
    </p>

    <ul v-else class="divide-y" style="border-color: var(--line)">
      <li v-for="(s, i) in sessions" :key="s.id" style="border-color: var(--line)" :class="{ 'border-t': i === 0 }">
        <button
          @click="emit('open', s.id)"
          class="w-full text-left py-3.5 flex items-center justify-between gap-4 group transition-colors"
          style="border-radius: 8px"
          onmouseover="this.style.background='var(--paper-2)'"
          onmouseout="this.style.background='transparent'"
        >
          <div class="min-w-0 pl-1">
            <div class="text-[15px] font-medium truncate group-hover:underline" style="text-underline-offset: 3px">{{ s.title }}</div>
            <div class="label mt-1">{{ fmtDate(s.created_at) }} · {{ t('hist.recipientCount', { n: s.recipient_count }) }}</div>
          </div>
          <span class="label shrink-0 px-2 py-1 rounded-full mr-1" :style="badgeStyle(s.status)">
            {{ statusLabel(s.status) }}
          </span>
        </button>
      </li>
    </ul>
  </div>
</template>
