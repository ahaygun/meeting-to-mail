# Toplantı Kayıt & Özet Otomasyonu — Proje Planı

> Çalışma adı; kesinleşmedi.
> Son güncelleme: 7 Temmuz 2026

## 1. Genel Bakış

Belirli bir toplantı odasında yapılan toplantıların sesini kaydeden, kaydı metne döken, metinden yapılandırılmış bir özet çıkaran ve bu özeti seçilen alıcılara e-posta ile gönderen bir web uygulaması. Alıcılar ve ayarlar her oturumda, kayıt başlamadan önce belirlenir. Kayıt durdurulunca boru hattı elle müdahale olmadan (otomatik) çalışır.

Hedefler:

- Mobil uyumlu (telefondan da kullanılabilir), responsive web arayüzü.
- AI-first: hem geliştirmede AI (Claude Code vb.) hem de ürünün merkezinde AI (transkript + özet).
- Sade bir "kur → başlat → durdur → özet ilgili kutulara düşsün" deneyimi.

## 2. Temel Akış

Kayıt (tarayıcıda mikrofon) → parça parça yükleme → backend'de birleştirme → metne dökme (ASR) → LLM ile özet → gönderim politikasına göre e-posta.

Kayıt öncesi kurulum ekranı şunları toplar: toplantı başlığı (mail konusu olur), alıcılar, opsiyonel katılımcı isimleri, özet stili, gönderim politikası. Ardından tek büyük **Başlat** butonu.

## 3. Alınan Kararlar

| Konu | Karar |
|---|---|
| İşleme modeli | Kaydet-sonra-işle (batch). Gerçek-zamanlı transkript 2. iterasyona bırakıldı. |
| Kayıt cihazı | İkisi de: odada sabit cihaz + kişisel telefonlar. |
| Backend | Go (chi router). |
| Frontend | Vue 3 (Vite + Pinia + Tailwind), mobil-first. |
| Konfigürasyon | Kayıt başlamadan önce girilir (alıcılar + ayarlar). |
| Durdurunca | Boru hattı otomatik, elle müdahalesiz çalışır. |
| Gönderim politikası | Ayarlanabilir. Varsayılan: hemen gönder (`immediate`). Seçenek: iptal pencereli (`cancel_window`). |

## 4. Mimari

### Boru hattı

MediaRecorder ses parçalarını (~15–30 sn'lik chunk'lar) Go backend'e POST eder → backend parçaları saklar ve birleştirir → transkript işi kuyruğa girer → ASR → transkript → LLM özet → gönderim politikasına göre e-posta. İlerleme SSE ile canlı olarak frontend'e yansır.

### Bileşenler

- **Frontend (Vue):** merkezde bir `useRecorder` composable — MediaRecorder'ı timeslice ile çalıştırır, her parçayı yükler, Wake Lock dener, büyük aktivasyon butonu + canlı sayaç gösterir. Sonlandırınca SSE ilerleme ekranı → (opsiyonel) düzenlenebilir özet/transkript → alıcı seçici.
- **Backend (Go):** HTTP API + async worker. Endpoint'ler: session oluştur, parça yükle, sonlandır, durum, transkript+özet getir, iptal. Uzun süren işler için PostgreSQL'de `jobs` tablosu + goroutine worker (ileride Redis/asynq'e taşınabilir).
- **Depolama:** ses parçaları yerel diske (MinIO/S3'e geçmeye hazır), metaveri Postgres'te.
- **Sağlayıcılar (arayüz + stub):** ASR, LLM özet, mailer. Önce stub, sonra gerçek servisler.

### Durum makinesi

```
configuring → recording → processing → transcribing → summarizing
                                          ├─ immediate:     → sending → sent
                                          └─ cancel_window: → pending_send → sending → sent
                                                                    └─(iptal)→ cancelled
   herhangi bir aşamada hata → failed
```

`immediate`'te `send` işi hemen kuyruğa girer; `cancel_window`'da `run_at = ended_at + pencere`. İptal, bekleyen `send` işini `cancelled`'a çevirmekten ibarettir — ekstra makine yok.

## 5. Veri Modeli

Tam DDL: `db/migrations/0001_init.sql`. Özet:

- **sessions** — kayıt öncesi konfig + durum. Anahtar alanlar: `title`, `status`, `summary_style`, `send_policy`, `cancel_window_seconds`, zaman damgaları.
- **session_recipients** — oturumun alıcı e-postaları (ileride kayıtlı gruplardan beslenebilir).
- **session_participants** — opsiyonel katılımcı isimleri (özette konuşmacı atfı için).
- **audio_chunks** — parça parça gelen ses (`seq`, `storage_path`, `size_bytes`; `(session_id, seq)` unique).
- **transcripts** — ASR çıktısı (`provider`, `language`, `text`).
- **summaries** — birden çok satır olabilir (yeniden özetleme için). `content_json` (yapılandırılmış) + `content_text` (mail gövdesi).
- **jobs** — async iş kuyruğu (`type`: transcribe/summarize/send, `status`, `run_at` gönderim penceresini de yönetir).
- **email_deliveries** — gönderim logu (alıcı bazında durum).

## 6. Kritik Kısıtlar & Riskler

**Telefonda kayıt güvenilirliği.** iOS Safari, tarayıcı arka plana ya da kilit ekranına geçince ses kaydını askıya alır ve bu web'de tam çözülemez. Azaltıcılar: Wake Lock ile ekranı açık tutmak, net bir "önde ve ekran açık tut" UX'i, ve parça parça yükleme (çökme olursa yalnızca son parça kaybolur). Pratik kural: uzun toplantılar için güvenilir yol odadaki sabit cihaz; telefon yalnızca önde + ekran açıkken güvenilir. Ürün bu beklentiyle kurulmalı.

**Uzun ses.** 1–2 saatlik toplantı büyük dosya demek; bazı ASR'lerin dosya limiti var (ör. Whisper API'de 25MB). Parçalama (chunking) ve maliyet baştan planlanmalı.

**KVKK / rıza.** İnsan sesini kaydetmek rıza gerektirir. Dağıtımda "kayıt başlıyor" bildirimi / onay akışı düşünülmeli.

## 7. AI-First Katman

- **Yapılandırılmış özet:** düz metin değil — ana başlık, ana maddeler, kararlar, aksiyon maddeleri (sahip + tarih). `summaries.content_json` bunu tutar; `content_text` mail için render edilir.
- **Yeniden özetleme:** "kısalt", "kararlara odaklan" gibi komutlarla aynı transkriptten yeni özet üretmek (çoklu `summaries` satırı).
- **ASR seçenekleri (Türkçe):** Whisper, Deepgram, AssemblyAI, ya da doğrudan ses alan Gemini. Konuşmacı ayrımı (diarization) isteniyorsa Deepgram/AssemblyAI güçlü. Güncel model/fiyat, uygulama aşamasında teyit edilecek (bilgi tabanı yılbaşı itibarıyla).
- **Geliştirme yaklaşımı:** boru hattı önce stub'larla uçtan uca kurulur, sonra gerçek AI servisleri takılır — böylece her katman izole test edilebilir.

## 8. Teknoloji Yığını

- **Backend:** Go + chi. Hedef persistence: Postgres/pgx. MVP iskeleti bellek içi store ile anında çalışır (DB kurmadan akış görülür). Async: goroutine worker. İlerleme: SSE. Ses: yerel disk (MinIO-ready).
- **Frontend:** Vue 3 + Vite + Pinia + Tailwind, mobil-first responsive. Kayıt: MediaRecorder + Wake Lock.
- **Sağlayıcılar:** ASR (TBD, Türkçe), LLM özet (TBD), mail (Resend / SendGrid / SES).

## 9. Proje Yapısı (backend)

```
backend/
  cmd/server/main.go
  internal/
    config/      env + ayarlar
    domain/      tipler + status sabitleri
    store/       repo arayüzü (MVP: bellek içi → sonra pgx)
    httpapi/     router, sessions, uploads, events (SSE)
    worker/      iş kuyruğu: transcribe → summarize → send
    storage/     ses parçalarını diske yaz + birleştir
    providers/   asr / summarize / mailer arayüzleri + stub
  db/migrations/0001_init.sql
  .env.example   Makefile
```

## 10. İterasyon Planı

1. **Backend vertical slice:** session oluştur + parça yükle + sonlandır + parça birleştirme (ASR/mail stub).
2. **Frontend kayıt:** aktivasyon butonu + parça parça kayıt/yükleme + Wake Lock + sayaç.
3. **Uçtan uca bağlama:** sonlandır → sahte transkript → sahte özet → SSE ilerleme → ekranda göster.
4. **Gerçek servisler:** sırayla gerçek ASR → gerçek LLM özet → gerçek mail.
5. **Cilalama:** gönderim politikası UI'ı (`immediate` / `cancel_window`), yeniden özetleme, hata durumları, kalıcılık (Postgres'e geçiş).

## 11. Açık Sorular / Sonraki Kararlar

- **Kimlik doğrulama:** iç kullanım (tek kurum) mu, çok kiracılı mı? Login gerekli mi? (Henüz konuşulmadı.)
- **ASR sağlayıcısı:** Türkçe kalite + diarization + uzun ses ihtiyacına göre seçim.
- **LLM:** özet için model seçimi.
- **Mail sağlayıcısı:** Resend / SendGrid / SES arası tercih.
- **Alıcılar:** her seferinde elle mi girilecek, yoksa kayıtlı gruplar mı olacak?
- **Özet stilleri:** kaç seçenek olacak (kararlar/aksiyon, tam tutanak, kısa özet...)?
- **Kalıcılık kapsamı:** transkript ve ses ne kadar süre saklanacak? (KVKK ile doğrudan ilişkili.)
