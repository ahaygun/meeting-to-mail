<script setup>
import { computed } from 'vue'
import { useSessionStore, Phase } from '../stores/session'

const store = useSessionStore()

const content = computed(() => store.summary?.content ?? null)
const cancelled = computed(() => store.phase === Phase.Cancelled)

function newMeeting() {
  store.reset()
}
</script>

<template>
  <div class="space-y-6">
    <div
      class="rounded-lg border p-3 text-sm"
      :class="
        cancelled
          ? 'border-amber-500/40 bg-amber-500/10 text-amber-200'
          : 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200'
      "
    >
      <span v-if="cancelled">⛔ Gönderim iptal edildi. Özet kaydedildi ama e-posta yollanmadı.</span>
      <span v-else>✅ Özet hazırlandı ve alıcılara gönderildi.</span>
    </div>

    <div>
      <h2 class="text-lg font-semibold">{{ store.title }}</h2>
    </div>

    <div v-if="content" class="space-y-5">
      <p class="text-base text-slate-100">{{ content.headline }}</p>

      <section v-if="content.key_points?.length">
        <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">Ana Maddeler</h3>
        <ul class="space-y-1.5">
          <li v-for="(k, i) in content.key_points" :key="i" class="flex gap-2 text-sm">
            <span class="text-indigo-400">•</span><span>{{ k }}</span>
          </li>
        </ul>
      </section>

      <section v-if="content.decisions?.length">
        <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">Kararlar</h3>
        <ul class="space-y-1.5">
          <li v-for="(d, i) in content.decisions" :key="i" class="flex gap-2 text-sm">
            <span class="text-emerald-400">✓</span><span>{{ d }}</span>
          </li>
        </ul>
      </section>

      <section v-if="content.action_items?.length">
        <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">Aksiyon Maddeleri</h3>
        <ul class="space-y-2">
          <li
            v-for="(a, i) in content.action_items"
            :key="i"
            class="rounded-lg bg-slate-800 border border-slate-700 px-3 py-2 text-sm"
          >
            <div class="font-medium">{{ a.task }}</div>
            <div class="text-xs text-slate-400 mt-0.5">
              <span v-if="a.owner">👤 {{ a.owner }}</span>
              <span v-if="a.due"> · 📅 {{ a.due }}</span>
            </div>
          </li>
        </ul>
      </section>
    </div>
    <p v-else class="text-sm text-slate-400">Özet yükleniyor…</p>

    <section v-if="store.deliveries.length">
      <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">Gönderim Durumu</h3>
      <ul class="space-y-1">
        <li
          v-for="d in store.deliveries"
          :key="d.id"
          class="flex items-center justify-between text-sm rounded-lg bg-slate-800/60 px-3 py-2"
        >
          <span>{{ d.recipient }}</span>
          <span
            class="text-xs px-2 py-0.5 rounded-full"
            :class="
              d.status === 'sent'
                ? 'bg-emerald-500/20 text-emerald-300'
                : d.status === 'failed'
                  ? 'bg-red-500/20 text-red-300'
                  : 'bg-slate-600/40 text-slate-300'
            "
            >{{ d.status }}</span
          >
        </li>
      </ul>
    </section>

    <button
      @click="newMeeting"
      class="w-full rounded-xl bg-slate-700 hover:bg-slate-600 px-4 py-3.5 text-base font-semibold transition"
    >
      Yeni Toplantı
    </button>
  </div>
</template>
