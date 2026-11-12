import { test, expect, Page } from '@playwright/test';
import { readFileSync } from 'fs';
import path from 'path';
import { trackPageErrors } from './helpers/pageErrors';

// ============================================================================
// E2E de la refonte (passe 2) des pages /cv et /gitstats.
// ----------------------------------------------------------------------------
// 1. Aucun libellé FRANÇAIS ne reste visible sur /en : les libellés de ces pages (namespaces
//    `cv` et `gitstats`) ont tous été réécrits et traduits. Le test lit les DEUX catalogues et
//    cherche dans la page chaque libellé français qui diffère de sa traduction anglaise.
// 2. /gitstats ne saute plus au chargement : CLS mesuré < 0,1 (il valait 0,65), avec une API
//    volontairement lente pour que la réserve de chargement soit réellement affichée.
//
// Tourne contre un front local pointé sur l'API de prod (lecture seule, aucun appel IA).
// ============================================================================

const FR = JSON.parse(readFileSync(path.join(__dirname, '../../messages/fr.json'), 'utf-8'));
const EN = JSON.parse(readFileSync(path.join(__dirname, '../../messages/en.json'), 'utf-8'));

// Sous-arbres repris tels quels de l'existant (non réécrits par la refonte) : hors périmètre.
const UNTOUCHED = new Set(['cv.duration', 'cv.skillDetail']);

// Coupe un message ICU en fragments de TEXTE : en-têtes de pluriel (`{n, plural,`), sélecteurs de
// branche (`one {`, `other {`), balises riches (`<unit>`), variables et `#` sont des séparateurs.
// Sans ça, « {count, plural, one {# dépôt} other {# dépôts}} » donnait le fragment « other », mot
// anglais : le test l'aurait cherché — et trouvé — dans la page anglaise. Avec ça, le fragment
// retenu est du vrai texte français (« dépôts »).
function textFragments(message: string): string[] {
  const CUT = '\u0000';
  return message
    .replace(/\{\s*\w+\s*,\s*(?:plural|select|selectordinal)\s*,/g, CUT)
    .replace(/(?:=\d+|zero|one|two|few|many|other)\s*\{/g, CUT)
    .replace(/<\/?\w+>/g, CUT)
    .split(/\{[^{}]*\}|[\u0000{}#]/)
    .map((s) => s.trim());
}

/**
 * Libellés français à traquer : feuilles du namespace qui diffèrent de leur traduction anglaise.
 * Les chaînes à variables ({count}…) sont réduites à leur plus long fragment fixe ; on écarte les
 * fragments de moins de 4 caractères, trop courts pour être un indice fiable.
 */
function frenchLabels(namespace: string): string[] {
  const out: string[] = [];
  const walk = (fr: unknown, en: unknown, at: string) => {
    if (UNTOUCHED.has(at)) return;
    if (typeof fr === 'string') {
      if (fr === en) return;
      const fragment = textFragments(fr).sort((a, b) => b.length - a.length)[0];
      if (fragment && fragment.length >= 4) out.push(fragment);
      return;
    }
    if (fr && typeof fr === 'object') {
      for (const [k, v] of Object.entries(fr as Record<string, unknown>)) {
        walk(v, (en as Record<string, unknown> | undefined)?.[k], `${at}.${k}`);
      }
    }
  };
  walk(FR[namespace], EN[namespace], namespace);
  return out;
}

// Recherche « mot entier » : « Profil » ne doit pas être trouvé dans « Profile ».
function containsWord(haystack: string, needle: string): boolean {
  const escaped = needle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`(?<![\\p{L}\\p{N}])${escaped}(?![\\p{L}\\p{N}])`, 'u').test(haystack);
}

// Texte visible + noms accessibles (aria-label, title) : un libellé lu par un lecteur d'écran
// compte autant qu'un libellé affiché (cf. les tendances de DevPortrait, en français en dur avant).
async function visibleAndAccessibleText(page: Page): Promise<string> {
  return page.evaluate(() => {
    const labels = Array.from(document.querySelectorAll('[aria-label], [title]'))
      .flatMap((el) => [el.getAttribute('aria-label'), el.getAttribute('title')])
      .filter(Boolean);
    return [document.body.innerText, ...labels].join('\n');
  });
}

test.describe('Refonte /cv et /gitstats', () => {
  test('/en/cv : aucun libellé français ne reste', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/en/cv', { waitUntil: 'load' });
    // Le portrait (fetch serveur) et le sélecteur (chunk paresseux) sont arrivés.
    await expect(page.getByRole('heading', { level: 1, name: EN.cv.title })).toBeVisible({ timeout: 20000 });
    await expect(page.getByRole('combobox')).toHaveText(EN.cv.themes.ai.name, { timeout: 15000 });
    await expect(page.getByRole('heading', { name: EN.cv.devPortrait.title })).toBeVisible({ timeout: 20000 });

    const text = await visibleAndAccessibleText(page);
    const leaks = frenchLabels('cv').filter((label) => containsWord(text, label));
    expect(leaks, `libellés français visibles sur /en/cv : ${leaks.join(' | ')}`).toEqual([]);

    // Et les libellés anglais sont bien là (le test ne passe pas sur une page vide).
    expect(text).toContain(EN.cv.sections.experiences);
    expect(text).toContain(EN.cv.export.download);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('/en/gitstats : aucun libellé français ne reste', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/en/gitstats', { waitUntil: 'load' });
    await expect(page.getByTestId('gitstats-kpi-value').first()).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('gitstats-repo').first()).toBeAttached({ timeout: 10000 });

    const text = await visibleAndAccessibleText(page);
    const leaks = frenchLabels('gitstats').filter((label) => containsWord(text, label));
    expect(leaks, `libellés français visibles sur /en/gitstats : ${leaks.join(' | ')}`).toEqual([]);

    expect(text).toContain(EN.gitstats.title);
    expect(text).toContain(EN.gitstats.commits6mo);
    // Titre d'onglet traduit (le gabarit du layout y ajoute « | maicivy »).
    expect(await page.title()).toContain(EN.gitstats.metaTitle);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('/fr/gitstats : CLS < 0,1 pendant le chargement des données', async ({ page }) => {
    // API lente (1,5 s) : la réserve de chargement s'affiche vraiment, puis les données la
    // remplacent. C'est ce remplacement qui faisait sauter la page.
    for (const pattern of ['**/cv/gitstats**', '**/cv/loc**']) {
      await page.route(pattern, async (route) => {
        await new Promise((r) => setTimeout(r, 1500));
        await route.continue();
      });
    }
    // L'observateur est posé AVANT tout script de la page (buffered : rien n'est perdu).
    await page.addInitScript(() => {
      (window as unknown as { __cls: number }).__cls = 0;
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries() as unknown as { value: number; hadRecentInput: boolean }[]) {
          if (!entry.hadRecentInput) (window as unknown as { __cls: number }).__cls += entry.value;
        }
      }).observe({ type: 'layout-shift', buffered: true });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    await expect(page.getByTestId('gitstats-kpi-value').first()).toBeVisible({ timeout: 20000 });
    // Laisse les graphes (ResponsiveContainer) se mesurer et se dessiner.
    await page.waitForTimeout(1500);

    const cls = await page.evaluate(() => (window as unknown as { __cls: number }).__cls);
    console.log(`[gitstats] CLS = ${cls.toFixed(4)}`);
    expect(cls, `CLS mesuré ${cls.toFixed(4)}`).toBeLessThan(0.1);
  });
});
