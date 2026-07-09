<script setup>
import { ref } from 'vue'
import { useSessionStore, Phase } from './stores/session'
import SetupForm from './components/SetupForm.vue'
import RecordingView from './components/RecordingView.vue'
import ProgressView from './components/ProgressView.vue'
import SummaryView from './components/SummaryView.vue'
import HistoryView from './components/HistoryView.vue'
import { t, locale, toggleLocale } from './i18n'

const store = useSessionStore()

// 'flow' = oturum akışı (kur/kayıt/işle/özet), 'history' = geçmiş listesi.
const view = ref('flow')

// Tema (açık varsayılan, karanlık opsiyonel).
const theme = ref(document.documentElement.getAttribute('data-theme') || 'light')
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', theme.value)
  try {
    localStorage.setItem('m2m-theme', theme.value)
  } catch (e) {
    /* yoksay */
  }
}

const busyPhases = [Phase.Recording, Phase.Processing]

function goNew() {
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
    <!-- Masthead -->
    <header class="px-5 pt-7 pb-0">
      <div class="max-w-2xl mx-auto">
        <div class="flex items-center justify-between gap-4">
          <h1 class="serif-title text-[2rem] sm:text-[2.3rem] leading-none" style="color: var(--signal)">
            meeting<span style="color: var(--ink-3)">·</span>to<span style="color: var(--ink-3)">·</span>mail
          </h1>
          <nav class="flex items-center gap-1 shrink-0">
            <button
              @click="goNew"
              class="btn px-3 py-1.5 text-[13px] font-semibold"
              :class="view === 'flow' ? 'btn-ghost' : ''"
              :style="view === 'flow' ? 'color:var(--ink);border-color:var(--line-2)' : 'background:transparent;color:var(--ink-3)'"
            >
              {{ t('app.new') }}
            </button>
            <button
              @click="view = 'history'"
              class="btn px-3 py-1.5 text-[13px] font-semibold"
              :style="view === 'history' ? 'color:var(--ink);border-color:var(--line-2)' : 'background:transparent;color:var(--ink-3)'"
            >
              {{ t('app.archive') }}
            </button>
            <button
              class="theme-toggle ml-1 font-semibold text-[12px] tracking-wide"
              :title="t('app.langTitle')"
              @click="toggleLocale"
            >
              {{ locale === 'tr' ? 'EN' : 'TR' }}
            </button>
            <button
              class="theme-toggle"
              :title="theme === 'dark' ? t('app.themeLight') : t('app.themeDark')"
              @click="toggleTheme"
            >
              <span v-if="theme === 'dark'">☾</span><span v-else>☀</span>
            </button>
          </nav>
        </div>
        <div class="flex items-center gap-2 mt-3 mb-3">
          <span class="inline-block w-1.5 h-1.5 rounded-full" style="background: var(--sent)"></span>
          <span class="label">{{ t('app.localOffline') }}</span>
          <hr class="rule flex-1" />
        </div>
      </div>
    </header>

    <main class="flex-1 w-full max-w-2xl mx-auto px-5 py-8">
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

    <footer class="px-5 py-5">
      <div class="max-w-2xl mx-auto">
        <hr class="rule mb-3" />
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 label">
          <span>{{ t('app.footerLeft') }}</span>
          <span>{{ t('app.footerRight') }}</span>
        </div>
      </div>
    </footer>
  </div>
</template>
