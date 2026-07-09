import { ref, computed } from 'vue'
import { api } from '../lib/api'

// Desteklenen ilk MIME tipini seç (tarayıcıya göre değişir).
function pickMimeType() {
  const candidates = [
    'audio/webm;codecs=opus',
    'audio/webm',
    'audio/mp4', // Safari
    'audio/ogg;codecs=opus',
  ]
  for (const t of candidates) {
    if (window.MediaRecorder && MediaRecorder.isTypeSupported(t)) return t
  }
  return '' // tarayıcı varsayılanı
}

// useRecorder: MediaRecorder'ı timeslice ile çalıştırır, her parçayı yükler,
// Wake Lock dener, canlı sayaç tutar. sessionId reaktif olmalı.
export function useRecorder(getSessionId) {
  const recording = ref(false)
  const elapsed = ref(0) // saniye
  const uploaded = ref(0) // yüklenen parça sayısı
  const pending = ref(0) // yükleme kuyruğundaki parça
  const errorMsg = ref('')

  let mediaRecorder = null
  let stream = null
  let wakeLock = null
  let timer = null
  let seq = 0
  const uploadQueue = []
  let draining = false

  const CHUNK_MS = 15000 // ~15 sn'lik parçalar

  const elapsedLabel = computed(() => {
    const s = elapsed.value
    const mm = String(Math.floor(s / 60)).padStart(2, '0')
    const ss = String(s % 60).padStart(2, '0')
    return `${mm}:${ss}`
  })

  async function requestWakeLock() {
    try {
      if ('wakeLock' in navigator) {
        wakeLock = await navigator.wakeLock.request('screen')
        wakeLock.addEventListener?.('release', () => {})
      }
    } catch {
      // Wake Lock reddedilebilir; kritik değil.
    }
  }

  function releaseWakeLock() {
    try {
      wakeLock?.release?.()
    } catch {
      /* yoksay */
    }
    wakeLock = null
  }

  // Sayfa tekrar öne gelince Wake Lock'ı geri al.
  function handleVisibility() {
    if (document.visibilityState === 'visible' && recording.value) {
      requestWakeLock()
    }
  }

  async function drainQueue() {
    if (draining) return
    draining = true
    while (uploadQueue.length > 0) {
      const item = uploadQueue.shift()
      pending.value = uploadQueue.length + 1
      try {
        await api.uploadChunk(getSessionId(), item.seq, item.blob)
        uploaded.value++
      } catch (e) {
        // Yükleme başarısızsa yeniden kuyruğa al ve kısa bekle.
        uploadQueue.unshift(item)
        errorMsg.value = `Parça yüklenemedi, yeniden denenecek: ${e.message}`
        await new Promise((r) => setTimeout(r, 2000))
      }
    }
    pending.value = 0
    draining = false
  }

  function enqueueChunk(blob) {
    if (!blob || blob.size === 0) return
    uploadQueue.push({ seq: seq++, blob })
    drainQueue()
  }

  async function start() {
    errorMsg.value = ''
    seq = 0
    uploaded.value = 0
    elapsed.value = 0

    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const mimeType = pickMimeType()
    mediaRecorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined)

    mediaRecorder.ondataavailable = (e) => enqueueChunk(e.data)
    mediaRecorder.start(CHUNK_MS)

    recording.value = true
    await requestWakeLock()
    document.addEventListener('visibilitychange', handleVisibility)

    timer = setInterval(() => {
      elapsed.value++
    }, 1000)
  }

  // Kaydı durdurur ve son parçalar yüklenene kadar bekler.
  async function stop() {
    if (!mediaRecorder) return
    clearInterval(timer)
    timer = null

    await new Promise((resolve) => {
      mediaRecorder.onstop = resolve
      mediaRecorder.stop()
    })

    recording.value = false
    stream?.getTracks().forEach((t) => t.stop())
    stream = null
    releaseWakeLock()
    document.removeEventListener('visibilitychange', handleVisibility)

    // Kuyrukta kalan tüm parçaların yüklenmesini bekle.
    await drainQueue()
  }

  return {
    recording,
    elapsed,
    elapsedLabel,
    uploaded,
    pending,
    errorMsg,
    start,
    stop,
  }
}
