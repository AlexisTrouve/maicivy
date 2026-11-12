// Mock jest de `next-intl/navigation`. POURQUOI : ce sous-module est de l'ESM pur que next/jest ne
// transforme pas → toute suite qui importe (même transitivement) `@/i18n/navigation` plantait avec
// "Unexpected token 'export'". On stub `createNavigation` pour que le VRAI `@/i18n/navigation` se charge
// et expose un Link/usePathname/useRouter/redirect neutres et testables.
import React from 'react';

// Reproduit la forme de l'objet renvoyé par le vrai createNavigation de next-intl.
export function createNavigation() {
  return {
    // Link → simple <a href>. href peut être un objet {pathname} côté next-intl → on dégrade en '#'.
    Link: ({ href, children, ...props }: { href: unknown; children?: React.ReactNode; [k: string]: unknown }) =>
      React.createElement('a', { href: typeof href === 'string' ? href : '#', ...props }, children as React.ReactNode),
    usePathname: () => '/',
    useRouter: () => ({
      push: () => {},
      replace: () => {},
      back: () => {},
      forward: () => {},
      refresh: () => {},
      prefetch: () => {},
    }),
    redirect: () => {},
    permanentRedirect: () => {},
    getPathname: () => '/',
  };
}
