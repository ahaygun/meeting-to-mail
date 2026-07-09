<script setup>
import { computed } from 'vue'
import { useSessionStore, Phase } from '../stores/session'
import { t } from '../i18n'

const store = useSessionStore()

const content = computed(() => store.summary?.content ?? null)
const cancelled = computed(() => store.phase === Phase.Cancelled)

function newMeeting() {
  store.reset()
}
</script>

<template>
  <div class="space-y-6 rise">
    <!-- durum bandı -->
    <div
      class="flex items-center gap-2.5 px-4 py-2.5 rounded-xl text-sm"
      :style="cancelled
        ? 'background:var(--pastel-butter);color:var(--ink)'
        : 'background:var(--pastel-sage);color:var(--ink)'"
    >
      <span class="rounded-full w-2 h-2" :style="`background:${cancelled ? 'var(--danger)' : 'var(--sent)'}`"></span>
      <span v-if="cancelled">{{ t('sum.cancelledBanner') }}</span>
      <span v-else>{{ t('sum.sentBanner') }}</span>
    </div>

    <!-- belge -->
    <article class="panel overflow-hidden">
      <!-- letterhead -->
      <div class="px-6 pt-6 pb-4" style="border-bottom: 1px solid var(--line)">
        <div class="flex items-center justify-between mb-3">
          <span class="label">{{ t('sum.docLabel') }}</span>
          <span class="label inline-flex items-center gap-1.5">
            <span v-if="!cancelled" class="rounded-full w-1.5 h-1.5" style="background: var(--sent)"></span>
            {{ cancelled ? t('sum.draft') : t('sum.sent') }}
          </span>
        </div>
        <h2 class="serif-title text-[1.9rem]">{{ store.title }}</h2>
        <p v-if="content" class="text-[15px] mt-2" style="color: var(--ink-2)">{{ content.headline }}</p>
      </div>

      <div v-if="content" class="px-6 py-5 space-y-6">
        <section v-if="content.key_points?.length">
          <div class="label mb-2.5">{{ t('sum.keyPoints') }}</div>
          <ul class="space-y-2">
            <li v-for="(k, i) in content.key_points" :key="i" class="flex gap-2.5 text-[15px]">
              <span class="font-mono text-xs pt-1 shrink-0" style="color: var(--ink-3)">{{ String(i + 1).padStart(2, '0') }}</span>
              <span>{{ k }}</span>
            </li>
          </ul>
        </section>

        <hr class="rule" v-if="content.decisions?.length" />

        <section v-if="content.decisions?.length">
          <div class="label mb-2.5" style="color: var(--sent)">{{ t('sum.decisions') }}</div>
          <ul class="space-y-2">
            <li v-for="(d, i) in content.decisions" :key="i" class="flex gap-2.5 text-[15px]">
              <span class="shrink-0 pt-0.5" style="color: var(--sent)">✓</span>
              <span>{{ d }}</span>
            </li>
          </ul>
        </section>

        <hr class="rule" v-if="content.action_items?.length" />

        <section v-if="content.action_items?.length">
          <div class="label mb-2.5" style="color: var(--signal)">{{ t('sum.actionItems') }}</div>
          <ul class="space-y-2.5">
            <li
              v-for="(a, i) in content.action_items"
              :key="i"
              class="rounded-xl px-3.5 py-2.5"
              style="background: var(--paper-3); border: 1px solid var(--line)"
            >
              <div class="text-[15px] font-medium">{{ a.task }}</div>
              <div class="label mt-1 flex gap-3" v-if="a.due">
                <span>◷ {{ a.due }}</span>
              </div>
            </li>
          </ul>
        </section>
      </div>
      <div v-else class="px-6 py-5 text-sm" style="color: var(--ink-3)">{{ t('sum.loading') }}</div>

      <!-- alıcı satırı -->
      <div v-if="store.deliveries.length" class="px-6 py-4" style="border-top: 1px solid var(--line); background: var(--paper-3)">
        <div class="label mb-2">{{ t('sum.delivery') }}</div>
        <ul class="space-y-1.5">
          <li v-for="d in store.deliveries" :key="d.id" class="flex items-center justify-between text-sm">
            <span class="font-mono text-[13px]" style="color: var(--ink-2)">{{ d.recipient }}</span>
            <span
              class="label px-2 py-0.5 rounded-full"
              :style="d.status === 'sent'
                ? 'background:var(--pastel-sage);color:var(--sent)'
                : d.status === 'failed'
                  ? 'background:color-mix(in srgb,var(--danger) 12%,transparent);color:var(--danger)'
                  : 'background:var(--paper-2);color:var(--ink-3)'"
            >{{ d.status === 'sent' ? t('sum.statusSent') : d.status === 'failed' ? t('sum.statusFailed') : d.status }}</span>
          </li>
        </ul>
      </div>
    </article>

    <button @click="newMeeting" class="btn btn-ghost w-full py-3.5 text-[15px]">{{ t('sum.newMeeting') }}</button>
  </div>
</template>
