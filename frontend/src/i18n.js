// Bağımlılıksız, hafif i18n. Modül düzeyinde reaktif `locale` (localStorage'a kalıcı);
// `t(key, params)` çeviri + {param} enterpolasyonu yapar. Template'lerde t() çağrısı
// locale.value'yu okuduğu için dil değişince otomatik yeniden render olur.
import { ref } from 'vue'

const STORAGE_KEY = 'm2m-lang'
const SUPPORTED = ['tr', 'en']

function initialLocale() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (SUPPORTED.includes(saved)) return saved
    const nav = (navigator.language || '').slice(0, 2)
    if (SUPPORTED.includes(nav)) return nav
  } catch {
    /* yoksay */
  }
  return 'tr'
}

export const locale = ref(initialLocale())

export function setLocale(l) {
  if (!SUPPORTED.includes(l)) return
  locale.value = l
  try {
    localStorage.setItem(STORAGE_KEY, l)
  } catch {
    /* yoksay */
  }
  document.documentElement.setAttribute('lang', l)
}

export function toggleLocale() {
  setLocale(locale.value === 'tr' ? 'en' : 'tr')
}

// İlk açılışta <html lang> ayarla.
try {
  document.documentElement.setAttribute('lang', locale.value)
} catch {
  /* yoksay */
}

const messages = {
  tr: {
    // App shell
    'app.new': 'Yeni',
    'app.archive': 'Arşiv',
    'app.localOffline': 'Yerel · Offline işleme',
    'app.footerLeft': 'whisper.cpp · ollama · postgres',
    'app.footerRight': 'batch · yerel işleme',
    'app.themeLight': 'Açık tema',
    'app.themeDark': 'Karanlık tema',
    'app.langTitle': 'Dil · Language',

    // Setup
    'setup.title': 'Toplantı Oluştur',
    'setup.subtitle': 'Kaynağı, alıcıları ve biçimi belirle — sonra kaydı başlat ya da dosya yükle.',
    'setup.source': 'Kaynak',
    'setup.liveRecord': 'Canlı Kayıt',
    'setup.liveRecordHint': 'Mikrofondan kaydet',
    'setup.uploadFile': 'Dosya Yükle',
    'setup.uploadFileHint': 'Var olan ses dosyası',
    'setup.filePlaceholder': 'mp3 · m4a · wav · webm seç…',
    'setup.titleLabel': 'Başlık',
    'setup.titlePlaceholder': 'ör. Haftalık Saha Değerlendirmesi',
    'setup.titleHint': 'E-posta konusu olarak kullanılır.',
    'setup.recipients': 'Alıcılar',
    'setup.participantsFormat': 'Katılımcılar & Biçim',
    'setup.participantsPlaceholder': 'Katılımcı isimleri (opsiyonel): Ali, Ayşe, Mehmet',
    'setup.style.decisions_actions': 'Kararlar & Aksiyonlar',
    'setup.style.full_minutes': 'Tam Tutanak',
    'setup.style.short': 'Kısa Özet',
    'setup.sendPolicy': 'Gönderim Politikası',
    'setup.immediate': 'Hemen gönder',
    'setup.immediateHint': 'Özet hazır olunca yolla',
    'setup.cancelWindow': 'İptal pencereli',
    'setup.cancelWindowHint': 'Göndermeden önce bekle',
    'setup.window': 'Pencere:',
    'setup.minutes': 'dakika',
    'setup.processing': 'İşleniyor…',
    'setup.startRecord': '● Kaydı Başlat',
    'setup.uploadSummarize': '↑ Yükle & Özetle',
    'setup.err.title': 'Toplantı başlığı gerekli.',
    'setup.err.recipients': 'En az bir alıcı seçin veya ekleyin.',
    'setup.err.file': 'Bir ses dosyası seçin.',
    'setup.err.generic': 'İşlem başlatılamadı.',

    // Recording
    'rec.recording': 'Kayıt sürüyor',
    'rec.uploaded': 'Yüklenen parça: {n}',
    'rec.queued': 'sırada {n}',
    'rec.phoneHint': '📱 Telefonda: ekranı açık ve uygulamayı önde tutun — arka plana geçince kayıt durabilir.',
    'rec.micError': 'Mikrofona erişilemedi. Tarayıcı izinlerini kontrol edin. ({e})',
    'rec.finalizeError': 'Sonlandırma başarısız.',
    'rec.stop': '■ Durdur & Özetle',
    'rec.stopping': 'Sonlandırılıyor…',

    // Progress
    'prog.processingLocal': 'İşleniyor · yerel',
    'prog.error': 'Bir hata oluştu',
    'prog.step.processing.label': 'Ses birleştiriliyor',
    'prog.step.processing.hint': 'parçalar tek dosyaya',
    'prog.step.transcribing.label': 'Metne dökülüyor',
    'prog.step.transcribing.hint': 'whisper.cpp · yerel',
    'prog.step.summarizing.label': 'Özet çıkarılıyor',
    'prog.step.summarizing.hint': 'ollama · yerel',
    'prog.step.pending_send.label': 'Gönderim bekliyor',
    'prog.step.pending_send.hint': 'iptal penceresi',
    'prog.step.sending.label': 'E-posta gönderiliyor',
    'prog.step.sending.hint': 'alıcılara',
    'prog.step.sent.label': 'Gönderildi',
    'prog.step.sent.hint': 'tamamlandı',

    // Summary
    'sum.sentBanner': 'Toplantı özeti alıcılara gönderildi.',
    'sum.cancelledBanner': 'Gönderim iptal edildi — özet kaydedildi, e-posta yollanmadı.',
    'sum.docLabel': 'Tutanak · meeting-to-mail',
    'sum.sent': 'Sevk edildi',
    'sum.draft': 'Taslak',
    'sum.keyPoints': 'Ana Maddeler',
    'sum.decisions': 'Kararlar',
    'sum.actionItems': 'Aksiyon Maddeleri',
    'sum.loading': 'Özet yükleniyor…',
    'sum.delivery': 'İletim',
    'sum.statusSent': 'Gönderildi',
    'sum.statusFailed': 'Başarısız',
    'sum.newMeeting': 'Yeni Tutanak',

    // History
    'hist.pastMinutes': 'Geçmiş tutanaklar',
    'hist.refresh': '↻ Yenile',
    'hist.loading': 'Yükleniyor…',
    'hist.loadError': 'Arşiv yüklenemedi.',
    'hist.empty': 'Henüz kayıt yok. İlk tutanağını oluştur.',
    'hist.recipientCount': '{n} alıcı',

    // Durum etiketleri (History rozetleri)
    'status.configuring': 'Kuruluyor',
    'status.recording': 'Kaydediliyor',
    'status.processing': 'İşleniyor',
    'status.transcribing': 'Metne dökülüyor',
    'status.summarizing': 'Özetleniyor',
    'status.pending_send': 'Gönderim bekliyor',
    'status.sending': 'Gönderiliyor',
    'status.sent': 'Gönderildi',
    'status.cancelled': 'İptal edildi',
    'status.failed': 'Hata',

    // RecipientPicker
    'rp.editRecipients': 'Alıcıları düzenle · {n} seçili',
    'rp.selectOrAdd': 'Alıcı seç veya ekle',
    'rp.recipients': 'Alıcılar',
    'rp.selected': '{n} seçili',
    'rp.addEmail': 'E-posta ekle',
    'rp.emailPlaceholder': 'ornek@firma.com',
    'rp.add': 'Ekle',
    'rp.pasteHint': 'Virgülle çoklu yapıştırabilirsin. "Ad <e-posta>" formatı ismi de kaydeder.',
    'rp.groupsHint': 'Gruplar — tek dokunuşla ekle',
    'rp.deleteGroup': 'Grubu sil',
    'rp.contacts': 'Kişiler',
    'rp.clearSelection': 'Seçimi temizle',
    'rp.searchContacts': 'Kişilerde ara…',
    'rp.editName': 'İsim düzenle',
    'rp.namePlaceholder': 'Ad',
    'rp.deleteContact': 'Rehberden sil',
    'rp.noMatch': 'Eşleşen kişi yok.',
    'rp.noContacts': 'Kayıtlı kişi yok — yukarıdan ekle.',
    'rp.groupNamePlaceholder': 'Grup adı — ör. Yönetim Kurulu',
    'rp.save': 'Kaydet',
    'rp.cancel': 'İptal',
    'rp.saveGroup': '＋ Grup kaydet',
    'rp.done': 'Tamam · {n} alıcı',
    'rp.err.validEmail': 'Geçerli bir e-posta gir.',
    'rp.err.selectFirst': 'Önce alıcı seç, sonra grup olarak kaydet.',
    'rp.err.groupName': 'Grup adı gerekli.',
    'rp.err.groupSave': 'Grup kaydedilemedi.',
    'rp.confirmDelGroup': '"{name}" grubunu sil? (Kişiler silinmez, sadece grup.)',
    'rp.confirmDelContact': '{who} kişisini rehberden sil?',

    // Store (ilerleme mesajları)
    'store.processingStart': 'İşleme başlıyor…',
    'store.uploading': 'Dosya yükleniyor… %{pct}',
    'store.errGeneric': 'Hata',
    'store.errProcessing': 'İşleme sırasında hata oluştu.',
  },

  en: {
    // App shell
    'app.new': 'New',
    'app.archive': 'Archive',
    'app.localOffline': 'Local · Offline processing',
    'app.footerLeft': 'whisper.cpp · ollama · postgres',
    'app.footerRight': 'batch · local processing',
    'app.themeLight': 'Light theme',
    'app.themeDark': 'Dark theme',
    'app.langTitle': 'Language · Dil',

    // Setup
    'setup.title': 'Create Meeting',
    'setup.subtitle': 'Choose the source, recipients and format — then start recording or upload a file.',
    'setup.source': 'Source',
    'setup.liveRecord': 'Live Recording',
    'setup.liveRecordHint': 'Record from microphone',
    'setup.uploadFile': 'Upload File',
    'setup.uploadFileHint': 'Existing audio file',
    'setup.filePlaceholder': 'choose mp3 · m4a · wav · webm…',
    'setup.titleLabel': 'Title',
    'setup.titlePlaceholder': 'e.g. Weekly Field Review',
    'setup.titleHint': 'Used as the email subject.',
    'setup.recipients': 'Recipients',
    'setup.participantsFormat': 'Participants & Format',
    'setup.participantsPlaceholder': 'Participant names (optional): Alice, Bob, Carol',
    'setup.style.decisions_actions': 'Decisions & Actions',
    'setup.style.full_minutes': 'Full Minutes',
    'setup.style.short': 'Short Summary',
    'setup.sendPolicy': 'Send Policy',
    'setup.immediate': 'Send immediately',
    'setup.immediateHint': 'Send once the summary is ready',
    'setup.cancelWindow': 'Cancel window',
    'setup.cancelWindowHint': 'Wait before sending',
    'setup.window': 'Window:',
    'setup.minutes': 'minutes',
    'setup.processing': 'Processing…',
    'setup.startRecord': '● Start Recording',
    'setup.uploadSummarize': '↑ Upload & Summarize',
    'setup.err.title': 'Meeting title is required.',
    'setup.err.recipients': 'Select or add at least one recipient.',
    'setup.err.file': 'Choose an audio file.',
    'setup.err.generic': 'Could not start.',

    // Recording
    'rec.recording': 'Recording',
    'rec.uploaded': 'Uploaded chunks: {n}',
    'rec.queued': '{n} queued',
    'rec.phoneHint': '📱 On phone: keep the screen on and the app in the foreground — recording may pause in the background.',
    'rec.micError': 'Could not access the microphone. Check browser permissions. ({e})',
    'rec.finalizeError': 'Finalizing failed.',
    'rec.stop': '■ Stop & Summarize',
    'rec.stopping': 'Finalizing…',

    // Progress
    'prog.processingLocal': 'Processing · local',
    'prog.error': 'An error occurred',
    'prog.step.processing.label': 'Merging audio',
    'prog.step.processing.hint': 'chunks into one file',
    'prog.step.transcribing.label': 'Transcribing',
    'prog.step.transcribing.hint': 'whisper.cpp · local',
    'prog.step.summarizing.label': 'Summarizing',
    'prog.step.summarizing.hint': 'ollama · local',
    'prog.step.pending_send.label': 'Awaiting send',
    'prog.step.pending_send.hint': 'cancel window',
    'prog.step.sending.label': 'Sending email',
    'prog.step.sending.hint': 'to recipients',
    'prog.step.sent.label': 'Sent',
    'prog.step.sent.hint': 'completed',

    // Summary
    'sum.sentBanner': 'Meeting summary sent to recipients.',
    'sum.cancelledBanner': 'Send cancelled — summary saved, no email sent.',
    'sum.docLabel': 'Minutes · meeting-to-mail',
    'sum.sent': 'Sent',
    'sum.draft': 'Draft',
    'sum.keyPoints': 'Key Points',
    'sum.decisions': 'Decisions',
    'sum.actionItems': 'Action Items',
    'sum.loading': 'Loading summary…',
    'sum.delivery': 'Delivery',
    'sum.statusSent': 'Sent',
    'sum.statusFailed': 'Failed',
    'sum.newMeeting': 'New Meeting',

    // History
    'hist.pastMinutes': 'Past minutes',
    'hist.refresh': '↻ Refresh',
    'hist.loading': 'Loading…',
    'hist.loadError': 'Could not load archive.',
    'hist.empty': 'No records yet. Create your first minutes.',
    'hist.recipientCount': '{n} recipients',

    // Status labels (History badges)
    'status.configuring': 'Configuring',
    'status.recording': 'Recording',
    'status.processing': 'Processing',
    'status.transcribing': 'Transcribing',
    'status.summarizing': 'Summarizing',
    'status.pending_send': 'Awaiting send',
    'status.sending': 'Sending',
    'status.sent': 'Sent',
    'status.cancelled': 'Cancelled',
    'status.failed': 'Failed',

    // RecipientPicker
    'rp.editRecipients': 'Edit recipients · {n} selected',
    'rp.selectOrAdd': 'Select or add recipients',
    'rp.recipients': 'Recipients',
    'rp.selected': '{n} selected',
    'rp.addEmail': 'Add email',
    'rp.emailPlaceholder': 'name@company.com',
    'rp.add': 'Add',
    'rp.pasteHint': 'Paste multiple separated by commas. The "Name <email>" format saves the name too.',
    'rp.groupsHint': 'Groups — add with one tap',
    'rp.deleteGroup': 'Delete group',
    'rp.contacts': 'Contacts',
    'rp.clearSelection': 'Clear selection',
    'rp.searchContacts': 'Search contacts…',
    'rp.editName': 'Edit name',
    'rp.namePlaceholder': 'Name',
    'rp.deleteContact': 'Delete from contacts',
    'rp.noMatch': 'No matching contacts.',
    'rp.noContacts': 'No saved contacts — add above.',
    'rp.groupNamePlaceholder': 'Group name — e.g. Board',
    'rp.save': 'Save',
    'rp.cancel': 'Cancel',
    'rp.saveGroup': '＋ Save group',
    'rp.done': 'Done · {n} recipients',
    'rp.err.validEmail': 'Enter a valid email.',
    'rp.err.selectFirst': 'Select recipients first, then save as a group.',
    'rp.err.groupName': 'Group name is required.',
    'rp.err.groupSave': 'Could not save group.',
    'rp.confirmDelGroup': 'Delete group "{name}"? (Contacts are kept, only the group is removed.)',
    'rp.confirmDelContact': 'Delete {who} from contacts?',

    // Store (progress messages)
    'store.processingStart': 'Processing starting…',
    'store.uploading': 'Uploading file… {pct}%',
    'store.errGeneric': 'Error',
    'store.errProcessing': 'An error occurred during processing.',
  },
}

// t(key, params?) — çeviri + {param} enterpolasyonu. Eksik anahtar TR'ye, o da yoksa key'e düşer.
export function t(key, params) {
  const dict = messages[locale.value] || messages.tr
  let str = dict[key]
  if (str == null) str = messages.tr[key]
  if (str == null) return key
  if (params) {
    for (const k in params) {
      str = str.split('{' + k + '}').join(String(params[k]))
    }
  }
  return str
}

export function useI18n() {
  return { t, locale, setLocale, toggleLocale }
}
