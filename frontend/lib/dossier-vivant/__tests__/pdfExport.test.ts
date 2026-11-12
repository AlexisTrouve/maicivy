/**
 * pdfExport.ts — ce que le navigateur envoie encore au backend pour le dossier.
 *  - le PDF : RIEN d'imprimable — une langue et, pour un run réel, l'id de la copie serveur ; une
 *    réponse non-PDF devient une erreur NOMMÉE (dossier expiré ≠ panne) ;
 *  - les sections que lit la route des compétences de l'écran : cinq, dans l'ordre, le schéma en
 *    géométrie A4, des liens absolus, aucun chiffre de preuve.
 */
import { foldEvents } from '../runState';
import { DossierPdfError, fetchDossierPdf, pdfFileName, pdfSections, PRINT_FIT } from '../pdfExport';
import { tesseraScenario } from '../tessera';

function tesseraSections() {
  const s = tesseraScenario({ projectCount: 75, siteUrl: '/fr', melodicodeVideoUrl: '/portfolio-images/melodicode.mp4' });
  return pdfSections(foldEvents(s.events), (d) => new URL(d.url, 'https://maicivy.etheryale.com').href);
}

describe('pdfSections', () => {
  const sections = tesseraSections();

  it('les cinq sections, dans l’ordre de lecture, avec leurs titres', () => {
    expect(sections.map((s) => s.id)).toEqual(['need', 'architecture', 'proofs', 'contribution', 'next']);
    expect(sections.every((s) => s.title.length > 0 && s.blocks.length > 0)).toBe(true);
  });

  it('le schéma part en géométrie mise en page pour la largeur A4', () => {
    const diagram = sections[1].blocks.find((b) => b.type === 'diagram');
    if (diagram?.type !== 'diagram') throw new Error('pas de schéma');
    const d = diagram.diagram;
    expect(d.boxes).toHaveLength(5);
    expect(d.edges).toHaveLength(5);
    expect(d.steps.map((s) => s.n).sort()).toEqual([1, 2, 3]);
    expect((d.width + 2 * d.margin) * d.scale).toBeLessThanOrEqual(PRINT_FIT.maxWidth);
  });

  it('preuves : identifiant, raison et lien ABSOLU — aucun chiffre', () => {
    const proofs = sections[2].blocks.filter((b) => b.type === 'proof');
    expect(proofs.length).toBeGreaterThan(0);
    for (const p of proofs) {
      expect(p).not.toHaveProperty('loc');
      expect(p).not.toHaveProperty('tests');
    }
    const demos = proofs.flatMap((p) => (p.type === 'proof' && p.demo ? [p.demo.url] : []));
    expect(demos.length).toBeGreaterThan(0);
    for (const u of demos) expect(u).toMatch(/^https:\/\//);
  });
});

describe('fetchDossierPdf', () => {
  const realFetch = global.fetch;
  afterEach(() => {
    global.fetch = realFetch;
  });

  function respond(status: number, contentType: string, body: string) {
    const fn = jest.fn().mockResolvedValue(new Response(body, { status, headers: { 'Content-Type': contentType } }));
    global.fetch = fn as unknown as typeof fetch;
    return fn;
  }

  it('n’envoie que la langue et l’id de la copie serveur', async () => {
    const fn = respond(200, 'application/pdf', '%PDF-1.4');
    const blob = await fetchDossierPdf('/api/v1/dossier-vivant/run/pdf', { lang: 'fr', dossierId: 'ab'.repeat(16) });
    expect(blob.size).toBeGreaterThan(0);
    expect(JSON.parse(fn.mock.calls[0][1].body)).toEqual({ lang: 'fr', dossierId: 'ab'.repeat(16) });
  });

  it('dossier inconnu ou expiré : erreur nommée DOSSIER_NOT_FOUND', async () => {
    respond(404, 'application/json', JSON.stringify({ code: 'DOSSIER_NOT_FOUND', error: 'x' }));
    await expect(fetchDossierPdf('/x', { lang: 'fr' })).rejects.toEqual(new DossierPdfError('DOSSIER_NOT_FOUND'));
  });

  it('réponse illisible : erreur nommée par son statut, jamais un faux fichier', async () => {
    respond(502, 'text/html', '<html>');
    await expect(fetchDossierPdf('/x', { lang: 'fr' })).rejects.toMatchObject({ code: 'HTTP_502' });
    respond(200, 'text/html', '<html>');
    await expect(fetchDossierPdf('/x', { lang: 'fr' })).rejects.toMatchObject({ code: 'HTTP_200' });
  });
});

describe('pdfFileName', () => {
  it('lettres, chiffres et tirets — accents retirés, écritures non latines gardées', () => {
    expect(pdfFileName('dossier de conception Cabinet Lumen')).toBe('dossier-de-conception-cabinet-lumen.pdf');
    expect(pdfFileName('Konzeptdossier Société Générale')).toBe('konzeptdossier-societe-generale.pdf');
    expect(pdfFileName('设计方案书 天元')).toBe('设计方案书-天元.pdf');
    expect(pdfFileName('***')).toBe('dossier.pdf');
  });
});
