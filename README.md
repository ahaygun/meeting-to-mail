# meeting-to-mail

> Toplantı Kayıt & Özet Otomasyonu

Toplantı sesini kaydeder → parça parça yükler → metne döker (ASR) → yapılandırılmış özet çıkarır (LLM) → gönderim politikasına göre e-posta yollar. Detaylı mimari ve alınan kararlar: [`PROJE_PLANI.md`](PROJE_PLANI.md).

> **Portföy / gösteri projesi.** Uçtan uca çalışır; **ses→metin ve özet tamamen yerel/offline** (whisper.cpp + Ollama) doğrulanmıştır. Mail katmanı anahtar eklenene kadar stub'tır. Kimlik doğrulama yoktur (tek-kurum/iç kullanım varsayımı).

**Ek özellikler:**
- 📁 **Dosya yükleme** — canlı kayda ek olarak var olan ses dosyasını (mp3/m4a/wav/webm…) yükleyip özetleme. Dosya istemcide ~5MB parçalara bölünür (kayıpsız birleşir).
- 🕘 **Geçmiş** — eski oturumları listeleme ve özet/gönderim detaylarını tekrar açma.
- 👥 **Kayıtlı alıcılar** — girilen mailler kişi rehberine kaydedilir; sonraki toplantıda seçmeli chip arayüzünden seçilir/silinir.

## Mimari

- **backend/** — Go + chi. HTTP API + arka plan worker (transcribe → summarize → send). Postgres (pgx). SSE ile canlı ilerleme. Ses parçaları yerel diske.
- **frontend/** — Vue 3 + Vite + Pinia + Tailwind (mobil-first). `useRecorder` composable: MediaRecorder + timeslice + parça yükleme + Wake Lock + sayaç.
- **docker-compose.yml** — Postgres 16 (host portu **5434**).

## Çalıştırma

Üç terminal:

```bash
# 1) Postgres
docker compose up -d db

# 2) Backend (migration'lar otomatik uygulanır)
cd backend && go run ./cmd/server        # :8080

# 3) Frontend
cd frontend && npm install && npm run dev  # :5173 (doluysa 5174)
```

Tarayıcıda dev sunucusunun yazdığı URL'yi aç. Mikrofon izni gerekir; `getUserMedia` yalnızca `localhost` veya HTTPS'te çalışır.

### Telefondan test
Vite `host: true` ile LAN'da yayınlar. Ama telefon tarayıcıları mikrofona **yalnızca HTTPS**'te izin verir — LAN IP'si (http) yetmez. Telefon testi için bir HTTPS tüneli (ör. Cloudflare Tunnel / ngrok) gerekir. Masaüstünde `localhost` sorunsuz çalışır.

## API (özet)

| Metot | Yol | Açıklama |
|---|---|---|
| GET | `/api/sessions` | Geçmiş oturumları listele (en yeni önce) |
| POST | `/api/sessions` | Oturum oluştur (başlık, alıcılar, ayarlar) |
| POST | `/api/sessions/{id}/start` | Kaydı başlat |
| POST | `/api/sessions/{id}/chunks?seq=N` | Ses parçası yükle (raw body) |
| POST | `/api/sessions/{id}/finalize` | Sonlandır → boru hattını tetikle |
| POST | `/api/sessions/{id}/cancel` | Bekleyen gönderimi iptal et (`cancel_window`) |
| GET | `/api/sessions/{id}` | Oturum durumu |
| GET | `/api/sessions/{id}/summary` | Özet |
| GET | `/api/sessions/{id}/transcript` | Transkript |
| GET | `/api/sessions/{id}/deliveries` | Gönderim logu |
| GET | `/api/sessions/{id}/events` | SSE ilerleme akışı |
| GET | `/api/contacts` | Kayıtlı alıcıları listele |
| POST | `/api/contacts` | Kişi ekle/güncelle |
| DELETE | `/api/contacts/{cid}` | Kişiyi sil |

## Gerçek servisler (İterasyon 4)

Sağlayıcılar `internal/providers` altında; **ilgili API anahtarı `.env`'de varsa gerçek servis, yoksa stub** çalışır (`main.go` seçer):

| Katman | Gerçek servis | Yapılandırma | Not |
|---|---|---|---|
| ASR (ses→metin) | **whisper.cpp** (yerel, önerilen) | `WHISPER_MODEL` | Tamamen offline, API'siz, Türkçe. Ses cihazdan çıkmaz. Yerel işleme → 25MB limiti yok. Kurulum: `scripts/setup_whisper.sh` (whisper-cli + ffmpeg + model). |
| ASR (alternatif) | OpenAI **Whisper API** | `OPENAI_API_KEY` | `WHISPER_MODEL` boşsa devreye girer. Uzun ses ffmpeg ile ~20dk parçalara bölünür (<25MB). |
| LLM (özet) | Google **Gemini** | `GOOGLE_API_KEY` | `gemini-2.5-flash` (varsayılan), `responseSchema` ile yapılandırılmış JSON. |
| Mail | **Resend** | `RESEND_API_KEY` | Boşsa stub yalnızca loga yazar. Resend'de doğrulanmış gönderen alan adı gerekir. |

ASR önceliği: `whisper.cpp` (WHISPER_MODEL varsa) → OpenAI Whisper (OPENAI_API_KEY varsa) → stub.

Anahtarları `backend/.env`'e koyup sunucuyu yeniden başlat (`.env.example` şablon). Açılışta log her katmanın gerçek mi stub mı olduğunu yazar.

## Sıradaki adımlar (İterasyon 5)

- `cancel_window` UI'ı ve yeniden özetleme ("kısalt", "kararlara odaklan").
- Hata durumları için UI cilası; ASR/LLM/mail hatalarının kullanıcıya net gösterimi.
