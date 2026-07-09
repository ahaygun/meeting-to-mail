<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../lib/api'
import { parseRecipients } from '../lib/emails'
import { t } from '../i18n'

// v-model: seçili e-posta dizisi.
const props = defineProps({ modelValue: { type: Array, default: () => [] } })
const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const contacts = ref([]) // [{id,email,name}]
const groups = ref([]) // [{id,name,emails}]
const selected = ref(new Set(props.modelValue))
const query = ref('')
const newEmail = ref('')
const error = ref('')

const savingGroup = ref(false)
const groupName = ref('')
const editingId = ref(null)
const editName = ref('')

function commit() {
  emit('update:modelValue', [...selected.value])
}

async function loadAll() {
  try {
    contacts.value = await api.listContacts()
  } catch {
    contacts.value = []
  }
  try {
    groups.value = await api.listGroups()
  } catch {
    groups.value = []
  }
}
onMounted(loadAll)

// Modal açıkken arka planı kilitle.
watch(open, (v) => {
  document.body.style.overflow = v ? 'hidden' : ''
  if (v) {
    error.value = ''
    loadAll()
  }
})

const selectedCount = computed(() => selected.value.size)
const selectedList = computed(() => [...selected.value])

// Görünen ad (varsa isim, yoksa e-posta).
function displayName(email) {
  const c = contacts.value.find((x) => x.email === email)
  return c && c.name ? c.name : email
}

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return contacts.value
  return contacts.value.filter(
    (c) => c.email.toLowerCase().includes(q) || (c.name || '').toLowerCase().includes(q),
  )
})

function isSel(email) {
  return selected.value.has(email)
}
function toggle(email) {
  const s = new Set(selected.value)
  if (s.has(email)) s.delete(email)
  else s.add(email)
  selected.value = s
  commit()
}
function remove(email) {
  const s = new Set(selected.value)
  s.delete(email)
  selected.value = s
  commit()
}
function clearAll() {
  selected.value = new Set()
  commit()
}

async function add() {
  const entries = parseRecipients(newEmail.value)
  if (entries.length === 0) {
    error.value = t('rp.err.validEmail')
    return
  }
  error.value = ''
  const s = new Set(selected.value)
  for (const { email, name } of entries) {
    s.add(email)
    const existing = contacts.value.find((c) => c.email === email)
    if (!existing) {
      try {
        await api.createContact(email, name)
      } catch {
        /* yine de seç */
      }
    } else if (name && !existing.name) {
      try {
        await api.updateContact(existing.id, name)
      } catch {
        /* yoksay */
      }
    }
  }
  selected.value = s
  commit()
  newEmail.value = ''
  await loadAll()
}

function onPaste(e) {
  const text = (e.clipboardData || window.clipboardData)?.getData('text') || ''
  if (/[,;\n]/.test(text) && text.includes('@')) {
    e.preventDefault()
    newEmail.value = text
    add()
  }
}

// --- Gruplar ---
function groupFullySelected(g) {
  return g.emails.length > 0 && g.emails.every((e) => selected.value.has(e))
}
function selectGroup(g) {
  const s = new Set(selected.value)
  if (groupFullySelected(g)) g.emails.forEach((e) => s.delete(e))
  else g.emails.forEach((e) => s.add(e))
  selected.value = s
  commit()
}
async function delGroup(g) {
  if (!confirm(t('rp.confirmDelGroup', { name: g.name }))) return
  try {
    await api.deleteGroup(g.id)
    groups.value = groups.value.filter((x) => x.id !== g.id)
  } catch {
    /* yoksay */
  }
}
function saveGroupStart() {
  if (selectedCount.value === 0) {
    error.value = t('rp.err.selectFirst')
    return
  }
  error.value = ''
  groupName.value = ''
  savingGroup.value = true
}
async function saveGroupConfirm() {
  const name = groupName.value.trim()
  if (!name) {
    error.value = t('rp.err.groupName')
    return
  }
  try {
    await api.createGroup(name, [...selected.value])
    savingGroup.value = false
    await loadAll()
  } catch (e) {
    error.value = e.message || t('rp.err.groupSave')
  }
}

// --- Kişi yönetimi ---
function startEdit(c) {
  editingId.value = c.id
  editName.value = c.name || ''
}
async function saveEdit(c) {
  try {
    await api.updateContact(c.id, editName.value.trim())
    c.name = editName.value.trim()
  } catch {
    /* yoksay */
  }
  editingId.value = null
}
async function del(c) {
  if (!confirm(t('rp.confirmDelContact', { who: c.name || c.email }))) return
  try {
    await api.deleteContact(c.id)
    contacts.value = contacts.value.filter((x) => x.id !== c.id)
    if (selected.value.has(c.email)) remove(c.email)
  } catch {
    /* yoksay */
  }
}
</script>

<template>
  <div>
    <!-- Seçili özet + aç butonu -->
    <div v-if="selectedList.length" class="flex flex-wrap gap-2 mb-2.5">
      <span v-for="e in selectedList" :key="e" class="chip" data-on="true">
        <span style="color: var(--signal)">✓</span>
        <span class="truncate max-w-[11rem]">{{ displayName(e) }}</span>
        <button type="button" class="opacity-60 hover:opacity-100" style="color: var(--danger)" @click="remove(e)">×</button>
      </span>
    </div>

    <button
      type="button"
      class="btn w-full py-3 flex items-center justify-center gap-2 text-[15px] font-semibold"
      style="border: 1.5px dashed var(--line-2); color: var(--ink-2)"
      @click="open = true"
    >
      <span style="color: var(--signal); font-size: 1.1em">＋</span>
      {{ selectedList.length ? t('rp.editRecipients', { n: selectedList.length }) : t('rp.selectOrAdd') }}
    </button>

    <!-- MODAL -->
    <Teleport to="body">
      <div v-if="open" class="modal-overlay" @click.self="open = false">
        <div class="modal-card">
          <!-- Başlık -->
          <div class="flex items-center justify-between px-5 py-4" style="border-bottom: 1px solid var(--line)">
            <div>
              <div class="serif-title text-[1.3rem]">{{ t('rp.recipients') }}</div>
              <div class="label mt-0.5">{{ t('rp.selected', { n: selectedCount }) }}</div>
            </div>
            <button type="button" class="icon-btn" style="font-size: 1.1rem" @click="open = false">×</button>
          </div>

          <!-- İçerik (kaydırılır) -->
          <div class="px-5 py-4 overflow-y-auto space-y-5" style="flex: 1">
            <!-- Ekle -->
            <div>
              <div class="label mb-2">{{ t('rp.addEmail') }}</div>
              <div class="flex gap-2">
                <input
                  v-model="newEmail"
                  type="text"
                  :placeholder="t('rp.emailPlaceholder')"
                  class="field flex-1"
                  @keydown.enter.prevent="add"
                  @paste="onPaste"
                />
                <button type="button" class="btn btn-signal px-5 shrink-0" @click="add">{{ t('rp.add') }}</button>
              </div>
              <p class="text-xs mt-1.5" style="color: var(--ink-3)">
                {{ t('rp.pasteHint') }}
              </p>
            </div>

            <!-- Gruplar -->
            <div v-if="groups.length">
              <div class="label mb-2">{{ t('rp.groupsHint') }}</div>
              <div class="flex flex-wrap gap-2">
                <div
                  v-for="g in groups"
                  :key="g.id"
                  class="group-pill"
                  :style="groupFullySelected(g) ? 'border-color:var(--signal);background:var(--signal-soft)' : ''"
                  @click="selectGroup(g)"
                >
                  <span style="opacity: 0.6">▦</span>
                  <span class="text-sm font-medium">{{ g.name }}</span>
                  <span class="font-mono text-[11px] px-1.5 py-0.5 rounded-full" style="background: var(--paper-3); color: var(--ink-3)">{{ g.emails.length }}</span>
                  <button type="button" class="icon-btn danger" style="width: 26px; height: 26px" :title="t('rp.deleteGroup')" @click.stop="delGroup(g)">🗑</button>
                </div>
              </div>
            </div>

            <!-- Kişiler -->
            <div>
              <div class="flex items-center justify-between mb-2">
                <span class="label">{{ t('rp.contacts') }}</span>
                <button v-if="selectedCount" type="button" class="btn btn-ghost text-xs px-2.5 py-1" @click="clearAll">{{ t('rp.clearSelection') }}</button>
              </div>
              <input v-model="query" type="text" :placeholder="t('rp.searchContacts')" class="field mb-1" />
              <div v-if="filtered.length" style="max-height: 42vh; overflow-y: auto">
                <label v-for="c in filtered" :key="c.id" class="contact-row">
                  <input type="checkbox" :checked="isSel(c.email)" @change="toggle(c.email)" />
                  <div class="flex-1 min-w-0" v-if="editingId !== c.id">
                    <div class="text-[15px] truncate">{{ c.name || c.email }}</div>
                    <div v-if="c.name" class="text-xs truncate" style="color: var(--ink-3)">{{ c.email }}</div>
                  </div>
                  <input
                    v-else
                    v-model="editName"
                    class="field flex-1 py-1"
                    :placeholder="t('rp.namePlaceholder')"
                    @keydown.enter.prevent="saveEdit(c)"
                    @blur="saveEdit(c)"
                    @click.prevent.stop
                  />
                  <button type="button" class="icon-btn" :title="t('rp.editName')" @click.prevent.stop="startEdit(c)">✎</button>
                  <button type="button" class="icon-btn danger" :title="t('rp.deleteContact')" @click.prevent.stop="del(c)">🗑</button>
                </label>
              </div>
              <p v-else class="text-sm py-3" style="color: var(--ink-3)">
                {{ query.trim() ? t('rp.noMatch') : t('rp.noContacts') }}
              </p>
            </div>

            <!-- Grup kaydet (inline) -->
            <div v-if="savingGroup" class="flex gap-2">
              <input
                v-model="groupName"
                type="text"
                :placeholder="t('rp.groupNamePlaceholder')"
                class="field flex-1"
                @keydown.enter.prevent="saveGroupConfirm"
              />
              <button type="button" class="btn btn-signal px-4 shrink-0" @click="saveGroupConfirm">{{ t('rp.save') }}</button>
              <button type="button" class="btn btn-ghost px-3 shrink-0" @click="savingGroup = false">{{ t('rp.cancel') }}</button>
            </div>

            <p v-if="error" class="text-sm" style="color: var(--danger)">{{ error }}</p>
          </div>

          <!-- Alt bar -->
          <div class="flex items-center gap-2 px-5 py-3.5" style="border-top: 1px solid var(--line); background: var(--paper-2)">
            <button
              type="button"
              class="btn btn-ghost px-4 py-2.5 text-sm"
              :disabled="selectedCount === 0 || savingGroup"
              @click="saveGroupStart"
            >
              {{ t('rp.saveGroup') }}
            </button>
            <button type="button" class="btn btn-signal flex-1 py-2.5" @click="open = false">
              {{ t('rp.done', { n: selectedCount }) }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
