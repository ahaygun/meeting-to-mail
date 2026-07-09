// Alıcı e-postalarını ayrıştıran saf yardımcılar.
// Test edilebilirlik için bileşenden ayrı tutulur.

// parseRecipients, serbest metinden alıcıları çıkarır.
// Virgül/noktalı virgül/satır ile ayrılmış çokluyu böler.
// "Ad <email>" veya "Ad (email)" biçimi ismi de yakalar.
// Dönen: [{ name, email }, ...]
export function parseRecipients(text) {
  if (!text) return []
  return text
    .split(/[\n,;]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((part) => {
      const named = part.match(/^(.*?)[<(]\s*([^>)\s]+@[^>)\s]+)\s*[>)]$/)
      if (named) return { name: named[1].trim(), email: named[2].trim() }
      const bare = part.match(/([^\s,;]+@[^\s,;]+)/)
      return bare ? { name: '', email: bare[1].trim() } : null
    })
    .filter(Boolean)
}

// isValidEmail, basit bir e-posta geçerlilik kontrolü (kullanıcı girdisi için yeterli).
export function isValidEmail(s) {
  return typeof s === 'string' && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.trim())
}
