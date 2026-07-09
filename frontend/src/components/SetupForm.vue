<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useSessionStore } from '../stores/session'
import { api } from '../lib/api'

const store = useSessionStore()

const form = reactive({
  title: '',
  participantsText: '',
  summaryStyle: 'decisions_actions',
  sendPolicy: 'immediate',
  cancelWindowMinutes: 5,
})

const mode = ref('record') // 'record' | 'file'
const file = ref(null)
const fileName = ref('')

const contacts = ref([]) // [{id,email,name}]
const selected = ref(new Set()) // seçili e-postalar
const newEmail = ref('')

const submitting = ref(false)
const error = ref('')

const styles = [
  { value: 'decisions_actions', label: 'Kararlar & Aksiyonlar' },
  { value: 'full_minutes', label: 'Tam Tutanak' },
  { value: 'short', label: 'Kısa Özet' },
]

const selectedCount = computed(() => selected.value.size)

async function loadContacts() {
  try {
    contacts.value = await api.listContacts()
  } catch {
    contacts.value = []
  }
}
onMounted(loadContacts)

function toggle(email) {
  const s = selected.value
  if (s.has(email)) s.delete(email)
  else s.add(email)
  selected.value = new Set(s) // reaktifliği tetikle
}

async function addNewEmail() {
  const e = newEmail.value.trim()
  if (!e) return
  if (!e.includes('@')) {
    error.value = 'Geçersiz e-posta.'
    return
  }
  error.value = ''
  // Rehberde yoksa ekle.
  if (!contacts.value.some((c) => c.email === e)) {
    try {
      await api.createContact(e)
      await loadContacts()
    } catch {
      /* yine de seç */
    }
  }
  selected.value = new Set(selected.value).add(e)
  newEmail.value = ''
}

async function removeContact(c) {
  try {
    await api.deleteContact(c.id)
    contacts.value = contacts.value.filter((x) => x.id !== c.id)
    if (selected.value.has(c.email)) {
      const s = new Set(selected.value)
      s.delete(c.email)
      selected.value = s
    }
  } catch {
    /* yoksay */
  }
}

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
  const recipients = [...selected.value]

  if (!form.title.trim()) {
    error.value = 'Toplantı başlığı gerekli.'
    return
  }
  if (recipients.length === 0) {
    error.value = 'En az bir alıcı seçin veya ekleyin.'
    return
  }
  if (mode.value === 'file' && !file.value) {
    error.value = 'Bir ses dosyası seçin.'
    return
  }

  submitting.value = true
  try {
    await store.create({
      title: form.title.trim(),
      recipients,
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
    error.value = e.message || 'İşlem başlatılamadı.'
    submitting.value = false
  }
}
</script>

<template>
  <form class="space-y-5" @submit.prevent="submit">
    <div>
      <h2 class="text-lg font-semibold mb-1">Toplantıyı Kur</h2>
      <p class="text-sm text-slate-400">Alıcıları ve ayarları belirle, sonra kaydet ya da dosya yükle.</p>
    </div>

    <!-- Kaynak modu -->
    <div class="grid grid-cols-2 gap-2">
      <button
        type="button"
        @click="mode = 'record'"
        :class="[
          'rounded-lg border px-3 py-2.5 text-sm text-center transition',
          mode === 'record' ? 'border-indigo-500 bg-indigo-500/10' : 'border-slate-700 bg-slate-800',
        ]"
      >
        🎙️ Canlı Kayıt
      </button>
      <button
        type="button"
        @click="mode = 'file'"
        :class="[
          'rounded-lg border px-3 py-2.5 text-sm text-center transition',
          mode === 'file' ? 'border-indigo-500 bg-indigo-500/10' : 'border-slate-700 bg-slate-800',
        ]"
      >
        📁 Dosya Yükle
      </button>
    </div>

    <div v-if="mode === 'file'" class="space-y-1">
      <label class="text-sm font-medium">Ses Dosyası</label>
      <label
        class="flex items-center gap-3 rounded-lg bg-slate-800 border border-dashed border-slate-600 px-3 py-3 text-sm cursor-pointer hover:border-indigo-500/60"
      >
        <span class="text-xl">📁</span>
        <span class="text-slate-300 truncate">{{ fileName || 'Dosya seç (mp3, m4a, wav, webm…)' }}</span>
        <input type="file" accept="audio/*" class="hidden" @change="onFile" />
      </label>
    </div>

    <div class="space-y-1">
      <label class="text-sm font-medium">Toplantı Başlığı</label>
      <input
        v-model="form.title"
        type="text"
        placeholder="ör. Haftalık Ekip Toplantısı"
        class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
      />
      <p class="text-xs text-slate-500">E-posta konusu olarak kullanılır.</p>
    </div>

    <!-- Alıcı seçici -->
    <div class="space-y-2">
      <label class="text-sm font-medium">
        Alıcılar
        <span v-if="selectedCount" class="text-indigo-400">({{ selectedCount }} seçili)</span>
      </label>

      <div v-if="contacts.length" class="flex flex-wrap gap-2">
        <span
          v-for="c in contacts"
          :key="c.id"
          :class="[
            'group inline-flex items-center gap-1.5 rounded-full border pl-3 pr-2 py-1.5 text-sm cursor-pointer transition',
            selected.has(c.email)
              ? 'border-indigo-500 bg-indigo-500/15 text-indigo-200'
              : 'border-slate-700 bg-slate-800 text-slate-300 hover:border-slate-500',
          ]"
          @click="toggle(c.email)"
        >
          <span v-if="selected.has(c.email)">✓</span>
          <span class="truncate max-w-[10rem]">{{ c.name || c.email }}</span>
          <button
            type="button"
            title="Rehberden sil"
            class="ml-0.5 text-slate-500 hover:text-red-400"
            @click.stop="removeContact(c)"
          >
            ×
          </button>
        </span>
      </div>
      <p v-else class="text-xs text-slate-500">Kayıtlı alıcı yok — aşağıdan ekleyebilirsin.</p>

      <div class="flex gap-2">
        <input
          v-model="newEmail"
          type="email"
          placeholder="yeni@ornek.com"
          @keydown.enter.prevent="addNewEmail"
          class="flex-1 rounded-lg bg-slate-800 border border-slate-700 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <button
          type="button"
          @click="addNewEmail"
          class="rounded-lg bg-slate-700 hover:bg-slate-600 px-4 text-sm font-medium"
        >
          Ekle
        </button>
      </div>
      <p class="text-xs text-slate-500">Eklenen adresler kaydedilir; sonraki toplantıda seçilebilir.</p>
    </div>

    <div class="space-y-1">
      <label class="text-sm font-medium">Katılımcılar <span class="text-slate-500">(opsiyonel)</span></label>
      <textarea
        v-model="form.participantsText"
        rows="2"
        placeholder="Ali, Ayşe, Mehmet"
        class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
      ></textarea>
    </div>

    <div class="space-y-1">
      <label class="text-sm font-medium">Özet Stili</label>
      <select
        v-model="form.summaryStyle"
        class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
      >
        <option v-for="s in styles" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>
    </div>

    <div class="space-y-2">
      <label class="text-sm font-medium">Gönderim Politikası</label>
      <div class="grid grid-cols-2 gap-2">
        <button
          type="button"
          @click="form.sendPolicy = 'immediate'"
          :class="[
            'rounded-lg border px-3 py-2.5 text-sm text-left transition',
            form.sendPolicy === 'immediate' ? 'border-indigo-500 bg-indigo-500/10' : 'border-slate-700 bg-slate-800',
          ]"
        >
          <div class="font-medium">Hemen gönder</div>
          <div class="text-xs text-slate-400">Özet hazır olunca yollar.</div>
        </button>
        <button
          type="button"
          @click="form.sendPolicy = 'cancel_window'"
          :class="[
            'rounded-lg border px-3 py-2.5 text-sm text-left transition',
            form.sendPolicy === 'cancel_window' ? 'border-indigo-500 bg-indigo-500/10' : 'border-slate-700 bg-slate-800',
          ]"
        >
          <div class="font-medium">İptal pencereli</div>
          <div class="text-xs text-slate-400">Göndermeden önce bekler.</div>
        </button>
      </div>
      <div v-if="form.sendPolicy === 'cancel_window'" class="flex items-center gap-2 pt-1">
        <label class="text-sm text-slate-300">Pencere:</label>
        <input
          v-model.number="form.cancelWindowMinutes"
          type="number"
          min="1"
          max="60"
          class="w-20 rounded-lg bg-slate-800 border border-slate-700 px-2 py-1.5 text-sm"
        />
        <span class="text-sm text-slate-400">dakika</span>
      </div>
    </div>

    <p v-if="error" class="text-sm text-red-400">{{ error }}</p>

    <button
      type="submit"
      :disabled="submitting"
      class="w-full rounded-xl px-4 py-4 text-base font-semibold transition disabled:opacity-50"
      :class="mode === 'record' ? 'bg-indigo-600 hover:bg-indigo-500' : 'bg-emerald-600 hover:bg-emerald-500'"
    >
      <span v-if="submitting">İşleniyor…</span>
      <span v-else-if="mode === 'record'">● Kaydı Başlat</span>
      <span v-else>⬆ Yükle ve Özetle</span>
    </button>
  </form>
</template>
