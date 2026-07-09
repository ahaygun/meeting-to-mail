import { describe, it, expect } from 'vitest'
import { parseRecipients, isValidEmail } from './emails'

describe('parseRecipients', () => {
  it('tek e-postayı ayrıştırır', () => {
    expect(parseRecipients('ali@ornek.com')).toEqual([{ name: '', email: 'ali@ornek.com' }])
  })

  it('virgül/noktalı virgül/satır ile çokluyu böler', () => {
    const out = parseRecipients('a@x.com, b@y.com; c@z.com\nd@w.com')
    expect(out.map((r) => r.email)).toEqual(['a@x.com', 'b@y.com', 'c@z.com', 'd@w.com'])
  })

  it('"Ad <email>" biçiminden ismi yakalar', () => {
    expect(parseRecipients('Ali Yılmaz <ali@ornek.com>')).toEqual([
      { name: 'Ali Yılmaz', email: 'ali@ornek.com' },
    ])
  })

  it('"Ad (email)" biçimini de destekler', () => {
    expect(parseRecipients('Ayşe (ayse@ornek.com)')).toEqual([
      { name: 'Ayşe', email: 'ayse@ornek.com' },
    ])
  })

  it('geçersiz parçaları atar', () => {
    expect(parseRecipients('geçersiz, , b@y.com')).toEqual([{ name: '', email: 'b@y.com' }])
  })

  it('boş girdi için boş dizi döner', () => {
    expect(parseRecipients('')).toEqual([])
    expect(parseRecipients(null)).toEqual([])
  })
})

describe('isValidEmail', () => {
  it('geçerli adresleri kabul eder', () => {
    expect(isValidEmail('ali@ornek.com')).toBe(true)
    expect(isValidEmail(' ayse.yilmaz@firma.co ')).toBe(true)
  })

  it('geçersiz adresleri reddeder', () => {
    for (const bad of ['ali', 'ali@', '@ornek.com', 'ali@ornek', 'a b@c.com', '']) {
      expect(isValidEmail(bad)).toBe(false)
    }
  })
})
