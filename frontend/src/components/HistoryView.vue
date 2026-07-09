<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../lib/api'

const emit = defineEmits(['open'])

const sessions = ref([])
const loading = ref(true)
const error = ref('')

const statusLabels = {
  configuring: 'Kuruluyor',
  recording: 'Kaydediliyor',
  processing: 'İşleniyor',
  transcribing: 'Metne dökülüyor',
  summarizing: 'Özetleniyor',
  pending_send: 'Gönderim bekliyor',
  sending: 'Gönderiliyor',
  sent: 'Gönderildi',
  cancelled: 'İptal edildi',
  failed: 'Hata',
}

function badgeClass(status) {
  if (status === 'sent') return 'bg-emerald-500/20 text-emerald-300'
  if (status === 'failed') return 'bg-red-500/20 text-red-300'
  if (status === 'cancelled') return 'bg-amber-500/20 text-amber-300'
  return 'bg-indigo-500/20 text-indigo-300'
}

function fmtDate(iso) {
  try {
    return new Date(iso).toLocaleString('tr-TR', {
      day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit',
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
    error.value = e.message || 'Geçmiş yüklenemedi.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
defineExpose({ reload: load })
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold">Geçmiş Toplantılar</h2>
      <button @click="load" class="text-sm text-slate-400 hover:text-slate-200">↻ Yenile</button>
    </div>

    <p v-if="loading" class="text-sm text-slate-400">Yükleniyor…</p>
    <p v-else-if="error" class="text-sm text-red-400">{{ error }}</p>
    <p v-else-if="sessions.length === 0" class="text-sm text-slate-400">
      Henüz kayıt yok. İlk toplantını oluştur.
    </p>

    <ul v-else class="space-y-2">
      <li v-for="s in sessions" :key="s.id">
        <button
          @click="emit('open', s.id)"
          class="w-full text-left rounded-lg bg-slate-800 border border-slate-700 hover:border-indigo-500/60 px-3 py-3 transition"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="font-medium truncate">{{ s.title }}</div>
              <div class="text-xs text-slate-400 mt-0.5">
                {{ fmtDate(s.created_at) }} · {{ s.recipient_count }} alıcı
              </div>
            </div>
            <span class="shrink-0 text-xs px-2 py-0.5 rounded-full" :class="badgeClass(s.status)">
              {{ statusLabels[s.status] || s.status }}
            </span>
          </div>
        </button>
      </li>
    </ul>
  </div>
</template>
