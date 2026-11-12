import {
  computeTargetSize,
  extractImageFiles,
  isSupportedImage,
  stripDataUrlPrefix,
  MAX_EDGE,
} from '../chat-images';

// QUOI : règles de préparation des captures avant envoi au modèle.
//
// POURQUOI ces règles : une capture 4K coûte des tokens de vision pour des pixels que le modèle
// n'exploite pas, mais réduire trop rendrait illisible le petit texte d'un formulaire — or c'est
// précisément ce que l'agent doit lire. Le plafond est donc la résolution utile du modèle, et une
// capture qui tient déjà dessous n'est PAS ré-encodée (aucun flou ajouté).

describe('isSupportedImage', () => {
  it('accepte les quatre types que l’API autorise', () => {
    for (const t of ['image/png', 'image/jpeg', 'image/gif', 'image/webp']) {
      expect(isSupportedImage(t)).toBe(true);
    }
  });

  it('refuse le reste', () => {
    for (const t of ['image/bmp', 'image/svg+xml', 'application/pdf', 'text/plain', '']) {
      expect(isSupportedImage(t)).toBe(false);
    }
  });
});

describe('computeTargetSize', () => {
  // Une capture 1440p (2560×1440) tient sous le plafond : elle doit passer INTACTE, sans
  // ré-encodage — c'est le cas le plus courant et celui où la netteté compte le plus.
  it('laisse intacte une image qui tient déjà', () => {
    expect(computeTargetSize(2560, 1440)).toEqual({ width: 2560, height: 1440, resized: false });
  });

  it('réduit une capture 4K en préservant le ratio', () => {
    const r = computeTargetSize(3840, 2160);
    expect(r.resized).toBe(true);
    expect(r.width).toBe(MAX_EDGE);
    // 3840/2160 = 16/9 → la hauteur doit suivre le même rapport.
    expect(r.height).toBe(Math.round((MAX_EDGE * 2160) / 3840));
  });

  it('plafonne le grand côté même en portrait', () => {
    const r = computeTargetSize(1200, 5000);
    expect(r.height).toBe(MAX_EDGE);
    expect(r.width).toBeLessThan(1200);
  });

  it('ne divise pas par zéro sur une image dégénérée', () => {
    expect(computeTargetSize(0, 0)).toEqual({ width: 0, height: 0, resized: false });
  });
});

describe('stripDataUrlPrefix', () => {
  // L'API veut du base64 NU : laisser le préfixe produirait une image invalide côté modèle.
  it('retire l’en-tête data:', () => {
    expect(stripDataUrlPrefix('data:image/png;base64,AAAB')).toBe('AAAB');
  });

  it('laisse passer une chaîne déjà nue', () => {
    expect(stripDataUrlPrefix('AAAB')).toBe('AAAB');
  });
});

describe('extractImageFiles', () => {
  const fakeTransfer = (items: Array<{ kind: string; type: string }>) =>
    ({
      items: items.map((i) => ({ ...i, getAsFile: () => new File(['x'], 'c.png', { type: i.type }) })),
    }) as unknown as DataTransfer;

  // Le collage et le glisser-déposer passent par la MÊME structure : une seule extraction pour les
  // deux gestes garantit qu'ils se comportent pareil.
  it('récupère les images d’un presse-papier ou d’un drop', () => {
    const files = extractImageFiles(fakeTransfer([{ kind: 'file', type: 'image/png' }]));
    expect(files).toHaveLength(1);
  });

  it('ignore le texte — un collage de texte doit suivre son cours normal', () => {
    const files = extractImageFiles(fakeTransfer([{ kind: 'string', type: 'text/plain' }]));
    expect(files).toHaveLength(0);
  });

  it('ignore un fichier d’un type non supporté', () => {
    const files = extractImageFiles(fakeTransfer([{ kind: 'file', type: 'application/pdf' }]));
    expect(files).toHaveLength(0);
  });

  it('tolère l’absence de presse-papier', () => {
    expect(extractImageFiles(null)).toEqual([]);
  });
});
