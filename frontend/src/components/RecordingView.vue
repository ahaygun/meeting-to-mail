<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useSessionStore } from '../stores/session'
import { useRecorder } from '../composables/useRecorder'

const store = useSessionStore()
const rec = useRecorder(() => store.id)
const stopping = ref(false)
const startError = ref('')

onMounted(async () => {
  try {
    await rec.start()
  } catch (e) {
    startError.value =
      'Mikrofona erişilemedi. Tarayıcı izinlerini kontrol edin. (' + (e.message || e) + ')'
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
    startError.value = e.message || 'Sonlandırma başarısız.'
    stopping.value = false
  }
}
</script>

<template>
  <div class="flex flex-col items-center text-center gap-6 pt-6">
    <div>
      <div class="text-sm text-slate-400">Kaydediliyor</div>
      <h2 class="text-lg font-semibold mt-0.5">{{ store.title }}</h2>
    </div>

    <!-- Canlı sayaç + nabız halkası -->
    <div class="relative flex items-center justify-center">
      <span
        v-if="rec.recording.value"
        class="absolute inline-flex h-40 w-40 rounded-full bg-red-500/20 animate-ping"
      ></span>
      <div
        class="relative h-40 w-40 rounded-full bg-slate-800 border-4 border-red-500/70 flex flex-col items-center justify-center"
      >
        <span class="text-3xl font-mono tabular-nums">{{ rec.elapsedLabel.value }}</span>
        <span class="mt-1 text-xs text-red-400 flex items-center gap-1">
          <span class="h-2 w-2 rounded-full bg-red-500 animate-pulse"></span> CANLI
        </span>
      </div>
    </div>

    <div class="text-xs text-slate-400 space-y-1">
      <div>Yüklenen parça: {{ rec.uploaded.value }}<span v-if="rec.pending.value"> · sırada {{ rec.pending.value }}</span></div>
      <p class="max-w-xs">
        📱 Telefonda: ekranı açık ve uygulamayı önde tutun — arka plana geçince kayıt durabilir.
      </p>
    </div>

    <p v-if="rec.errorMsg.value" class="text-xs text-amber-400">{{ rec.errorMsg.value }}</p>
    <p v-if="startError" class="text-sm text-red-400">{{ startError }}</p>

    <button
      @click="stopAndProcess"
      :disabled="stopping"
      class="w-full max-w-xs rounded-xl bg-red-600 hover:bg-red-500 disabled:opacity-50 px-4 py-4 text-base font-semibold transition"
    >
      {{ stopping ? 'Sonlandırılıyor…' : '■ Durdur ve Özetle' }}
    </button>
  </div>
</template>
