import { writable, derived } from 'svelte/store';
import en from './locales/en.json' with { type: 'json' };
import id from './locales/id.json' with { type: 'json' };
import { STORAGE_KEYS } from '../constants.js';

const LOCALES = {
  'en-US': { name: 'English (US)', dictionary: en },
  'en': { name: 'English', dictionary: en },
  'id-ID': { name: 'Bahasa Indonesia', dictionary: id },
  'id': { name: 'Bahasa Indonesia', dictionary: id }
};

const DEFAULT_LOCALE = 'en-US';

function getInitialLocale() {
  if (typeof window === 'undefined') return DEFAULT_LOCALE;
  try {
    const saved = localStorage.getItem(STORAGE_KEYS.LOCALE);
    if (saved && LOCALES[saved]) return saved;
    const browserLang = navigator.language;
    if (browserLang && LOCALES[browserLang]) return browserLang;
    const shortLang = browserLang?.split('-')[0];
    if (shortLang && LOCALES[shortLang]) return shortLang;
  } catch (err) {
    console.error('Failed to read saved locale:', err);
  }
  return DEFAULT_LOCALE;
}

export const locale = writable(getInitialLocale());

export function setLocale(newLocale) {
  if (LOCALES[newLocale]) {
    locale.set(newLocale);
    try {
      localStorage.setItem(STORAGE_KEYS.LOCALE, newLocale);
    } catch (err) {
      console.error('Failed to save locale:', err);
    }
  }
}

export function getAvailableLocales() {
  return [
    { code: 'en-US', name: 'English (US)' },
    { code: 'id', name: 'Bahasa Indonesia' }
  ];
}

function resolveNestedKey(obj, keyPath) {
  if (!obj || !keyPath) return undefined;
  const parts = keyPath.split('.');
  let current = obj;
  for (const part of parts) {
    if (current && typeof current === 'object' && part in current) {
      current = current[part];
    } else {
      return undefined;
    }
  }
  return typeof current === 'string' ? current : undefined;
}

function interpolate(template, params) {
  if (!template || !params) return template;
  return template.replace(/\{(\w+)\}/g, (match, token) => {
    return token in params ? String(params[token]) : match;
  });
}

export function translate(activeLocale, key, params = {}) {
  const activeDict = LOCALES[activeLocale]?.dictionary || en;
  const defaultDict = en;

  let resolvedKey = key;

  // Pluralization dispatch: check key_one / key_other if count exists
  if (params && typeof params.count === 'number') {
    const pluralSuffix = params.count === 1 ? '_one' : '_other';
    const candidateKey = `${key}${pluralSuffix}`;
    if (resolveNestedKey(activeDict, candidateKey) || resolveNestedKey(defaultDict, candidateKey)) {
      resolvedKey = candidateKey;
    }
  }

  let text = resolveNestedKey(activeDict, resolvedKey);

  // Fallback to default English locale
  if (text === undefined && activeDict !== defaultDict) {
    text = resolveNestedKey(defaultDict, resolvedKey);
  }

  // Missing in both: return key and warn in dev
  if (text === undefined) {
    if (typeof import.meta !== 'undefined' && import.meta.env?.DEV) {
      console.warn(`[i18n] Missing translation key: "${resolvedKey}" for locale "${activeLocale}"`);
    }
    return resolvedKey;
  }

  return interpolate(text, params);
}

// Svelte-reactive translation store
export const t = derived(locale, ($locale) => {
  return (key, params = {}) => translate($locale, key, params);
});
