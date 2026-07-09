<script setup>
import { ref, reactive } from 'vue'
import { useSessionStore } from '../stores/session'
import RecipientPicker from './RecipientPicker.vue'
import { t } from '../i18n'

const store = useSessionStore()

const form = reactive({
  title: '',
  participantsText: '',
  summaryStyle: 'decisions_actions',
  sendPolicy: 'immediate',
  cancelWindowMinutes: 5,
})

const recipients = ref([]) // seçili alıcı e-postaları (RecipientPicker'dan)
const mode = ref('record') // 'record' | 'file'
const file = ref(null)
const fileName = ref('')

const submitting = ref(false)
const error = ref('')

const styles = [
  { value: 'decisions_actions', labelKey: 'setup.style.decisions_actions' },
  { value: 'full_minutes', labelKey: 'setup.style.full_minutes' },
  { value: 'short', labelKey: 'setup.style.short' },
]

function onFile(e) {
  const f = e.target.files?.[0] || null
  file.value = f
  fileName.value = f ? f.name : ''
}

function parseList(text) {
  return text.split(/[\n,;]+/).map((s) => s.trim()).filter(Boolean)
}

async function submit() {
  error.value = ''

  if (!form.title.trim()) {
    error.value = t('setup.err.title')
    return
  }
  if (recipients.value.length === 0) {
    error.value = t('setup.err.recipients')
    return
  }
  if (mode.value === 'file' && !file.value) {
    error.value = t('setup.err.file')
    return
  }

  submitting.value = true
  try {
    await store.create({
      title: form.title.trim(),
      recipients: recipients.value,
      participants: parseList(form.participantsText),
      summary_style: form.summaryStyle,
      send_policy: form.sendPolicy,
      cancel_window_seconds:
        form.sendPolicy === 'cancel_window'
          ? Math.max(0, Math.round(form.cancelWindowMinutes * 60))
          : 0,
    })
    if (mode.value === 'record') {
      await store.start()
    } else {
      await store.uploadFile(file.value)
    }
  } catch (e) {
    error.value = e.message || t('setup.err.generic')
    submitting.value = false
  }
}
</script>

<template>
  <form class="space-y-9" @submit.prevent="submit">
    <div class="rise">
      <h2 class="serif-title text-[1.7rem] mb-1">{{ t('setup.title') }}</h2>
      <p class="text-sm" style="color: var(--ink-2)">
        {{ t('setup.subtitle') }}
      </p>
    </div>

    <!-- 01 · Kaynak -->
    <section class="rise" style="animation-delay: 40ms">
      <div class="flex items-center gap-3 mb-3">
        <span class="label">01</span><span class="label">{{ t('setup.source') }}</span>
        <hr class="rule flex-1" />
      </div>
      <div class="grid grid-cols-2 gap-2.5">
        <button type="button" class="select-card text-left" :data-on="mode === 'record'" @click="mode = 'record'">
          <div class="text-[15px] font-semibold flex items-center gap-2">
            <span style="color: var(--live)">●</span> {{ t('setup.liveRecord') }}
          </div>
          <div class="text-xs mt-0.5" style="color: var(--ink-3)">{{ t('setup.liveRecordHint') }}</div>
        </button>
        <button type="button" class="select-card text-left" :data-on="mode === 'file'" @click="mode = 'file'">
          <div class="text-[15px] font-semibold flex items-center gap-2">
            <span style="color: var(--signal)">↑</span> {{ t('setup.uploadFile') }}
          </div>
          <div class="text-xs mt-0.5" style="color: var(--ink-3)">{{ t('setup.uploadFileHint') }}</div>
        </button>
      </div>
      <label
        v-if="mode === 'file'"
        class="mt-2.5 flex items-center gap-3 field cursor-pointer"
        style="border-style: dashed"
      >
        <span class="font-mono text-lg" style="color: var(--signal)">♪</span>
        <span class="truncate" :style="fileName ? 'color:var(--ink)' : 'color:var(--ink-3)'">
          {{ fileName || t('setup.filePlaceholder') }}
        </span>
        <input type="file" accept="audio/*" class="hidden" @change="onFile" />
      </label>
    </section>

    <!-- 02 · Başlık -->
    <section class="rise" style="animation-delay: 80ms">
      <div class="flex items-center gap-3 mb-3">
        <span class="label">02</span><span class="label">{{ t('setup.titleLabel') }}</span>
        <hr class="rule flex-1" />
      </div>
      <input v-model="form.title" type="text" :placeholder="t('setup.titlePlaceholder')" class="field" />
      <p class="text-xs mt-1.5" style="color: var(--ink-3)">{{ t('setup.titleHint') }}</p>
    </section>

    <!-- 03 · Alıcılar -->
    <section class="rise" style="animation-delay: 120ms">
      <div class="flex items-center gap-3 mb-3">
        <span class="label">03</span><span class="label">{{ t('setup.recipients') }}</span>
        <hr class="rule flex-1" />
      </div>
      <RecipientPicker v-model="recipients" />
    </section>

    <!-- 04 · Katılımcılar + Biçim -->
    <section class="rise" style="animation-delay: 160ms">
      <div class="flex items-center gap-3 mb-3">
        <span class="label">04</span><span class="label">{{ t('setup.participantsFormat') }}</span>
        <hr class="rule flex-1" />
      </div>
      <textarea
        v-model="form.participantsText"
        rows="2"
        :placeholder="t('setup.participantsPlaceholder')"
        class="field mb-2.5 resize-none"
      ></textarea>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="s in styles"
          :key="s.value"
          type="button"
          class="chip"
          :data-on="form.summaryStyle === s.value"
          @click="form.summaryStyle = s.value"
        >{{ t(s.labelKey) }}</button>
      </div>
    </section>

    <!-- 05 · Gönderim -->
    <section class="rise" style="animation-delay: 200ms">
      <div class="flex items-center gap-3 mb-3">
        <span class="label">05</span><span class="label">{{ t('setup.sendPolicy') }}</span>
        <hr class="rule flex-1" />
      </div>
      <div class="grid grid-cols-2 gap-2.5">
        <button type="button" class="select-card text-left" :data-on="form.sendPolicy === 'immediate'" @click="form.sendPolicy = 'immediate'">
          <div class="text-[15px] font-semibold">{{ t('setup.immediate') }}</div>
          <div class="text-xs mt-0.5" style="color: var(--ink-3)">{{ t('setup.immediateHint') }}</div>
        </button>
        <button type="button" class="select-card text-left" :data-on="form.sendPolicy === 'cancel_window'" @click="form.sendPolicy = 'cancel_window'">
          <div class="text-[15px] font-semibold">{{ t('setup.cancelWindow') }}</div>
          <div class="text-xs mt-0.5" style="color: var(--ink-3)">{{ t('setup.cancelWindowHint') }}</div>
        </button>
      </div>
      <div v-if="form.sendPolicy === 'cancel_window'" class="flex items-center gap-2 mt-2.5">
        <span class="text-sm" style="color: var(--ink-2)">{{ t('setup.window') }}</span>
        <input v-model.number="form.cancelWindowMinutes" type="number" min="1" max="60" class="field font-mono" style="width: 5rem" />
        <span class="text-sm" style="color: var(--ink-2)">{{ t('setup.minutes') }}</span>
      </div>
    </section>

    <p v-if="error" class="text-sm rise" style="color: var(--danger)">{{ error }}</p>

    <button
      type="submit"
      :disabled="submitting"
      class="btn w-full py-4 text-[15px] rise"
      :class="mode === 'record' ? 'btn-live' : 'btn-signal'"
      style="animation-delay: 240ms"
    >
      <span v-if="submitting">{{ t('setup.processing') }}</span>
      <span v-else-if="mode === 'record'">{{ t('setup.startRecord') }}</span>
      <span v-else>{{ t('setup.uploadSummarize') }}</span>
    </button>
  </form>
</template>
