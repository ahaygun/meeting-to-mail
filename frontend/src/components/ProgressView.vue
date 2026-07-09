<script setup>
import { computed } from 'vue'
import { useSessionStore, Phase } from '../stores/session'
import { t } from '../i18n'

const store = useSessionStore()

const order = ['processing', 'transcribing', 'summarizing', 'pending_send', 'sending', 'sent']
const steps = order.map((key) => ({ key }))
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
  <div class="rise">
    <div class="label mb-1.5">{{ t('prog.processingLocal') }}</div>
    <h2 class="serif-title text-[1.7rem] mb-7">{{ store.title }}</h2>

    <div v-if="isFailed" class="panel p-5" style="border-color: var(--danger)">
      <div class="font-semibold" style="color: var(--danger)">{{ t('prog.error') }}</div>
      <p class="text-sm mt-1" style="color: var(--ink-2)">{{ store.error }}</p>
    </div>

    <ol v-else class="relative">
      <li v-for="(s, i) in steps" :key="s.key" class="flex gap-4 pb-6 last:pb-0 relative">
        <!-- bağlantı çizgisi -->
        <span
          v-if="i < steps.length - 1"
          class="absolute left-[13px] top-7 bottom-0 w-px"
          :style="`background:${stateFor(s.key) === 'done' ? 'var(--sent)' : 'var(--line-2)'}`"
        ></span>

        <!-- düğüm -->
        <span
          class="relative z-10 shrink-0 flex items-center justify-center rounded-full font-mono"
          style="width: 27px; height: 27px; font-size: 11px"
          :style="{
            background: stateFor(s.key) === 'done' ? 'var(--sent)' : stateFor(s.key) === 'active' ? 'var(--signal)' : 'var(--paper-2)',
            color: stateFor(s.key) === 'todo' ? 'var(--ink-3)' : '#fff',
            border: stateFor(s.key) === 'todo' ? '1px solid var(--line-2)' : 'none',
          }"
        >
          <span v-if="stateFor(s.key) === 'done'">✓</span>
          <span v-else-if="stateFor(s.key) === 'active'" class="rounded-full" style="width: 9px; height: 9px; background: #fff; animation: blink 1s infinite"></span>
          <span v-else>{{ String(i + 1).padStart(2, '0') }}</span>
        </span>

        <!-- etiket -->
        <div class="pt-0.5">
          <div
            class="text-[15px]"
            :style="{
              color: stateFor(s.key) === 'todo' ? 'var(--ink-3)' : 'var(--ink)',
              fontWeight: stateFor(s.key) === 'active' ? 600 : 500,
            }"
          >{{ t('prog.step.' + s.key + '.label') }}</div>
          <div class="label mt-0.5">{{ t('prog.step.' + s.key + '.hint') }}</div>
        </div>
      </li>
    </ol>

    <p v-if="!isFailed && store.progressMessage" class="mt-6 text-sm flex items-center gap-2" style="color: var(--ink-2)">
      <span class="inline-block rounded-full" style="width: 12px; height: 12px; border: 2px solid var(--signal-soft); border-top-color: var(--signal); animation: spin 0.7s linear infinite"></span>
      {{ store.progressMessage }}
    </p>
  </div>
</template>
