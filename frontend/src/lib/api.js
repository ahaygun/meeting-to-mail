// Backend API istemcisi. Taban URL, ortam değişkeninden ya da varsayılan olarak
// aynı hosttaki :8080'den okunur (telefondan LAN erişiminde de çalışır).
const BASE =
  import.meta.env.VITE_API_BASE ||
  `${window.location.protocol}//${window.location.hostname}:8080`

async function req(path, opts = {}) {
  const res = await fetch(`${BASE}${path}`, opts)
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) msg = body.error
    } catch {
      /* gövde JSON değilse yoksay */
    }
    throw new Error(msg)
  }
  if (res.status === 204) return null
  const ct = res.headers.get('content-type') || ''
  return ct.includes('application/json') ? res.json() : res.text()
}

export const api = {
  base: BASE,

  createSession(payload) {
    return req('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },

  listSessions(limit = 50) {
    return req(`/api/sessions?limit=${limit}`)
  },

  // --- Kişiler (kayıtlı alıcılar) ---
  listContacts() {
    return req('/api/contacts')
  },

  createContact(email, name = '') {
    return req('/api/contacts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, name }),
    })
  },

  updateContact(id, name) {
    return req(`/api/contacts/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
  },

  deleteContact(id) {
    return req(`/api/contacts/${id}`, { method: 'DELETE' })
  },

  // --- Gruplar (dağıtım listeleri) ---
  listGroups() {
    return req('/api/groups')
  },

  createGroup(name, emails) {
    return req('/api/groups', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, emails }),
    })
  },

  deleteGroup(id) {
    return req(`/api/groups/${id}`, { method: 'DELETE' })
  },

  getSession(id) {
    return req(`/api/sessions/${id}`)
  },

  start(id) {
    return req(`/api/sessions/${id}/start`, { method: 'POST' })
  },

  uploadChunk(id, seq, blob) {
    return req(`/api/sessions/${id}/chunks?seq=${seq}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/octet-stream' },
      body: blob,
    })
  },

  finalize(id) {
    return req(`/api/sessions/${id}/finalize`, { method: 'POST' })
  },

  cancel(id) {
    return req(`/api/sessions/${id}/cancel`, { method: 'POST' })
  },

  getSummary(id) {
    return req(`/api/sessions/${id}/summary`)
  },

  getTranscript(id) {
    return req(`/api/sessions/${id}/transcript`)
  },

  getDeliveries(id) {
    return req(`/api/sessions/${id}/deliveries`)
  },

  // SSE ilerleme akışı için EventSource döner.
  events(id) {
    return new EventSource(`${BASE}/api/sessions/${id}/events`)
  },
}
