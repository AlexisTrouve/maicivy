// Mock jest GLOBAL de next/navigation.
//
// POURQUOI : en test, useParams()/usePathname()/useRouter() retournent undefined → le code qui fait
// `params.locale` ou `pathname.startsWith(...)` plante ("Cannot read properties of undefined/null").
// On fournit des valeurs neutres (locale fr, pathname "/"). Un test peut toujours surcharger via un
// jest.mock('next/navigation', …) inline si besoin d'un cas précis.

export function useRouter() {
  return {
    push: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
  };
}

export function usePathname() {
  return '/';
}

export function useParams() {
  return { locale: 'fr' };
}

export function useSearchParams() {
  return new URLSearchParams();
}

export function redirect() {}
export function notFound() {}
export const permanentRedirect = () => {};
