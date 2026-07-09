<div align="center">

[English](README.md) · **Türkçe**

# meeting·to·mail

### Toplantıyı kaydet → **yerelde** metne dök → yapılandırılmış tutanağa çevir → ilgili kutulara postala.

**Ses cihazdan çıkmadan. Bulut API'sine yüklenmeden. İnternet gerekmeden.**

[![CI](https://github.com/ahaygun/meeting-to-mail/actions/workflows/ci.yml/badge.svg)](https://github.com/ahaygun/meeting-to-mail/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Vue 3](https://img.shields.io/badge/Vue_3-42b883?logo=vuedotjs&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![whisper.cpp](https://img.shields.io/badge/ASR-whisper.cpp-555?logo=openai&logoColor=white)
![Ollama](https://img.shields.io/badge/LLM-Ollama-000?logo=ollama&logoColor=white)
![i18n](https://img.shields.io/badge/i18n-TR%20%2F%20EN-4169E1)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow)

</div>

<div align="center">
  <img src="docs/screenshot-setup.png" alt="Kurulum ekranı — kaynak, başlık, alıcılar, biçim ve gönderim politikası" width="480" />
  <br />
  <sub><em>Kur → başlat → durdur → özet ilgili kutulara düşsün. Üstte <strong>YEREL · OFFLINE İŞLEME</strong> rozeti ve TR/EN dil düğmesi.</em></sub>
</div>

---

## Neden farklı? — Ses cihazdan çıkmaz

Piyasadaki "toplantı → özet" araçlarının neredeyse tamamı sesi bir **bulut API'sine** (OpenAI, Deepgram, AssemblyAI…) yükler; metni de bir bulut LLM'ine gönderir. Yani toplantınızın ham sesi ve dökümü üçüncü taraf sunuculardan geçer.

**meeting·to·mail** bu zincirin **her iki ağır adımını da cihazda** çalıştırır:

| | Tipik bulut araçları | **meeting·to·mail** |
|---|---|---|
| Ses → metin (ASR) | OpenAI / Deepgram / AssemblyAI (bulut) | **whisper.cpp — cihazda** |
| Metin → özet (LLM) | GPT / Gemini / Claude (bulut) | **Ollama — cihazda** |
| Ham ses nereye gider? | 3. taraf sunuculara yüklenir | **hiçbir yere — diskten çıkmaz** |
| İnternet | zorunlu | **gerekmez** (yalnızca mail gönderimi hariç) |
| Gizlilik / KVKK | veri işleyen sözleşmesi gerekir | **veri kurum dışına çıkmaz** |

> **Dışarı çıkan tek şey, zaten postalamak istediğiniz nihai tutanaktır.** Ham ses ve transkript cihazda kalır.
> Bulut sağlayıcılar (OpenAI Whisper, Gemini) yalnızca **isteğe bağlı yedek** olarak desteklenir — `.env`'e anahtar eklemediğiniz sürece hiçbir veri dışarı gitmez.

---

## Öne çıkanlar

- 🔒 **Yerel-öncelik / offline** — ASR (whisper.cpp) ve özet (Ollama) tamamen cihazda; uçtan uca offline doğrulandı.
- 🎙️ **Tarayıcıda kayıt** — `MediaRecorder` + timeslice ile **parça parça** yükleme; çökme olursa yalnızca son parça kaybolur. Ekranı açık tutmak için Wake Lock, canlı sayaç.
- 📁 **Dosya yükleme** — canlı kayıt yerine mevcut ses dosyasını (mp3/m4a/wav/webm…) yükle; istemcide ~5 MB parçalara bölünür, kayıpsız birleşir.
- 🧾 **Yapılandırılmış tutanak** — düz metin değil: **ana maddeler · kararlar · aksiyon maddeleri (sahip + tarih)**. LLM'den `responseSchema` ile JSON olarak alınır, mail gövdesine render edilir.
- 📨 **Gönderim politikası** — `hemen gönder` ya da `iptal pencereli` (belirli süre içinde geri al).
- 🕘 **Geçmiş** — eski oturumları listele, özet/gönderim detaylarını tekrar aç.
- 👥 **Kayıtlı alıcılar** — girilen mailler kişi rehberine kaydedilir; sonraki toplantıda chip arayüzünden seçilir/silinir.
- ⚡ **Canlı ilerleme** — boru hattının her adımı SSE ile arayüze anlık yansır.
- 🌐 **İki dilli (TR/EN)** — tüm arayüz Türkçe/İngilizce; tema düğmesinin yanından anlık geçiş, tercih tarayıcıda saklanır. Bağımlılıksız, ~130 anahtarlık hafif i18n katmanı.
- 📱 **Mobil-first & tema** — responsive tasarım, aydınlık/karanlık mod.

<div align="center">
  <img src="docs/screenshot-recipients.png" alt="Alıcı seçici — kişi rehberi, gruplar ve tek dokunuşla ekleme" width="440" />
  <br />
  <sub><em>Kayıtlı alıcılar & gruplar: girilen mailler rehbere düşer, sonraki toplantıda chip arayüzünden seçilir; grup ile tek dokunuşla eklenir.</em></sub>
</div>

---

## Nasıl çalışır — boru hattı

```
  Tarayıcı                         Go backend (chi)                 Yerel işleme
  ────────                         ────────────────                 ────────────
  🎙️ MediaRecorder ──parçalar──►  birleştir + sakla  ──────────►  🧠 whisper.cpp   (ASR, cihazda)
     (~15–30 sn chunk)             (yerel disk)                          │
                                                                         ▼
                                                                    📝 transkript   (cihazda)
                                                                         │
                                                                         ▼
                                   gönderim politikası  ◄──────────  🤖 Ollama       (özet, cihazda)
                                        │
                                        ▼
                                   📨 e-posta (SMTP/Resend)  ── dışarı çıkan tek adım ─►  alıcılar
        ▲                               │
        └──────────  ⚡ SSE canlı ilerleme  ──────────────────────────────┘

   ═══ cihaz sınırı: ses ve transkript bu çizgiyi geçmez ══════════════════════════
```

Durdur butonuna basılınca boru hattı **elle müdahalesiz** çalışır: birleştir → ASR → özet → gönderim politikasına göre e-posta.

<div align="center">
  <img src="docs/screenshot-pipeline.png" alt="Canlı ilerleme — ses birleştiriliyor, whisper.cpp · yerel, ollama · yerel adımları" width="520" />
  <br />
  <sub><em>Canlı ilerleme (SSE). Adımlarda <strong>whisper.cpp · yerel</strong> ve <strong>ollama · yerel</strong> — ses ve transkript cihazdan çıkmadan işlenir.</em></sub>
</div>

---

## Backend mühendisliği

Durdur butonu yalnızca işi kuyruğa atar; gerisini küçük ve dayanıklı bir iş sistemi yürütür:

- **PostgreSQL tabanlı iş kuyruğu** — işler **`FOR UPDATE SKIP LOCKED` ile atomik** olarak çekilir; böylece birden çok worker aynı işi asla kapmaz (harici broker'a gerek yok).
- **Aşamalı boru hattı** — `transcribe → summarize → send` **birbirini kuyruğa atan ayrı işler** olarak çalışır. Bir adım bağımsızca hata verip incelenebilir; oturum açık bir durum makinesi taşır.
- **Açılışta otomatik migration** (`backend/db/migrations`).
- **SSE ile canlı ilerleme** — worker her geçişi süreç-içi bir hub'a yayınlar, hub bağlı istemcilere dağıtır.
- **Pluggable sağlayıcılar** — ASR / LLM / mail arayüzlerin arkasında; `main.go` env'e göre *gerçek / yerel / stub* seçer ve açılışta hangisinin aktif olduğunu loglar.

---

## Yerel-öncelik mimarisi

Her katman için sağlayıcı `internal/providers` altında bir arayüz; **`.env`'de ilgili anahtar varsa gerçek servis, yoksa yerel ya da stub** seçilir (`main.go` karar verir). Öncelik yerelden buluta doğrudur:

| Katman | 1. tercih (yerel) | 2. tercih (bulut, opsiyonel) | Yoksa |
|---|---|---|---|
| **ASR** (ses→metin) | **whisper.cpp** — `WHISPER_MODEL` | OpenAI Whisper — `OPENAI_API_KEY` | stub |
| **LLM** (özet) | **Ollama** — `OLLAMA_MODEL` | Google Gemini — `GOOGLE_API_KEY` | stub |
| **Mail** | **SMTP** (kurum içi / Mailpit) — `SMTP_HOST` | Resend — `RESEND_API_KEY` | stub (yalnızca loga yazar) |

- **whisper.cpp** — offline, API'siz, Türkçe. Ses cihazdan çıkmaz; yerel işleme olduğu için 25 MB dosya limiti yok. Kurulum: `scripts/setup_whisper.sh` (whisper-cli + ffmpeg + model).
- **Ollama** — offline özet. Kurulum: `brew install ollama && ollama serve && ollama pull qwen2.5:7b`.
- **SMTP** — mail doğası gereği egress'tir; ama SMTP ile veri **3. taraf SaaS yerine kurumun kendi mail sunucusundan** geçer. Geliştirme/offline demo için `docker compose up -d mailpit` (SMTP :1025, gelen kutusu http://localhost:8025) — hiçbir anahtar olmadan mail dahil tüm boru hattı offline çalışır.
- Açılışta log her katmanın **gerçek mi / yerel mi / stub mı** çalıştığını yazar.

---

## Teknoloji yığını

- **Backend** — Go + [chi](https://github.com/go-chi/chi) router · PostgreSQL (pgx) · PostgreSQL tabanlı iş kuyruğu (`FOR UPDATE SKIP LOCKED`) · aşamalı boru hattı + worker · SSE canlı ilerleme · ses parçaları yerel diske.
- **Frontend** — Vue 3 + Vite + Pinia + Tailwind (mobil-first). Merkezde `useRecorder` composable: MediaRecorder + timeslice + parça yükleme + Wake Lock + sayaç. Bağımlılıksız i18n (TR/EN) — `src/i18n.js`.
- **Altyapı** — `docker-compose.yml` ile PostgreSQL 16 (host portu **5434**).

---

## Çalıştırma

Üç terminal:

```bash
# 1) Postgres
docker compose up -d db

# 2) Backend (migration'lar otomatik uygulanır)
cd backend && go run ./cmd/server          # :8080

# 3) Frontend
cd frontend && npm install && npm run dev  # :5173 (doluysa 5174)
```

Tarayıcıda dev sunucusunun yazdığı URL'yi aç. Mikrofon izni gerekir; `getUserMedia` yalnızca **`localhost` veya HTTPS**'te çalışır.

Sağlayıcı anahtarlarını `backend/.env`'e koy (`backend/.env.example` şablon). Hiç anahtar vermeden de akış uçtan uca görülebilir (whisper.cpp + Ollama yerelde, mail stub).

### Telefondan test

Vite `host: true` ile LAN'da yayınlar; ancak telefon tarayıcıları mikrofona **yalnızca HTTPS**'te izin verir — LAN IP'si (http) yetmez. Telefon testi için bir HTTPS tüneli (Cloudflare Tunnel / ngrok) gerekir. Masaüstünde `localhost` sorunsuz çalışır.

> **Not (KVKK):** İnsan sesini kaydetmek rıza gerektirir. Dağıtımda "kayıt başlıyor" bildirimi / onay akışı önerilir. Yerel-öncelik mimarisi bu yükü hafifletir: ham ses ve transkript kurum dışına çıkmaz.

---

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

Detaylı mimari, veri modeli ve alınan kararlar: [`PROJE_PLANI.md`](PROJE_PLANI.md).

---

## Proje durumu

> **Portföy / gösteri projesi.** Uçtan uca çalışır; **ses→metin ve özet tamamen yerel/offline** (whisper.cpp + Ollama) doğrulanmıştır. Kimlik doğrulama yoktur (tek-kurum/iç kullanım varsayımı).

**Sıradaki adımlar:**
- `cancel_window` UI'ı ve yeniden özetleme ("kısalt", "kararlara odaklan").
- ASR/LLM/mail hatalarının kullanıcıya net gösterimi için UI cilası.

---

## Lisans

[MIT](LICENSE)
