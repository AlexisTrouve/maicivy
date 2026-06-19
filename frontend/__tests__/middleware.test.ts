/**
 * Verrouille le fix i18n du middleware (régression du bug « bloqué en FR malgré EN choisi »).
 *
 * QUOI : on vérifie que le choix de langue explicite (cookie NEXT_LOCALE) est respecté sur les
 *   entrées SANS préfixe de locale, et que sinon le défaut FR (délégation à next-intl) reste intact.
 * POURQUOI : avant le fix, la racine `/` redirigeait toujours vers /fr même avec NEXT_LOCALE=en, et
 *   écrasait le cookie → le choix anglais ne « collait » pas en revenant par la racine.
 * COMMENT : on mocke next-intl/middleware et next/server pour isoler NOTRE logique d'aiguillage,
 *   indépendamment de l'environnement (pas de NextRequest réel). Les jest.fn sont créés DANS les
 *   factories (jest.mock est hoisté au-dessus des const → pas de référence à une var en TDZ), puis
 *   récupérés via les imports mockés.
 */

// next-intl/middleware : createMiddleware(config) retourne LA fonction middleware déléguée (intlFn).
jest.mock('next-intl/middleware', () => {
  const intlFn = jest.fn(() => ({ delegated: true }));
  return { __esModule: true, default: jest.fn(() => intlFn) };
});

// next/server : on capture l'URL passée à NextResponse.redirect.
jest.mock('next/server', () => ({
  NextResponse: { redirect: jest.fn((url: URL) => ({ redirectTo: url.pathname })) },
}));

import middleware from '@/middleware';
import createMiddleware from 'next-intl/middleware';
import { NextResponse } from 'next/server';

// intlFn = la valeur retournée par createMiddleware lors du chargement de middleware.ts.
const intlFn = (createMiddleware as jest.Mock).mock.results[0].value as jest.Mock;
const redirectMock = NextResponse.redirect as unknown as jest.Mock;

// Fausse requête : juste ce que le middleware lit (nextUrl + cookies), sans dépendre de NextRequest.
function makeRequest(pathname: string, cookie?: string) {
  const base = new URL(`https://maicivy.test${pathname}`);
  return {
    nextUrl: Object.assign(base, { clone: () => new URL(base.toString()) }),
    cookies: { get: (_name: string) => (cookie ? { value: cookie } : undefined) },
  } as any;
}

describe('middleware i18n — le choix de langue (cookie) colle sur entrée sans préfixe', () => {
  // La config jest (clearMocks/restoreMocks) réinitialise les mocks avant chaque test → on ré-établit
  // les implémentations ici (sinon intlFn/redirect retourneraient undefined après reset).
  beforeEach(() => {
    intlFn.mockReturnValue({ delegated: true });
    redirectMock.mockImplementation((url: URL) => ({ redirectTo: url.pathname }));
  });

  it('racine "/" + cookie en → redirige vers /en (le choix colle)', () => {
    const res = middleware(makeRequest('/', 'en'));
    expect(redirectMock).toHaveBeenCalledTimes(1);
    expect(redirectMock.mock.calls[0][0].pathname).toBe('/en');
    expect(res).toEqual({ redirectTo: '/en' });
    expect(intlFn).not.toHaveBeenCalled();
  });

  it('racine "/" SANS cookie → délègue à next-intl (défaut FR, comportement inchangé)', () => {
    const res = middleware(makeRequest('/'));
    expect(redirectMock).not.toHaveBeenCalled();
    expect(intlFn).toHaveBeenCalledTimes(1);
    expect(res).toEqual({ delegated: true });
  });

  it('"/cv" + cookie de → redirige vers /de/cv (préserve le sous-chemin, pas de double slash)', () => {
    middleware(makeRequest('/cv', 'de'));
    expect(redirectMock.mock.calls[0][0].pathname).toBe('/de/cv');
  });

  it('URL déjà préfixée "/en" + cookie fr → délègue (l\'URL explicite prime sur le cookie)', () => {
    middleware(makeRequest('/en', 'fr'));
    expect(redirectMock).not.toHaveBeenCalled();
    expect(intlFn).toHaveBeenCalledTimes(1);
  });

  it('cookie invalide "xx" → ignoré, délègue (défaut FR)', () => {
    middleware(makeRequest('/', 'xx'));
    expect(redirectMock).not.toHaveBeenCalled();
    expect(intlFn).toHaveBeenCalledTimes(1);
  });
});
