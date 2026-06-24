import { createI18n } from 'vue-i18n'
import en from './locales/en.json'

export type MessageSchema = typeof en

// Auto-discover available locales from the locales folder
const localeModules = import.meta.glob('./locales/*.json', { eager: true, import: 'default' }) as Record<string, MessageSchema>

// Supported locales (only en and pt)
const localeNames: Record<string, { name: string; nativeName: string }> = {
  en: { name: 'English', nativeName: 'English' },
  pt: { name: 'Português (BR)', nativeName: 'Português (BR)' },
}

// Auto-generate SUPPORTED_LOCALES from available files
export const SUPPORTED_LOCALES = Object.keys(localeModules).map(path => {
  const code = path.replace('./locales/', '').replace('.json', '')
  const names = localeNames[code] || { name: code, nativeName: code }
  return { code, ...names }
})

export type SupportedLocale = string

// Build messages object from all locale files
const messages: Record<string, MessageSchema> = {}
for (const path in localeModules) {
  const code = path.replace('./locales/', '').replace('.json', '')
  messages[code] = localeModules[path]
}

// Get saved locale or default to pt
function getDefaultLocale(): string {
  const saved = localStorage.getItem('locale')
  if (saved && messages[saved]) {
    return saved
  }

  const browserLang = navigator.language.split('-')[0]
  if (messages[browserLang]) {
    return browserLang
  }

  return 'pt'
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'en',
  messages,
})

// Helper to change locale
export function setLocale(locale: string) {
  if (!messages[locale]) {
    console.warn(`Locale '${locale}' not available`)
    return
  }
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.setAttribute('lang', locale)
}

// Get current locale
export function getLocale(): string {
  return i18n.global.locale.value
}
