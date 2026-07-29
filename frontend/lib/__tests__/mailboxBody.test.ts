import { splitMailboxBody, shortLinkLabel } from '../mailboxBody';

describe('splitMailboxBody', () => {
  it('renvoie un seul segment texte quand il n\'y a aucun lien', () => {
    const segs = splitMailboxBody('Bonjour Alexis, ceci est un mail sans lien.');
    expect(segs).toEqual([{ type: 'text', value: 'Bonjour Alexis, ceci est un mail sans lien.' }]);
  });

  it('isole une URL entre parenthèses en son propre segment "link"', () => {
    const segs = splitMailboxBody('Malt (https://exemple.com/tracking/abc123) vous informe.');
    expect(segs).toEqual([
      { type: 'text', value: 'Malt (' },
      { type: 'link', value: 'https://exemple.com/tracking/abc123' },
      { type: 'text', value: ') vous informe.' },
    ]);
  });

  it('gère plusieurs liens dans le même texte', () => {
    const segs = splitMailboxBody('Voir A (https://a.com/x) et B (https://b.com/y) pour plus.');
    const links = segs.filter((s) => s.type === 'link').map((s) => s.value);
    expect(links).toEqual(['https://a.com/x', 'https://b.com/y']);
  });

  it('une URL très longue (tracking HubSpot) reste un seul segment link, pas coupée', () => {
    const longUrl = 'https://d2Z2Gr04.eu1.hubspotlinks.com/Ctc/W3+113/' + 'W'.repeat(300);
    const segs = splitMailboxBody(`Lien (${longUrl})`);
    const linkSeg = segs.find((s) => s.type === 'link');
    expect(linkSeg?.value).toBe(longUrl);
  });

  it('body vide → aucun segment', () => {
    expect(splitMailboxBody('')).toEqual([]);
  });

  it('une URL en toute fin de texte (pas de texte après) ne produit pas de segment vide', () => {
    const segs = splitMailboxBody('Cliquez ici: https://exemple.com/fin');
    expect(segs).toEqual([
      { type: 'text', value: 'Cliquez ici: ' },
      { type: 'link', value: 'https://exemple.com/fin' },
    ]);
  });
});

describe('shortLinkLabel', () => {
  it('extrait le hostname d\'une URL de tracking énorme', () => {
    const longUrl = 'https://d2Z2Gr04.eu1.hubspotlinks.com/Ctc/W3+113/' + 'W'.repeat(300);
    expect(shortLinkLabel(longUrl)).toBe('d2z2gr04.eu1.hubspotlinks.com');
  });

  it('retire le préfixe www.', () => {
    expect(shortLinkLabel('https://www.malt.fr/mission/123')).toBe('malt.fr');
  });

  it('URL invalide → repli "lien" plutôt que planter', () => {
    expect(shortLinkLabel('pas-une-url')).toBe('lien');
  });
});
