import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Locale } from '@/i18n/messages';
import { translate } from '@/i18n/messages';

const STORAGE_KEY = 'locale';

function getInitialLocale(): Locale {
  if (typeof window === 'undefined') return 'en';
  const saved = window.localStorage.getItem(STORAGE_KEY);
  if (saved === 'en' || saved === 'id') return saved;
  return 'en';
}

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref<Locale>(getInitialLocale());

  const t = computed(() => {
    return (key: string, params?: Record<string, string | number>) =>
      translate(locale.value, key, params);
  });

  function setLocale(value: Locale) {
    locale.value = value;
    window.localStorage.setItem(STORAGE_KEY, value);
  }

  function toggleLocale() {
    setLocale(locale.value === 'en' ? 'id' : 'en');
  }

  return {
    locale,
    t,
    setLocale,
    toggleLocale,
  };
});
