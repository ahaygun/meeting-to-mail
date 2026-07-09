<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useSessionStore } from '../stores/session'
import { useRecorder } from '../composables/useRecorder'
import { t } from '../i18n'

const store = useSessionStore()
const rec = useRecorder(() => store.id)
const stopping = ref(false)
const startError = ref('')

onMounted(async () => {
  try {
    await rec.start()
  } catch (e) {
    startError.value = t('rec.micError', { e: e.message || e })
  }
})

onBeforeUnmount(() => {
  if (rec.recording.value) rec.stop()
})

async function stopAndProcess() {
  stopping.value = true
  try {
    await rec.stop()
    await store.finalize()
  } catch (e) {
    startError.value = e.message || t('rec.finalizeError')
    stopping.value = false
  }
}
</script>

<template>
  <div class="flex flex-col items-center text-center gap-8 pt-4 rise">
    <div>
      <div class="label mb-1.5 flex items-center justify-center gap-1.5">
        <span class="inline-block w-1.5 h-1.5 rounded-full" style="background: var(--live); animation: blink 1s infinite"></span>
        {{ t('rec.recording') }}
      </div>
      <h2 class="serif-title text-[1.7rem]">{{ store.title }}</h2>
    </div>

    <!-- Kayıt kadranı -->
    <div class="relative flex items-center justify-center my-2">
      <span
        v-if="rec.recording.value"
        class="absolute rounded-full"
        style="width: 190px; height: 190px; border: 1.5px solid var(--live); animation: ring-pulse 2s cubic-bezier(0,0,0.2,1) infinite"
      ></span>
      <span
        v-if="rec.recording.value"
        class="absolute rounded-full"
        style="width: 190px; height: 190px; border: 1.5px solid var(--live); animation: ring-pulse 2s cubic-bezier(0,0,0.2,1) infinite 1s"
      ></span>
      <div
        class="relative rounded-full flex flex-col items-center justify-center panel"
        style="width: 190px; height: 190px; border-color: var(--live)"
      >
        <div class="font-mono tabular-nums" style="font-size: 2.2rem; letter-spacing: -0.02em">
          {{ rec.elapsedLabel.value }}
        </div>
        <!-- ekolayzer -->
        <div class="flex items-end gap-1 mt-2" style="height: 18px">
          <span
            v-for="i in 7"
            :key="i"
            class="w-1 rounded-full"
            :style="`height:18px;background:var(--live);transform-origin:bottom;animation:eq ${0.7 + (i % 3) * 0.25}s ease-in-out infinite ${i * 0.08}s`"
          ></span>
        </div>
      </div>
    </div>

    <div class="label" style="color: var(--ink-2)">
      {{ t('rec.uploaded', { n: rec.uploaded.value }) }}<span v-if="rec.pending.value"> · {{ t('rec.queued', { n: rec.pending.value }) }}</span>
    </div>

    <div class="panel px-4 py-3 max-w-sm text-sm" style="color: var(--ink-2)">
      {{ t('rec.phoneHint') }}
    </div>

    <p v-if="rec.errorMsg.value" class="text-xs" style="color: var(--signal)">{{ rec.errorMsg.value }}</p>
    <p v-if="startError" class="text-sm" style="color: var(--danger)">{{ startError }}</p>

    <button @click="stopAndProcess" :disabled="stopping" class="btn btn-live w-full max-w-xs py-4 text-[15px]">
      {{ stopping ? t('rec.stopping') : t('rec.stop') }}
    </button>
  </div>
</template>
