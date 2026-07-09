<script setup>
import { ref } from 'vue'
import { useSessionStore, Phase } from './stores/session'
import SetupForm from './components/SetupForm.vue'
import RecordingView from './components/RecordingView.vue'
import ProgressView from './components/ProgressView.vue'
import SummaryView from './components/SummaryView.vue'
import HistoryView from './components/HistoryView.vue'

const store = useSessionStore()

// 'flow' = oturum akışı (kur/kayıt/işle/özet), 'history' = geçmiş listesi.
const view = ref('flow')

const busyPhases = [Phase.Recording, Phase.Processing]

function goNew() {
  // Aktif kayıt/işleme varsa sıfırlama; sadece görünümü değiştir.
  if (!busyPhases.includes(store.phase)) store.reset()
  view.value = 'flow'
}

function openFromHistory(id) {
  store.openSession(id)
  view.value = 'flow'
}
</script>

<template>
  <div class="min-h-full flex flex-col">
    <header class="px-4 py-3 border-b border-slate-800/80">
      <div class="max-w-xl mx-auto flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span class="text-xl">🎙️</span>
          <h1 class="text-base font-semibold tracking-tight">Toplantı Kayıt &amp; Özet</h1>
        </div>
        <nav class="flex items-center gap-1 text-sm">
          <button
            @click="goNew"
            :class="[
              'px-3 py-1.5 rounded-lg transition',
              view === 'flow' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-slate-200',
            ]"
          >
            ＋ Yeni
          </button>
          <button
            @click="view = 'history'"
            :class="[
              'px-3 py-1.5 rounded-lg transition',
              view === 'history' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-slate-200',
            ]"
          >
            Geçmiş
          </button>
        </nav>
      </div>
    </header>

    <main class="flex-1 w-full max-w-xl mx-auto px-4 py-6">
      <HistoryView v-if="view === 'history'" @open="openFromHistory" />
      <template v-else>
        <SetupForm v-if="store.phase === Phase.Setup" />
        <RecordingView v-else-if="store.phase === Phase.Recording" />
        <SummaryView
          v-else-if="store.phase === Phase.Done || store.phase === Phase.Cancelled"
        />
        <ProgressView v-else />
      </template>
    </main>

    <footer class="px-4 py-3 text-center text-xs text-slate-500">
      Batch işleme · stub AI · MVP
    </footer>
  </div>
</template>
