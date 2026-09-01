import en from './en';
import id from './id';

export type Locale = 'en' | 'id';

export const locales: { code: Locale; label: string; native: string }[] = [
  { code: 'en', label: 'English', native: 'English' },
  { code: 'id', label: 'Indonesian', native: 'Bahasa Indonesia' },
];

type NestedDict = { [key: string]: string | NestedDict };

export const messages: Record<Locale, NestedDict> = { en, id };

export function translate(
  locale: Locale,
  key: string,
  params?: Record<string, string | number>,
): string {
  const dict: NestedDict = messages[locale] ?? messages.en;
  const parts = key.split('.');
  let node: NestedDict | string | undefined = dict;
  for (const part of parts) {
    if (typeof node === 'string' || !node) return key;
    node = node[part];
  }
  if (typeof node !== 'string') return key;
  if (params) {
    return Object.entries(params).reduce(
      (acc, [k, v]) => acc.replace(new RegExp(`{${k}}`, 'g'), String(v)),
      node,
    );
  }
  return node;
}
