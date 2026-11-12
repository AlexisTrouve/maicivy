// QUOI : la page /dossier-vivant suit sa bascule de publication (lib/publication.ts) — balise
// robots ET sitemap, lus à leur vraie source (generateMetadata de la page, app/sitemap.ts).
// POURQUOI les deux ensemble : un noindex sans retrait du sitemap (ou l'inverse) est le désaccord
// que ce module existe pour empêcher. Le test suit le drapeau : l'ouvrir ne le casse pas, il vérifie
// alors l'état publié.
import sitemap from '../sitemap';
import { generateMetadata } from '../[locale]/dossier-vivant/page';
import { PUBLICATION } from '@/lib/publication';

// La page tire des modules de rendu (client, next/font, next/headers) sans rapport avec ses
// métadonnées : neutralisés pour lire generateMetadata seule.
jest.mock('@/components/dossier-vivant/DossierVivant', () => ({ DossierVivant: () => null }));
jest.mock('@/lib/dossier-vivant/data', () => ({}));
jest.mock('@/lib/montagne-api', () => ({}));
jest.mock('@/lib/montagne/fonts', () => ({ montagneSerif: { variable: '' } }));
jest.mock('next/headers', () => ({ cookies: jest.fn() }));

const published = PUBLICATION['dossier-vivant'];

beforeEach(() => {
  // Le sitemap interroge l'API (blog, portfolio) : réponse vide, seules les routes statiques restent.
  global.fetch = jest.fn().mockResolvedValue({ ok: false }) as unknown as typeof fetch;
});

describe('publication de /dossier-vivant', () => {
  it('la page déclare robots noindex/nofollow tant qu’elle n’est pas publiée', async () => {
    for (const locale of ['fr', 'en', 'zh']) {
      const meta = await generateMetadata({
        params: Promise.resolve({ locale }),
        searchParams: Promise.resolve({}),
      });
      expect(meta.robots).toEqual(published ? undefined : { index: false, follow: false });
    }
  });

  it('le sitemap ne la déclare dans aucune langue tant qu’elle n’est pas publiée', async () => {
    const urls = (await sitemap()).map((e) => e.url);
    // Garde-fou : le sitemap a bien été produit (sinon « absente » serait vrai pour rien).
    expect(urls.some((u) => u.endsWith('/fr/cv'))).toBe(true);
    expect(urls.filter((u) => u.includes('/dossier-vivant'))).toHaveLength(published ? 5 : 0);
  });
});
