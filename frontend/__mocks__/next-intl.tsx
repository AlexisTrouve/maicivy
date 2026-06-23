// Mock jest de next-intl.
//
// POURQUOI : next-intl (et use-intl) sont publiés en ESM pur ; next/jest (SWC) ne transforme pas les
// node_modules, donc tout test important un composant i18n explose sur `SyntaxError: Unexpected token
// 'export'`. transformIgnorePatterns ne suffit pas (ignoré par next/jest). On bypass le module réel via
// moduleNameMapper et on résout les VRAIES traductions FR — les tests assertent des strings françaises.
// COMMENT : useTranslations(namespace) retourne un t(key, values) qui lit messages/fr.json à la clé
// `namespace.key`, avec interpolation simple des {var}. (Les ICU plural ne sont pas gérés — aucun test
// composant n'en dépend ; les fiches qui en utilisent sont couvertes en E2E.)
import React from 'react';
import messages from '../messages/fr.json';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resolve(path: string): any {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return path.split('.').reduce<any>((o, k) => (o == null ? undefined : o[k]), messages as any);
}

function makeT(namespace?: string) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (key: string, values?: Record<string, any>) => {
    const full = namespace ? `${namespace}.${key}` : key;
    let s = resolve(full);
    if (s == null) s = key; // clé absente → on rend la clé (comportement de fallback)
    if (typeof s === 'string' && values) {
      s = s.replace(/\{(\w+)\}/g, (_m, k) => (values[k] != null ? String(values[k]) : `{${k}}`));
    }
    return s;
  };
}

export function useTranslations(namespace?: string) {
  return makeT(namespace);
}

// next-intl/server expose getTranslations (async) — même résolution.
export async function getTranslations(arg?: string | { namespace?: string }) {
  const namespace = typeof arg === 'string' ? arg : arg?.namespace;
  return makeT(namespace);
}

export function useLocale() {
  return 'fr';
}

export function useFormatter() {
  return {
    dateTime: (d: Date) => String(d),
    number: (n: number) => String(n),
    relativeTime: (d: unknown) => String(d),
  };
}

export function useMessages() {
  return messages;
}
export function useNow() {
  return new Date(0);
}
export function useTimeZone() {
  return 'UTC';
}

export function NextIntlClientProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
