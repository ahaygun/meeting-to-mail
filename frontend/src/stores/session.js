import { defineStore } from 'pinia'
import { api } from '../lib/api'

// Uygulama fazları (UI görünümlerini yönlendirir).
export const Phase = {
  Setup: 'setup', // kayıt öncesi kurulum
  Recording: 'recording', // aktif kayıt
  Processing: 'processing', // sonlandırıldı → boru hattı
  Done: 'done', // özet hazır / gönderildi
  Cancelled: 'cancelled',
  Failed: 'failed',
}

// Backend session durumlarını UI fazına eşle.
function phaseForStatus(status) {
  switch (status) {
    case 'recording':
      return Phase.Recording
    case 'processing':
    case 'transcribing':
    case 'summarizing':
    case 'pending_send':
    case 'sending':
      return Phase.Processing
    case 'sent':
      return Phase.Done
    case 'cancelled':
      return Phase.Cancelled
    case 'failed':
      return Phase.Failed
    default:
      return Phase.Setup
  }
}

export const useSessionStore = defineStore('session', {
  state: () => ({
    id: null,
    phase: Phase.Setup,
    status: 'configuring',
    title: '',
    sendPolicy: 'immediate',
    cancelWindowSeconds: 0,
    progressMessage: '',
    summary: null,
    deliveries: [],
    error: '',
    _es: null, // EventSource
  }),

  actions: {
    async create(config) {
      this.error = ''
      const sess = await api.createSession(config)
      this.id = sess.id
      this.title = sess.title
      this.status = sess.status
      this.sendPolicy = sess.send_policy
      this.cancelWindowSeconds = sess.cancel_window_seconds
      return sess
    },

    async start() {
      await api.start(this.id)
      this.status = 'recording'
      this.phase = Phase.Recording
    },

    // Kaydı sonlandır: SSE dinlemeye başla, boru hattını tetikle.
    async finalize() {
      this.phase = Phase.Processing
      this.status = 'processing'
      this.progressMessage = 'İşleme başlıyor…'
      this.listen()
      await api.finalize(this.id)
    },

    // Var olan bir ses dosyasını yükleyip boru hattını başlat.
    // Dosya, limitleri aşmamak için ~5MB'lık parçalara bölünür; birleştirme kayıpsızdır.
    async uploadFile(file) {
      this.phase = Phase.Processing
      this.status = 'processing'
      const CHUNK = 5 * 1024 * 1024
      let seq = 0
      for (let off = 0; off < file.size; off += CHUNK) {
        const part = file.slice(off, off + CHUNK)
        const pct = Math.min(100, Math.round(((off + part.size) / file.size) * 100))
        this.progressMessage = `Dosya yükleniyor… %${pct}`
        await api.uploadChunk(this.id, seq++, part)
      }
      this.progressMessage = 'İşleme başlıyor…'
      this.listen()
      await api.finalize(this.id)
    },

    // Geçmiş bir oturumu aç: durumunu getir, sonuçları/ilerlemeyi göster.
    async openSession(id) {
      this.reset()
      this.id = id
      const sess = await api.getSession(id)
      this.title = sess.title
      this.status = sess.status
      this.sendPolicy = sess.send_policy
      this.cancelWindowSeconds = sess.cancel_window_seconds
      this.phase = phaseForStatus(sess.status)

      if (this.phase === Phase.Done || this.phase === Phase.Cancelled || this.phase === Phase.Failed) {
        await this.loadResults()
        if (this.phase === Phase.Failed) this.error = sess.error_message || 'Hata'
      } else if (this.phase === Phase.Processing) {
        // Hâlâ işleniyorsa canlı ilerlemeye abone ol.
        this.listen()
      }
    },

    // SSE ilerleme akışına abone ol.
    listen() {
      this.closeStream()
      const es = api.events(this.id)
      this._es = es
      es.addEventListener('progress', (ev) => this._onEvent(ev))
      es.addEventListener('state', (ev) => this._onEvent(ev))
      es.onerror = () => {
        // EventSource otomatik yeniden bağlanır; terminal durumdaysak kapat.
        if ([Phase.Done, Phase.Cancelled, Phase.Failed].includes(this.phase)) {
          this.closeStream()
        }
      }
    },

    async _onEvent(ev) {
      let data = {}
      try {
        data = JSON.parse(ev.data)
      } catch {
        return
      }
      if (data.status) {
        this.status = data.status
        this.phase = phaseForStatus(data.status)
      }
      if (data.message) this.progressMessage = data.message

      if (this.phase === Phase.Done) {
        await this.loadResults()
        this.closeStream()
      }
      if (this.phase === Phase.Failed) {
        this.error = data.message || 'İşleme sırasında hata oluştu.'
        this.closeStream()
      }
    },

    async loadResults() {
      try {
        this.summary = await api.getSummary(this.id)
      } catch {
        /* özet henüz yoksa yoksay */
      }
      try {
        this.deliveries = await api.getDeliveries(this.id)
      } catch {
        this.deliveries = []
      }
    },

    async cancelSend() {
      await api.cancel(this.id)
      this.status = 'cancelled'
      this.phase = Phase.Cancelled
      this.closeStream()
    },

    closeStream() {
      if (this._es) {
        this._es.close()
        this._es = null
      }
    },

    reset() {
      this.closeStream()
      this.$reset()
    },
  },
})
