// Page projet du portfolio : seul un 404 du backend mène à « introuvable ».
//
// POURQUOI : panne du 2026-09-27 — maiProFiles saturé, backend en 502, et le front affichait « 404 »
// sur des projets qui existent (toute erreur devenait notFound()). Pour un crawler, un 404 fait
// sortir la page de l'index ; une erreur serveur est vue comme temporaire. Ce test verrouille la
// distinction : 404 → notFound() ; toute autre erreur → relancée vers la frontière d'erreur.

const notFound = jest.fn(() => {
  throw new Error('NEXT_NOT_FOUND');
});
jest.mock('next/navigation', () => ({ notFound: () => notFound() }));

const getPage = jest.fn();
jest.mock('@/lib/portfolio-api', () => ({
  portfolioApi: { getPage: (...a: unknown[]) => getPage(...a) },
  portfolioImageUrl: (u: string) => u,
}));

jest.mock('next-intl/server', () => ({
  getTranslations: async () => Object.assign((k: string) => k, { has: () => false }),
}));

import PortfolioProjectPage from '../page';

// Next 15 : les pages reçoivent params/searchParams en Promise — les tests les passent pareil.
const params = Promise.resolve({ locale: 'fr', id: 'theomen' });

describe('PortfolioProjectPage — panne amont vs projet absent', () => {
  beforeEach(() => {
    notFound.mockClear();
    getPage.mockReset();
  });

  it('projet absent de la curation (404 backend) → notFound()', async () => {
    getPage.mockRejectedValue({ success: false, statusCode: 404 });

    await expect(PortfolioProjectPage({ params })).rejects.toThrow('NEXT_NOT_FOUND');
    expect(notFound).toHaveBeenCalledTimes(1);
  });

  it('maiProFiles en panne (502 backend) → erreur relancée, PAS de notFound()', async () => {
    const panne = { success: false, statusCode: 502, message: 'Erreur 502' };
    getPage.mockRejectedValue(panne);

    await expect(PortfolioProjectPage({ params })).rejects.toBe(panne);
    expect(notFound).not.toHaveBeenCalled();
  });

  it('erreur réseau sans statut (timeout) → relancée, PAS de notFound()', async () => {
    getPage.mockRejectedValue(new Error('aborted'));

    await expect(PortfolioProjectPage({ params })).rejects.toThrow('aborted');
    expect(notFound).not.toHaveBeenCalled();
  });
});
