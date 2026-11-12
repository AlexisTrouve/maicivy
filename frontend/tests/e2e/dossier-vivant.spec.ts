import { readFileSync } from 'fs';
import path from 'path';

import { test, expect, type APIRequestContext, type Locator, type Page } from '@playwright/test';

import { cors, mockLiveRun, recordedRun, sse, streamLiveRun } from './helpers/dossierVivant';
import { trackDuplicateKeys } from './helpers/reactKeys';

/**
 * Dossier vivant — le parcours réel sur /[locale]/dossier-vivant, par de VRAIS clics.
 *
 * VISITEUR, RUN RÉEL (le défaut) : sans aucun cookie, le visiteur écrit son besoin et « Lancer
 * l'équipe » part sur la route PUBLIQUE du run. Deux niveaux :
 *  - contre le backend du BANC (`cmd/dossier-live`, DOSSIER_LIVE_FAKE_AGENTS=1 : agents simulés
 *    par sa fausse passerelle, tout le reste en production) — flux SSE réel, dossier, PDF réel ;
 *  - flux simulé AU RÉSEAU (page.route) pour les cas que le banc ne produit pas (arrêt sur
 *    sortie inutilisable, flux brut d'un vrai run enregistré).
 *
 * DÉMO ENREGISTRÉE (rejeu Tessera, « Voir la démo » dans la barre) :
 *  1. l'ouverture : la voix, le besoin d'exemple (Tessera) pré-rempli dans le champ ;
 *  2. clic sur « Voir la démo » → le champ se replie en tuile du visiteur, l'équipe entre :
 *     l'orchestrateur et les quatre sous-agents sont visibles, et se mettent au travail ;
 *  3. des cartes et tuiles passent à l'état ALLUMÉ (`data-state="lit"`) et un filet d'or se trace
 *     vers chacune depuis la tuile du visiteur ;
 *  4. le dossier final : cinq sections écrites, le tableau des preuves chiffré avec les chiffres
 *     RÉELS des fiches (GET /api/v1/projects), le bouton « Télécharger le dossier » ;
 *  5. la section « Compétences mobilisées », composée par le backend réel à partir des preuves.
 * Plus : pause qui fige l'écran, et l'interface traduite en anglais (le contenu rejoué reste en FR).
 * PDF : « Télécharger le dossier » remet un VRAI fichier PDF (événement `download` du navigateur),
 * rendu par le backend réel — route publique du rejeu, ou route publique du run réel. Le navigateur
 * n'y envoie QUE la langue et, pour un run réel, l'id de la copie serveur du dossier : le backend
 * imprime sa copie, jamais un dossier venu de la page.
 *
 * Données : la montagne et les fiches viennent de l'endpoint RÉEL — le test exige un backend
 * (MONTAGNE_API_URL, défaut http://localhost:8080 ; le serveur Next doit lire le même backend via
 * API_URL). Le seul contenu fixé ici est celui du scénario rejoué (les projets qu'il retient).
 * Horloge du rejeu accélérée (`?speed=2`) : même parcours, deux fois plus court.
 *
 * Lancer : `npx playwright test tests/e2e/dossier-vivant.spec.ts --project=chromium`
 * (version PC : viewport 1920×1080 imposé ci-dessous ; le téléphone : dossier-vivant-mobile.spec.ts).
 * Clés React : un parcours entier (rejeu Tessera, puis le flux brut d'un VRAI run) ne doit laisser
 * aucun avertissement « same key » en console (helpers/reactKeys.ts).
 * LA PAGE DÉFILE : une fois le dossier remis (rejeu ET run réel simulé), il se lit en faisant défiler
 * la PAGE à la molette — plus d'ascenseur interne — jusqu'au pied de page et ses quatre liens ; en
 * grand écran et en fenêtre de téléphone (390 × 844).
 * LA FLEUR : un clic (ou Entrée) sur une tuile, une carte, un projet retenu ou une ligne de preuve
 * ouvre la fleur du BON projet (composants de /montagne) ; le second clic mène au portfolio. Ouvrir
 * la fleur ne met en pause ni le rejeu ni un run réel (flux servi EN CONTINU, streamLiveRun).
 */

const API = process.env.MONTAGNE_API_URL || 'http://localhost:8080';
const FR = JSON.parse(readFileSync(path.join(__dirname, '../../messages/fr.json'), 'utf-8'));
const EN = JSON.parse(readFileSync(path.join(__dirname, '../../messages/en.json'), 'utf-8'));
/** Projets que le scénario Tessera retient (lib/dossier-vivant/tessera.ts). */
const KEPT = ['melodicode', 'openclaw_platform', 'envoia', 'momos', 'claude_orchestrator', 'maicivy', 'civjdr_roman', 'demiurgos_game_master', 'anthropic_vendor_proxy'];
const PROOFS = ['melodicode', 'openclaw_platform', 'envoia', 'maicivy'];
const SECTIONS = ['need', 'architecture', 'proofs', 'contribution', 'next'];

test.use({ viewport: { width: 1920, height: 1080 } });

async function realFacts(request: APIRequestContext): Promise<Map<string, { loc: number; testCount: number }>> {
  const res = await request.get(`${API}/api/v1/projects?lang=fr`);
  expect(res.ok()).toBeTruthy();
  const body = (await res.json()) as { projects: { sourceId: string; loc: number; testCount: number }[] };
  return new Map(body.projects.map((p) => [p.sourceId, p]));
}

test.describe('dossier vivant', () => {
  test.setTimeout(120_000);

  test('voir la démo → agents au travail → projets allumés → dossier en cinq sections', async ({ page, request }) => {
    const facts = await realFacts(request);
    await page.goto('/fr/dossier-vivant?speed=2');

    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-phase', 'intro');
    await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });

    // Montagne réelle : autant de cartes + tuiles que de projets servis par l'API.
    const montagne = await (await request.get(`${API}/api/v1/montagne?lang=fr`)).json();
    await expect(page.getByTestId('dv-card').or(page.getByTestId('dv-tile'))).toHaveCount(montagne.nodes.length);

    // 1. L'action principale lance le run réel (bloc suivant) ; la démo est à un clic, dans la barre.
    await expect(page.getByTestId('dv-launch')).toHaveText(FR.dossierVivant.hero.launch);
    const demo = page.getByTestId('dv-pause');
    await expect(demo).toHaveText(FR.dossierVivant.bar.demo);
    await demo.click();
    await expect(page.getByTestId('dv-caption')).toHaveText(FR.dossierVivant.bar.caption);
    await expect(root).toHaveAttribute('data-phase', 'run');
    await expect(root).toHaveAttribute('data-run', 'running');

    // 2. Le visiteur devient une tuile ; l'équipe : orchestrateur + quatre colonnes au travail.
    await expect(page.getByTestId('dv-visitor-tile')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId('dv-visitor-tile')).toContainText('Inès Morel');
    await expect(page.getByTestId('dv-orchestrator')).toBeVisible();
    await expect(page.getByTestId('dv-orchestrator')).toContainText(FR.dossierVivant.team.agents.orchestrator.name);
    const columns = page.getByTestId('dv-agent-column');
    await expect(columns).toHaveCount(4);
    for (const agent of ['reader', 'researcher', 'architect', 'writer']) {
      await expect(page.locator(`[data-testid="dv-agent-column"][data-agent="${agent}"]`)).toHaveAttribute('data-status', /working|done/, { timeout: 20_000 });
    }
    await expect(page.getByTestId('dv-thread-message').first()).toBeVisible();
    await expect(page.getByTestId('dv-cursor').first()).toBeVisible();

    // 3. Des projets passent à l'état allumé, avec leur filet depuis la tuile du visiteur.
    const melodicode = page.locator('[data-project-id="melodicode"]').first();
    await expect(melodicode).toHaveAttribute('data-state', 'lit', { timeout: 20_000 });
    await expect(page.locator('[data-testid="dv-thread"][data-to="melodicode"]')).toHaveCount(1);
    await expect(page.locator('[data-testid="dv-tile"][data-project-id="liveconf"]')).toHaveAttribute('data-state', 'dropped', { timeout: 20_000 });

    // 4. Le dossier : cinq sections écrites, preuves chiffrées, bouton de téléchargement.
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
    for (const id of KEPT) await expect(page.locator(`[data-project-id="${id}"]:not([data-testid="dv-proof-row"])`)).toHaveAttribute('data-state', 'lit');
    await expect(page.getByTestId('dv-thread')).toHaveCount(KEPT.length);

    const dossier = page.getByTestId('dv-dossier');
    await expect(dossier).toHaveAttribute('data-ready', '');
    const sections = page.getByTestId('dv-section');
    await expect(sections).toHaveCount(5);
    for (const id of SECTIONS) await expect(page.locator(`[data-testid="dv-section"][data-section="${id}"]`)).toHaveAttribute('data-status', 'done');

    const rows = page.getByTestId('dv-proof-row');
    await expect(rows).toHaveCount(PROOFS.length);
    for (const id of PROOFS) {
      // Les chiffres affichés sont ceux de la fiche réelle, pas du flux.
      const loc = facts.get(id)!.loc;
      const row = page.locator(`[data-testid="dv-proof-row"][data-project-id="${id}"]`);
      await expect(row).toContainText(new Intl.NumberFormat('fr').format(loc));
    }

    // 5. Compétences mobilisées : composées par le backend RÉEL (POST /dossier-vivant/skills) à
    // partir des preuves — chaque ligne cite ses preuves, toutes parmi celles du tableau.
    const skills = page.getByTestId('dv-skills');
    await expect(skills).toHaveAttribute('data-status', 'ready', { timeout: 20_000 });
    await expect(skills).toContainText(FR.dossierVivant.dossier.skills.title);
    const skillRows = page.getByTestId('dv-skill-row');
    expect(await skillRows.count()).toBeGreaterThan(0);
    for (const sources of await skillRows.evaluateAll((els) => els.map((e) => e.getAttribute('data-sources') ?? ''))) {
      const ids = sources.split(',');
      expect(ids.length).toBeGreaterThan(0);
      for (const id of ids) expect(PROOFS).toContain(id);
    }
    await expect(page.getByTestId('dv-download')).toBeVisible();
    await expect(page.getByTestId('dv-download')).toHaveText(FR.dossierVivant.dossier.download);
    await expect(page.getByTestId('dv-dossier-state')).toHaveText(FR.dossierVivant.dossier.state.ready);
  });

  test('pause fige le parcours, lecture le reprend', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=2');
    await page.getByTestId('dv-pause').click();
    await expect(page.locator('[data-testid="dv-agent-column"][data-agent="reader"]')).toHaveAttribute('data-status', 'working', { timeout: 15_000 });

    await page.getByTestId('dv-pause').click();
    await expect(page.getByTestId('dv-pause')).toHaveText(FR.dossierVivant.bar.play);
    const before = await page.getByTestId('dv-totals').textContent();
    await page.waitForTimeout(1500);
    expect(await page.getByTestId('dv-totals').textContent()).toBe(before);

    await page.getByTestId('dv-pause').click();
    await expect(page.getByTestId('dv-pause')).toHaveText(FR.dossierVivant.bar.pause);
    await expect.poll(async () => page.getByTestId('dv-totals').textContent(), { timeout: 5_000 }).not.toBe(before);
  });

  test('télécharger le dossier remis : un vrai PDF, non vide, schéma lisible', async ({ page }) => {
    // Écran figé à la remise (`?t=`) : le parcours complet est couvert par le premier test.
    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });

    // Le schéma garde des libellés lisibles : jamais sous le plancher d'échelle du panneau.
    const diagram = page.getByTestId('dv-diagram');
    await expect(diagram).toBeVisible();
    expect(Number(await diagram.getAttribute('data-scale'))).toBeGreaterThanOrEqual(0.82);

    const button = page.getByTestId('dv-download');
    await expect(button).toHaveText(FR.dossierVivant.dossier.download);
    // Le corps ne porte que la langue : le serveur imprime le scénario tel qu'il le connaît.
    const asked = page.waitForRequest((r) => r.url().endsWith('/api/v1/dossier-vivant/pdf') && r.method() === 'POST');
    const [download] = await Promise.all([page.waitForEvent('download', { timeout: 60_000 }), button.click()]);
    expect((await asked).postDataJSON()).toEqual({ lang: 'fr' });
    expect(download.suggestedFilename()).toBe('dossier-de-conception-tessera.pdf');
    const file = await download.path();
    const bytes = readFileSync(file!);
    expect(bytes.length).toBeGreaterThan(5_000);
    expect(bytes.subarray(0, 4).toString('latin1')).toBe('%PDF');
    await expect(button).toHaveAttribute('data-pdf', 'idle');
    if (process.env.DV_EVIDENCE_DIR) await download.saveAs(path.join(process.env.DV_EVIDENCE_DIR, download.suggestedFilename()));
  });

  test('un projet dont le dépôt a un icon.png l’affiche — image réellement décodée — et les autres gardent leur monogramme', async ({ page, request }) => {
    // Pré-requis : un backend avec Gitea configuré (GITEA_STATS_TOKEN), dont l'index des icônes est
    // construit — il se construit EN FOND au premier appel, d'où l'attente.
    await expect
      .poll(async () => Object.keys(((await (await request.get(`${API}/api/v1/project-icons`)).json()) as { icons: Record<string, string> }).icons), {
        timeout: 90_000,
        message: 'index des icônes (le backend du banc a-t-il GITEA_STATS_TOKEN ?)',
      })
      .toEqual(expect.arrayContaining(['melodicode', 'maicivy', 'demiurgos_game_master']));

    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });
    /** L'icône dans `scope` : chargée ET décodée (une image cassée a une largeur naturelle nulle). */
    const decoded = async (scope: string) => {
      const img = page.locator(`${scope} [data-testid="dv-project-icon"]`);
      await expect(img).toHaveCount(1);
      await expect.poll(() => img.evaluate((el: HTMLImageElement) => el.complete && el.naturalWidth), { timeout: 10_000 }).toBeGreaterThan(0);
      expect(await img.evaluate((el: HTMLImageElement) => el.src)).toMatch(/\/api\/v1\/project-icons\/[a-z_]+\?v=/);
    };
    // Tableau des preuves : MelodiCode et maicivy ont une icône de dépôt.
    await decoded('[data-testid="dv-proof-row"][data-project-id="melodicode"]');
    await decoded('[data-testid="dv-proof-row"][data-project-id="maicivy"]');
    // Tuile de la montagne (un projet retenu par l'Architecte).
    await decoded('[data-testid="dv-tile"][data-project-id="demiurgos_game_master"]');
    // Sans icon.png : le monogramme, pas d'image.
    const sans = page.locator('[data-testid="dv-proof-row"][data-project-id="openclaw_platform"]');
    await expect(sans.locator('[data-testid="dv-project-icon"]')).toHaveCount(0);
    await expect(sans).toContainText('OP');
  });

  test('noms de la montagne, cartes allumées : aucun chevauchement, deux lignes au plus, aucun mot coupé, lisibles sur leur fond, au-dessus des filets', async ({ page }) => {
    // Écran figé à l'instant « cartes allumées » du rejeu : plusieurs projets retenus, filets tracés.
    await page.goto('/fr/dossier-vivant?t=34');
    await expect(page.getByTestId('dv-visitor-tile')).toBeVisible({ timeout: 30_000 });
    await expect.poll(() => page.locator('[data-testid="dv-tile"][data-state="lit"]').count(), { timeout: 15_000 }).toBeGreaterThanOrEqual(3);
    await expect.poll(() => page.getByTestId('dv-thread').count()).toBeGreaterThanOrEqual(3);

    const report = await page.evaluate(() => {
      type Box = { l: number; t: number; r: number; b: number };
      const box = (el: Element): Box => {
        const r = el.getBoundingClientRect();
        return { l: r.left, t: r.top, r: r.right, b: r.bottom };
      };
      // Chevauchement réel (au-delà d'un demi-pixel d'arrondi), pas un simple contact de bords.
      const overlap = (a: Box, b: Box) => Math.min(a.r, b.r) - Math.max(a.l, b.l) > 0.5 && Math.min(a.b, b.b) - Math.max(a.t, b.t) > 0.5;
      const name = (el: Element) => (el.textContent ?? '').trim();

      // Couleur CSS quelconque (oklch compris) → sRGB 0-255 composé sur `under`, via un canvas.
      const cv = document.createElement('canvas');
      cv.width = cv.height = 1;
      const cx = cv.getContext('2d', { willReadFrequently: true })!;
      const paint = (under: string, color: string) => {
        cx.clearRect(0, 0, 1, 1);
        cx.fillStyle = under;
        cx.fillRect(0, 0, 1, 1);
        cx.fillStyle = color;
        cx.fillRect(0, 0, 1, 1);
        return [...cx.getImageData(0, 0, 1, 1).data];
      };
      const lum = ([r, g, b]: number[]) => {
        const c = (v: number) => ((v /= 255) <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4);
        return 0.2126 * c(r) + 0.7152 * c(g) + 0.0722 * c(b);
      };

      const labels = [...document.querySelectorAll('[data-testid="dv-tile-name"]')].filter(
        (el) => el.checkVisibility({ visibilityProperty: true, opacityProperty: true }) && el.getBoundingClientRect().width > 0,
      );
      const tiles = [...document.querySelectorAll('[data-testid="dv-tile"], [data-testid="dv-visitor-tile"], [data-testid="dv-card"]')].filter((el) =>
        el.checkVisibility({ visibilityProperty: true }),
      );
      const problems: string[] = [];

      labels.forEach((el, i) => {
        const me = box(el);
        // 1. Aucune boîte de nom ne chevauche un autre nom ni une autre tuile (sa propre tuile exceptée).
        labels.slice(i + 1).forEach((o) => overlap(me, box(o)) && problems.push(`« ${name(el)} » chevauche le nom « ${name(o)} »`));
        tiles.forEach((tile) => tile !== el.parentElement && overlap(me, box(tile)) && problems.push(`« ${name(el)} » chevauche la tuile ${tile.getAttribute('data-project-id') ?? 'du visiteur'}`));

        // 2. Lignes VISIBLES (rognées par la boîte exclues) : deux au plus. 3. Aucun mot coupé : chaque
        // caractère d'un même mot (suite sans espace) est sur la même ligne.
        const lineTops = new Set<number>();
        const words: { word: string; tops: Set<number> }[] = [];
        let current: { word: string; tops: Set<number> } | null = null;
        const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
          const text = node.textContent ?? '';
          for (let k = 0; k < text.length; k++) {
            if (/\s/.test(text[k])) {
              current = null;
              continue;
            }
            const range = document.createRange();
            range.setStart(node, k);
            range.setEnd(node, k + 1);
            const r = range.getClientRects()[0];
            if (!r) continue;
            const top = Math.round(r.top);
            if (r.top < me.b - 1) lineTops.add(top);
            if (!current) words.push((current = { word: '', tops: new Set() }));
            current.word += text[k];
            current.tops.add(top);
          }
          // Un <br> sépare deux mots ; un <wbr> ne sépare RIEN (il découpe un mot).
          if (node.nextSibling?.nodeName === 'BR') current = null;
        }
        if (lineTops.size > 2) problems.push(`« ${name(el)} » tient sur ${lineTops.size} lignes`);
        for (const w of words) if (w.tops.size > 1) problems.push(`« ${name(el)} » : mot coupé « ${w.word} »`);

        // Contraste ≥ 4,5:1 sur le fond EFFECTIF : la plaque du nom, qui doit être opaque — sinon le
        // fond serait la peinture de la montagne et les filets, et rien ne garantirait la lecture.
        const st = getComputedStyle(el);
        const plate = paint('#000', st.backgroundColor);
        const plateWhite = paint('#fff', st.backgroundColor);
        if (plate.join() !== plateWhite.join()) problems.push(`« ${name(el)} » : fond transparent (${st.backgroundColor})`);
        else {
          const ink = paint(st.backgroundColor, st.color);
          const [hi, lo] = [lum(ink), lum(plate)].sort((a, b) => b - a);
          const ratio = (hi + 0.05) / (lo + 0.05);
          if (ratio < 4.5) problems.push(`« ${name(el)} » : contraste ${ratio.toFixed(2)}:1`);
        }
      });

      // 4. Les filets d'or ne passent pas PAR-DESSUS un nom : rendus cliquables le temps de la mesure,
      // ce qui est au-dessus en chaque point du nom doit être le nom lui-même (lui aussi rendu
      // cliquable : une plaque `pointer-events: none` laisserait passer la mesure, pas le regard).
      const svg = document.querySelector('[data-testid="dv-mountain"] svg') as SVGElement;
      svg.style.pointerEvents = 'auto';
      svg.querySelectorAll('*').forEach((n) => ((n as SVGElement).style.pointerEvents = 'visibleStroke'));
      labels.forEach((el) => ((el as HTMLElement).style.pointerEvents = 'auto'));
      for (const el of labels) {
        const r = el.getBoundingClientRect();
        const crossed = [0.1, 0.3, 0.5, 0.7, 0.9].some((fx) =>
          [0.25, 0.5, 0.75].some((fy) => svg.contains(document.elementFromPoint(r.left + fx * r.width, r.top + fy * r.height))),
        );
        if (crossed) problems.push(`« ${name(el)} » : un filet passe par-dessus`);
      }
      svg.removeAttribute('style');
      svg.querySelectorAll('*').forEach((n) => (n as SVGElement).style.removeProperty('pointer-events'));
      labels.forEach((el) => (el as HTMLElement).style.removeProperty('pointer-events'));
      return { count: labels.length, problems };
    });

    // Au moins les projets allumés et le visiteur portent leur nom.
    expect(report.count).toBeGreaterThanOrEqual(4);
    expect(report.problems, report.problems.join('\n')).toEqual([]);
  });

  test('interface en anglais (le contenu rejoué reste celui du scénario)', async ({ page }) => {
    await page.goto('/en/dossier-vivant?speed=2');
    await expect(page.getByTestId('dv-launch')).toHaveText(EN.dossierVivant.hero.launch);
    await expect(page.getByTestId('dv-caption')).toHaveText(EN.dossierVivant.live.captionIdle);
    await page.getByTestId('dv-pause').click();
    await expect(page.getByTestId('dv-orchestrator')).toContainText(EN.dossierVivant.team.agents.orchestrator.name, { timeout: 10_000 });
    await expect(page.locator('[data-testid="dv-agent-column"][data-agent="researcher"]')).toContainText(EN.dossierVivant.team.agents.researcher.name);
  });
});

/* --------------------------------------------------------------------------------------------
   Visiteur : run réel (flux SSE simulé au niveau réseau)
   -------------------------------------------------------------------------------------------- */

const MODEL = 'claude-haiku-5-5';
/** Le Dessinateur, seul agent sur un autre modèle (backend : DOSSIER_VIVANT_SCHEMA_MODEL). */
const SCHEMA_MODEL = 'claude-sonnet-5-5';
/** Projets du flux simulé : de vraies cartes de la montagne (les chiffres viennent des fiches). */
const LIVE_KEPT = ['melodicode', 'maicivy', 'envoia'];

/** Un run complet, au format du contrat (events.ts), tel que le backend l'émet. */
function liveRun(): object[] {
  const ev: object[] = [
    {
      type: 'run.started',
      visitor: { name: 'Jeanne Martin', affiliation: 'Directrice des opérations chez Atelier Nord', organisation: 'Atelier Nord', initials: 'JM', need: 'Nous voulons des agents qui préparent nos devis.' },
      agents: ['orchestrator', 'reader', 'researcher', 'architect', 'diagrammer', 'writer'].map((id) => ({ id, model: id === 'diagrammer' ? SCHEMA_MODEL : MODEL })),
    },
    { type: 'agent.status', agent: 'orchestrator', status: 'working', activity: 'lit la demande' },
    { type: 'message', from: 'orchestrator', to: 'orchestrator', text: 'quatre tâches en parallèle.', topic: 'plan' },
  ];
  for (const a of ['reader', 'researcher', 'architect', 'writer']) ev.push({ type: 'agent.status', agent: a, status: 'working', activity: 'au travail' });
  ev.push({ type: 'index.filtered', agent: 'researcher', total: 75, candidates: [...LIVE_KEPT, 'liveconf'] });
  for (const id of LIVE_KEPT) {
    ev.push({ type: 'tool.call', agent: 'researcher', call: `get_project("${id}")`, focus: { kind: 'project', projectId: id } });
    ev.push({ type: 'project.verdict', agent: 'researcher', projectId: id, verdict: 'keep', note: `${id} : retenu` });
    ev.push({ type: 'link.traced', from: 'visitor', to: id });
  }
  ev.push({ type: 'project.verdict', agent: 'researcher', projectId: 'liveconf', verdict: 'drop' });
  ev.push({ type: 'usage', agent: 'researcher', inputTokens: 9000, outputTokens: 700, costUsd: 0.00125 });
  // L'Architecte appelle son outil : le Dessinateur travaille sous les yeux et lui rend le schéma.
  ev.push({ type: 'tool.call', agent: 'architect', call: 'draw_schema(4 stages, 3 guardrails)', focus: { kind: 'visitor' } });
  ev.push({ type: 'message', from: 'architect', to: 'diagrammer', text: 'dessine la chaîne : 4 étapes, 3 garde-fous.' });
  ev.push({ type: 'agent.status', agent: 'diagrammer', status: 'working', activity: 'lit la commande de l’Architecte' });
  ev.push({ type: 'tool.call', agent: 'diagrammer', call: 'draft_schema(need, design)', focus: { kind: 'dossier' } });
  ev.push({ type: 'usage', agent: 'diagrammer', inputTokens: 2000, outputTokens: 3000, costUsd: 0.051 });
  ev.push({ type: 'agent.log', agent: 'diagrammer', kind: 'note', text: 'Le modèle rédige, le code tranche' });
  ev.push({ type: 'agent.status', agent: 'diagrammer', status: 'done' });
  ev.push({ type: 'message', from: 'diagrammer', to: 'architect', text: 'schéma en trois nœuds.' });
  ev.push({
    type: 'dossier.opened',
    sections: [
      { id: 'need', title: 'Le besoin, reformulé', waitingFor: 'reader' },
      { id: 'architecture', title: 'L’architecture proposée', waitingFor: 'architect' },
      { id: 'proofs', title: 'Les projets qui le prouvent', waitingFor: 'researcher' },
      { id: 'contribution', title: 'Ce qu’Alexis apporte', waitingFor: 'researcher' },
      { id: 'next', title: 'Prochaine étape', waitingFor: 'previous' },
    ],
  });
  const section = (id: string, blocks: object[]) => {
    ev.push({ type: 'dossier.section', section: id, status: 'writing' });
    for (const block of blocks) ev.push({ type: 'dossier.block', section: id, block });
    ev.push({ type: 'dossier.section', section: id, status: 'done' });
  };
  section('need', [{ type: 'lead', text: 'Préparer chaque devis en minutes.' }, { type: 'item', text: 'Un humain valide.' }]);
  section('architecture', [
    {
      type: 'pipeline',
      nodes: [
        { id: 'in', title: 'Demande', subtitle: 'entrante', col: 0, row: 0 },
        { id: 'draft', title: 'Rédacteur', subtitle: 'modèle léger', col: 1, row: 0 },
        { id: 'human', title: 'Validation', subtitle: 'humaine', col: 2, row: 0, human: true },
      ],
      edges: [{ from: 'in', to: 'draft', step: 1 }, { from: 'draft', to: 'human' }],
      journal: 'Journal des décisions',
      label: 'Chaîne proposée',
    },
  ]);
  section('proofs', LIVE_KEPT.map((id) => ({ type: 'proof', projectId: id, why: 'Des agents en production.' })));
  section('contribution', [{ type: 'item', text: 'L’architecture, de bout en bout.' }]);
  section('next', [{ type: 'paragraph', text: 'Un appel de 30 minutes, puis un prototype.' }, { type: 'contact' }]);
  for (const a of ['reader', 'researcher', 'architect', 'writer', 'orchestrator']) ev.push({ type: 'agent.status', agent: a, status: 'done' });
  ev.push({ type: 'run.completed', dossierId: LIVE_DOSSIER_ID });
  return ev;
}

/** L'id de la copie serveur que le flux simulé annonce à la remise. */
const LIVE_DOSSIER_ID = 'ab'.repeat(16);

/** Un PDF minimal (en-tête `%PDF`) : la réponse simulée de la route du PDF d'un run réel. */
const MINI_PDF = ['%PDF-1.4', '1 0 obj<<>>endobj', 'trailer<<>>', '%%EOF', ''].join('\n');


test.describe('dossier vivant — visiteur, run réel (flux simulé au réseau)', () => {
  test.setTimeout(120_000);

  test('le visiteur écrit son besoin, lance l’équipe : le flux SSE déroule le parcours jusqu’au dossier', async ({ page, request }) => {
    const facts = await realFacts(request);
    let posted: { need: string; who: string; lang: string } | null = null;
    await page.route('**/api/v1/dossier-vivant/run', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      posted = route.request().postDataJSON();
      return route.fulfill({ status: 200, headers: cors(route), contentType: 'text/event-stream', body: sse(liveRun()) });
    });

    await page.goto('/fr/dossier-vivant');
    const need = page.getByTestId('dv-need');
    await expect(need).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await need.fill('Nous voulons des agents qui préparent nos devis à partir des demandes clients.');
    await page.getByTestId('dv-who').fill('Jeanne Martin, Directrice des opérations chez Atelier Nord');
    await page.getByTestId('dv-launch').click();

    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-mode', 'live');
    await expect.poll(() => posted).toEqual({
      need: 'Nous voulons des agents qui préparent nos devis à partir des demandes clients.',
      who: 'Jeanne Martin, Directrice des opérations chez Atelier Nord',
      lang: 'fr',
    });

    // Le run est annoncé comme réel, avec le modèle venu du flux — jamais « rejoué ».
    await expect(page.getByTestId('dv-caption')).toHaveAttribute('data-mode', 'live');
    await expect(page.getByTestId('dv-caption')).toContainText(MODEL);
    await expect(page.getByTestId('dv-visitor-tile')).toContainText('Jeanne Martin', { timeout: 10_000 });

    // Le Dessinateur annoncé par le flux a SA colonne, avec SON modèle (le rejeu n'en a que quatre).
    await expect(page.getByTestId('dv-agent-column')).toHaveCount(5);
    const drafter = page.locator('[data-testid="dv-agent-column"][data-agent="diagrammer"]');
    await expect(drafter).toContainText(FR.dossierVivant.team.agents.diagrammer.name);
    await expect(drafter).toContainText(SCHEMA_MODEL);

    // Le parcours se déroule jusqu'au dossier : projets allumés, cinq sections, preuves chiffrées.
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    for (const id of LIVE_KEPT) {
      await expect(page.locator(`[data-project-id="${id}"]:not([data-testid="dv-proof-row"])`)).toHaveAttribute('data-state', 'lit');
      await expect(page.locator(`[data-testid="dv-thread"][data-to="${id}"]`)).toHaveCount(1);
    }
    for (const id of SECTIONS) await expect(page.locator(`[data-testid="dv-section"][data-section="${id}"]`)).toHaveAttribute('data-status', 'done');
    await expect(page.getByTestId('dv-proof-row')).toHaveCount(LIVE_KEPT.length);
    for (const id of LIVE_KEPT) {
      const row = page.locator(`[data-testid="dv-proof-row"][data-project-id="${id}"]`);
      await expect(row).toContainText(new Intl.NumberFormat('fr').format(facts.get(id)!.loc));
    }
    await expect(page.getByTestId('dv-dossier-state')).toHaveText(FR.dossierVivant.dossier.state.ready);
    await expect(page.getByTestId('dv-download')).toBeVisible();
    await expect(drafter).toHaveAttribute('data-status', 'done');
    // L'en-tête compte l'équipe du run : 6 agents, pas les 5 du rejeu.
    await expect(page.getByTestId('dv-dossier-subtitle')).toContainText('par 6 agents');

    // Le PDF d'un run réel part sur la route PUBLIQUE du run, avec l'id de la copie serveur et la
    // langue — rien d'imprimable.
    let pdfBody: unknown = null;
    await page.route('**/api/v1/dossier-vivant/run/pdf', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      pdfBody = route.request().postDataJSON();
      return route.fulfill({ status: 200, headers: cors(route), contentType: 'application/pdf', body: MINI_PDF });
    });
    const [download] = await Promise.all([page.waitForEvent('download'), page.getByTestId('dv-download').click()]);
    expect(download.suggestedFilename()).toBe('dossier-de-conception-atelier-nord.pdf');
    expect(readFileSync((await download.path())!).subarray(0, 4).toString('latin1')).toBe('%PDF');
    expect(pdfBody).toEqual({ lang: 'fr', dossierId: LIVE_DOSSIER_ID });
  });

  // Le serveur ne connaît pas (ou plus) le dossier : l'erreur est NOMMÉE sur le bouton, glyphe ✕
  // compris — pas un clic sans effet, ni le « réessayer » d'une panne.
  test('dossier expiré côté serveur : le bouton le dit', async ({ page }) => {
    await page.route('**/api/v1/dossier-vivant/run', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      return route.fulfill({ status: 200, headers: cors(route), contentType: 'text/event-stream', body: sse(liveRun()) });
    });
    await page.route('**/api/v1/dossier-vivant/run/pdf', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      return route.fulfill({ status: 404, headers: cors(route), contentType: 'application/json', body: JSON.stringify({ error: 'x', code: 'DOSSIER_NOT_FOUND' }) });
    });
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await page.getByTestId('dv-launch').click();
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    const button = page.getByTestId('dv-download');
    await button.click();
    await expect(button).toHaveAttribute('data-pdf', 'expired');
    await expect(button).toHaveText(FR.dossierVivant.dossier.pdf.expired);
  });

  // « Rejouer » en direct : un NOUVEAU round réel sur le même besoin, en un clic, une fois le
  // précédent fini — et l'état du PDF du round d'avant (ici expiré) ne déteint pas sur le suivant.
  test('rejouer après un round réel : un second run part, même besoin, dossier neuf', async ({ page }) => {
    const posted: Array<{ need: string; who: string; lang: string }> = [];
    await page.route('**/api/v1/dossier-vivant/run', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      posted.push(route.request().postDataJSON());
      return route.fulfill({ status: 200, headers: cors(route), contentType: 'text/event-stream', body: sse(liveRun()) });
    });
    await page.route('**/api/v1/dossier-vivant/run/pdf', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      return route.fulfill({ status: 404, headers: cors(route), contentType: 'application/json', body: JSON.stringify({ error: 'x', code: 'DOSSIER_NOT_FOUND' }) });
    });
    await page.goto('/fr/dossier-vivant');
    const replay = page.getByTestId('dv-replay');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    // Rien n'est encore lancé : rien à rejouer.
    await expect(replay).toBeDisabled();
    await page.getByTestId('dv-need').fill('Des agents qui trient nos demandes de devis.');
    await page.getByTestId('dv-launch').click();
    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    await page.getByTestId('dv-download').click();
    await expect(page.getByTestId('dv-download')).toHaveAttribute('data-pdf', 'expired');

    await expect(replay).toBeEnabled();
    await replay.click();
    await expect.poll(() => posted.length).toBe(2);
    expect(posted[1]).toEqual(posted[0]);
    expect(posted[1].need).toBe('Des agents qui trient nos demandes de devis.');
    await expect(root).toHaveAttribute('data-mode', 'live');
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    for (const id of SECTIONS) await expect(page.locator(`[data-testid="dv-section"][data-section="${id}"]`)).toHaveAttribute('data-status', 'done');
    await expect(page.getByTestId('dv-download')).toHaveAttribute('data-pdf', 'idle');
  });

  // Un run ne s'arrête plus ni sur sa consommation ni sur sa durée : seules restent les VRAIES
  // erreurs. Celle-ci — sortie de modèle inutilisable — se voit, nommée, et rien n'est remis.
  test('un arrêt sur sortie inutilisable se voit, nommé, et le dossier n’est pas remis', async ({ page }) => {
    await page.route('**/api/v1/dossier-vivant/run', (route) => {
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
      const events = liveRun().slice(0, 8);
      return route.fulfill({ status: 200, headers: cors(route), contentType: 'text/event-stream', body: sse([...events, { type: 'run.failed', reason: 'model' }]) });
    });
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await page.getByTestId('dv-launch').click();
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-run', 'failed', { timeout: 15_000 });
    await expect(page.getByTestId('dv-caption')).toHaveAttribute('data-failed', 'model');
    await expect(page.getByTestId('dv-caption')).toContainText(FR.dossierVivant.live.failed.model);
    await expect(page.getByTestId('dv-download')).toHaveCount(0);
  });

  test('sans connexion : champ éditable pré-rempli par l’exemple, mention « en direct », démo à un clic', async ({ page }) => {
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });
    await expect(page.getByTestId('dv-need')).toHaveJSProperty('readOnly', false);
    await expect(page.getByTestId('dv-who')).toHaveJSProperty('readOnly', false);
    const caption = page.getByTestId('dv-caption');
    await expect(caption).toHaveAttribute('data-mode', 'idle');
    await expect(caption).toHaveText(FR.dossierVivant.live.captionIdle);
    await expect(caption).not.toContainText(FR.dossierVivant.bar.caption);
    // La démo : le champ repasse en lecture seule sur le besoin du scénario, la mention le dit.
    await page.getByTestId('dv-pause').click();
    await expect(caption).toHaveAttribute('data-mode', 'replay');
    await expect(caption).toHaveText(FR.dossierVivant.bar.caption);
    await expect(page.getByTestId('dv-need')).toHaveJSProperty('readOnly', true);
  });
});

/* --------------------------------------------------------------------------------------------
   Visiteur sans cookie : run réel de bout en bout contre le backend (banc)
   -------------------------------------------------------------------------------------------- */

test.describe('dossier vivant — visiteur sans cookie, run réel sur le backend', () => {
  // Un run réel dure ce qu'il dure : ~15 s avec les agents simulés du banc, une à quelques dizaines
  // de secondes avec de vrais modèles — rien ne le coupe côté serveur en deçà de la limite de
  // transport (120 s).
  test.setTimeout(180_000);

  test('sans connexion : saisir un besoin, lancer, suivre le flux en direct, lire le dossier, télécharger SON PDF', async ({ page, context, request }) => {
    // Rien n'est simulé dans la page : ni cookie, ni route interceptée. Le banc répond pour de vrai.
    expect(await context.cookies()).toEqual([]);
    const calls: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/')) calls.push(`${req.method()} ${new URL(req.url()).pathname}`);
    });

    await page.goto('/fr/dossier-vivant');
    const need = page.getByTestId('dv-need');
    await expect(need).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    const besoin = 'Notre bureau d’études reçoit 300 demandes de devis par semaine ; nous voulons des agents qui préparent chaque devis, avec validation humaine.';
    await need.fill(besoin);
    await page.getByTestId('dv-who').fill('Jeanne Martin, Directrice des opérations chez Atelier Nord');

    const runResponse = page.waitForResponse((r) => r.url().endsWith('/api/v1/dossier-vivant/run') && r.request().method() === 'POST');
    await page.getByTestId('dv-launch').click();
    const res = await runResponse;
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toContain('text/event-stream');

    // Le run part, annoncé comme réel : le visiteur et son besoin, l'équipe annoncée par le flux.
    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-mode', 'live');
    const caption = page.getByTestId('dv-caption');
    await expect(caption).toHaveAttribute('data-mode', 'live');
    await expect(caption).not.toContainText(FR.dossierVivant.bar.caption);
    await expect(page.getByTestId('dv-visitor-tile')).toContainText('Jeanne Martin', { timeout: 15_000 });
    await expect(caption).toContainText(/claude-|gpt-/);
    await expect(page.getByTestId('dv-agent-column')).toHaveCount(5);

    // Les événements arrivent AU FIL du travail : des agents au travail et des messages AVANT la remise.
    await expect(page.locator('[data-testid="dv-agent-column"][data-status="working"]').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.getByTestId('dv-thread-message').first()).toBeVisible();
    expect(await root.getAttribute('data-run')).toBe('running');

    // Le dossier est remis : cinq sections écrites, au moins une preuve, des projets allumés.
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 150_000 });
    for (const id of SECTIONS) await expect(page.locator(`[data-testid="dv-section"][data-section="${id}"]`)).toHaveAttribute('data-status', 'done');
    expect(await page.getByTestId('dv-proof-row').count()).toBeGreaterThan(0);
    expect(await page.locator('[data-testid="dv-tile"][data-state="lit"], [data-testid="dv-card"][data-state="lit"]').count()).toBeGreaterThan(0);
    await expect(page.getByTestId('dv-dossier-state')).toHaveText(FR.dossierVivant.dossier.state.ready);

    // SON PDF : rendu par le backend (route publique du run réel), pas un fichier simulé. La page
    // n'envoie que la langue et l'id de la copie que le serveur a gardée à la remise.
    const button = page.getByTestId('dv-download');
    const asked = page.waitForRequest((r) => r.url().endsWith('/api/v1/dossier-vivant/run/pdf') && r.method() === 'POST');
    const [download] = await Promise.all([page.waitForEvent('download', { timeout: 60_000 }), button.click()]);
    const query = (await asked).postDataJSON() as { lang: string; dossierId: string };
    expect(Object.keys(query).sort()).toEqual(['dossierId', 'lang']);
    expect(query.lang).toBe('fr');
    expect(query.dossierId).toMatch(/^[0-9a-f]{32}$/);
    expect(download.suggestedFilename()).toBe('dossier-de-conception-atelier-nord.pdf');
    const bytes = readFileSync((await download.path())!);
    expect(bytes.subarray(0, 4).toString('latin1')).toBe('%PDF');
    expect(bytes.length).toBeGreaterThan(5_000);
    await expect(button).toHaveAttribute('data-pdf', 'idle');
    if (process.env.DV_EVIDENCE_DIR) {
      // Preuves d'un run réel (critère de recette) : le PDF et l'écran de fin, gardés hors du dépôt.
      await download.saveAs(path.join(process.env.DV_EVIDENCE_DIR, download.suggestedFilename()));
      await page.screenshot({ path: path.join(process.env.DV_EVIDENCE_DIR, 'fin-de-run.png') });
    }

    // Un id que le serveur n'a pas gardé : erreur nommée, aucun PDF.
    const unknown = await request.post(`${API}/api/v1/dossier-vivant/run/pdf`, { data: { lang: 'fr', dossierId: 'cd'.repeat(16) } });
    expect(unknown.status()).toBe(404);
    expect((await unknown.json()).code).toBe('DOSSIER_NOT_FOUND');

    // Le parcours n'a touché AUCUNE route /admin : ni session, ni run owner, ni PDF owner.
    expect(calls.filter((c) => c.includes('/admin/')), calls.join('\n')).toEqual([]);
    expect(calls).toContain('POST /api/v1/dossier-vivant/run');
    expect(calls).toContain('POST /api/v1/dossier-vivant/run/pdf');
  });
});

/* --------------------------------------------------------------------------------------------
   Clés React : aucune en double, du début à la remise
   -------------------------------------------------------------------------------------------- */

test.describe('dossier vivant — clés React', () => {
  test.setTimeout(120_000);

  test('rejeu Tessera complet : aucune clé en double en console', async ({ page }) => {
    const duplicates = trackDuplicateKeys(page);
    await page.goto('/fr/dossier-vivant?speed=4');
    await page.getByTestId('dv-pause').click();
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
    await expect(page.getByTestId('dv-skills')).toHaveAttribute('data-status', /ready|failed/, { timeout: 20_000 });
    expect(duplicates, duplicates.join('\n')).toEqual([]);
  });

  test('rejeu du flux brut d’un vrai run (reçu d’un bloc) : aucune clé en double en console', async ({ page }) => {
    // Le cas qui l'a révélé : deux messages « orchestrateur → orchestrateur » du même paquet, même instant.
    const duplicates = trackDuplicateKeys(page);
    await mockLiveRun(page, recordedRun());
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await page.getByTestId('dv-launch').click();
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    await expect(page.getByTestId('dv-agent-column')).toHaveCount(5);
    await expect(page.getByTestId('dv-skills')).toHaveAttribute('data-status', /ready|failed/, { timeout: 20_000 });
    expect(duplicates, duplicates.join('\n')).toEqual([]);
  });
});

/* --------------------------------------------------------------------------------------------
   La page défile : le dossier remis se lit en faisant défiler la PAGE, jusqu'au pied de page
   -------------------------------------------------------------------------------------------- */

/** Téléphone en portrait : sous 760 px de large, disposition `compact` (scene.ts › layoutFor). */
const PHONE = { width: 390, height: 844 };

/** `el` entièrement dans la fenêtre (haut ≥ 0, bas ≤ hauteur). */
const fullyInView = (el: Locator) =>
  el.evaluate((e) => {
    const r = e.getBoundingClientRect();
    return r.top >= 0 && r.bottom <= innerHeight;
  });

/** Molette (vrais événements `wheel`) au milieu de la feuille, jusqu'à ce que `target` soit entièrement visible. */
async function wheelUntil(page: Page, target: Locator) {
  const sheet = (await page.getByTestId('dv-dossier').boundingBox())!;
  await page.mouse.move(sheet.x + sheet.width / 2, page.viewportSize()!.height / 2);
  for (let i = 0; i < 80 && !(await fullyInView(target)); i++) {
    await page.mouse.wheel(0, 300);
    await page.waitForTimeout(40);
  }
}

/**
 * Preuve (capture) gardée hors du dépôt quand le banc le demande. Les animations CSS FINIES d'abord
 * (fondu d'entrée de la fleur) : sinon la capture montre un calque encore transparent. Les boucles
 * infinies (pulsations de la scène) sont écartées : leur `finished` ne se résout jamais.
 */
async function evidence(page: Page, name: string) {
  if (!process.env.DV_EVIDENCE_DIR) return;
  await page.evaluate(() =>
    Promise.all(
      document
        .getAnimations()
        .filter((a) => a instanceof CSSAnimation && a.effect?.getTiming().iterations !== Infinity)
        .map((a) => a.finished),
    ),
  );
  await page.screenshot({ path: path.join(process.env.DV_EVIDENCE_DIR, name) });
}

/**
 * Le dossier remis se lit en défilant la PAGE : document plus haut que la fenêtre, feuille sans
 * ascenseur interne, dernière section atteinte à la molette, puis le pied de page et ses liens.
 */
async function readsByScrollingThePage(page: Page, shots?: { top: string; bottom: string }) {
  const root = page.getByTestId('dv-root');
  await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
  await expect(page.getByTestId('dv-page')).toHaveAttribute('data-unrolled', '');
  // La feuille a sa hauteur finale (les compétences arrivent après la remise).
  await expect(page.getByTestId('dv-skills')).toHaveAttribute('data-status', /ready|failed/, { timeout: 20_000 });

  const start = await page.evaluate(() => ({ scrollHeight: document.scrollingElement!.scrollHeight, innerHeight, scrollY }));
  expect(start.scrollHeight, 'la page est plus haute que la fenêtre').toBeGreaterThan(start.innerHeight);
  // Au premier écran : grand écran, le haut de page (l'en-tête du site au-dessus de la scène) ;
  // téléphone, juste sous l'en-tête du site (la scène y garde tout l'écran sous la barre).
  const siteHeaderH = await page.getByTestId('site-header').evaluate((e) => e.getBoundingClientRect().height);
  expect(start.scrollY, 'au premier écran, au plus l’en-tête du site défilé').toBeLessThanOrEqual(siteHeaderH);
  // Une seule copie du dossier, et elle ne défile plus EN ELLE-MÊME : c'est la page qui défile.
  await expect(page.getByTestId('dv-dossier')).toHaveCount(1);
  const body = page.getByTestId('dv-dossier').locator('[class*="dosBody"]');
  expect(await body.evaluate((el) => el.scrollHeight - el.clientHeight), 'aucun ascenseur interne').toBeLessThanOrEqual(1);
  if (shots) await evidence(page, shots.top);

  // La dernière section n'est pas entièrement visible au premier écran : on l'atteint à la molette.
  const next = page.locator('[data-testid="dv-section"][data-section="next"]');
  await expect(next).not.toBeInViewport({ ratio: 1 });
  await wheelUntil(page, next);
  await expect(next).toBeInViewport({ ratio: 1 });
  expect(await page.evaluate(() => scrollY)).toBeGreaterThan(start.scrollY);
  expect(await body.evaluate((el) => el.scrollTop)).toBe(0);
  // La barre (pause, rejouer, état du run) reste à portée pendant la lecture.
  await expect(page.getByTestId('dv-replay')).toBeInViewport();

  // Puis le pied de page : quatre liens visibles, avec leurs vraies cibles.
  const footer = page.getByTestId('dv-footer');
  await wheelUntil(page, footer);
  await expect(footer).toBeInViewport({ ratio: 1 });
  const links: Record<string, string | RegExp> = {
    cv: /\/api\/v1\/cv\/export\?theme=fullstack&format=pdf&lang=fr$/,
    portfolio: '/fr/portfolio',
    montagne: '/fr', // la montagne est l'accueil depuis la refonte (2026-11)
    contact: 'mailto:alexistrouve.pro@gmail.com',
  };
  for (const [id, href] of Object.entries(links)) {
    const link = page.getByTestId(`dv-footer-${id}`);
    await expect(link).toBeVisible();
    await expect(link).toHaveText(FR.dossierVivant.footer[id]);
    await expect(link).toHaveAttribute('href', href);
  }
  if (shots) await evidence(page, shots.bottom);
}

test.describe('dossier vivant — la page défile jusqu’au pied de page (grand écran)', () => {
  test.setTimeout(120_000);

  test('avant tout départ, le pied de page est déjà là, sous la scène', async ({ page }) => {
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });
    const footer = page.getByTestId('dv-footer');
    await expect(footer).toHaveCount(1);
    // Sous le premier écran, jamais par-dessus la scène.
    expect((await footer.boundingBox())!.y).toBeGreaterThanOrEqual(page.viewportSize()!.height - 1);
    await expect(page.getByTestId('dv-page')).not.toHaveAttribute('data-unrolled', '');
  });

  test('rejeu Tessera remis : le dossier se lit en défilant la page, jusqu’au pied de page', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=4');
    await page.getByTestId('dv-pause').click();
    await readsByScrollingThePage(page, { top: 'grand-ecran-fin-de-run-haut.png', bottom: 'grand-ecran-fin-de-run-pied.png' });
  });

  test('run réel remis (flux brut d’un vrai run, simulé au réseau) : même lecture en défilant la page', async ({ page }) => {
    await mockLiveRun(page, recordedRun());
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await page.getByTestId('dv-launch').click();
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-mode', 'live');
    await readsByScrollingThePage(page);
  });

  test('« Rejouer » depuis le bas du dossier ramène au premier écran, où la scène repart', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=4');
    await page.getByTestId('dv-pause').click();
    await readsByScrollingThePage(page);
    await page.getByTestId('dv-replay').click();
    await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
    await expect(page.getByTestId('dv-page')).not.toHaveAttribute('data-unrolled', '');
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-run', 'running');
  });
});

test.describe('dossier vivant — la page défile jusqu’au pied de page (téléphone 390 × 844)', () => {
  test.setTimeout(120_000);
  test.use({ viewport: PHONE });

  test('rejeu Tessera remis : le dossier plein écran se lit en défilant la page', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=4');
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-layout', 'compact');
    await page.getByTestId('dv-pause').click();
    await readsByScrollingThePage(page, { top: 'telephone-fin-de-run-haut.png', bottom: 'telephone-defile-pied.png' });
  });

  test('run réel remis (flux simulé au réseau) : même lecture en défilant la page', async ({ page }) => {
    await mockLiveRun(page, recordedRun());
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await page.getByTestId('dv-launch').click();
    await readsByScrollingThePage(page);
  });
});

/* --------------------------------------------------------------------------------------------
   Ouverture : l'en-tête du site au-dessus de la barre, la montagne ENTIÈRE avant le pied de page
   -------------------------------------------------------------------------------------------- */

/** La barre du dossier vivant : l'en-tête qui porte pause / rejouer / progression. */
const dvBar = (page: Page) => page.locator('header').filter({ has: page.getByTestId('dv-pause') });

/**
 * La barre est sous l'en-tête du site (jamais dessous, jamais par-dessus) et c'est bien ELLE qui est
 * peinte en son centre — une barre cachée sous l'en-tête aurait une boîte, mais ne se verrait pas.
 */
async function barBelowSiteHeader(page: Page, moment: string) {
  const header = (await page.getByTestId('site-header').boundingBox())!;
  const bar = (await dvBar(page).boundingBox())!;
  expect.soft(bar.y, `barre sous l’en-tête du site — ${moment}`).toBeGreaterThanOrEqual(Math.max(0, header.y + header.height) - 0.5);
  const painted = await page.evaluate(
    ([x, y]) => !!document.elementFromPoint(x, y)?.closest('header:has([data-testid="dv-pause"])'),
    [bar.x + bar.width / 2, bar.y + bar.height / 2],
  );
  expect.soft(painted, `barre peinte au premier plan — ${moment}`).toBe(true);
  await expect.soft(page.getByTestId('dv-pause'), `pause à portée — ${moment}`).toBeInViewport({ ratio: 1 });
}

/**
 * Phase d'ouverture : (1) l'en-tête du site est là, en haut, la barre du dossier dessous ; (2) en
 * défilant, le DERNIER rang de tuiles de la montagne est entier — boîte au-dessus du haut du pied
 * de page, et réellement peint (pas rogné par la scène) — puis vient le pied de page.
 */
async function introShowsSiteHeaderAndWholeMountain(page: Page, shots?: { top: string; foot: string }) {
  await page.goto('/fr/dossier-vivant');
  await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });
  await expect(page.getByTestId('dv-root')).toHaveAttribute('data-phase', 'intro');
  await expect.poll(() => page.getByTestId('dv-tile').count(), { timeout: 15_000 }).toBeGreaterThan(0);

  // 1. L'en-tête du site, en haut de la page, avec sa navigation.
  const header = page.getByTestId('site-header');
  await expect.soft(header, 'en-tête du site présent').toBeVisible();
  if (await header.count()) {
    expect.soft((await header.boundingBox())!.y).toBeLessThanOrEqual(1);
    await barBelowSiteHeader(page, 'haut de page');
    // Un seul « Alexis Trouvé » : l'en-tête du site le porte, la barre du dossier ne le répète pas.
    await expect.soft(page.getByTestId('dv-wordmark'), 'nom pas répété dans la barre du dossier').toBeHidden();
  }
  if (shots) await evidence(page, shots.top);

  // 2. Le dernier rang de tuiles, entier, AVANT le pied de page.
  const boxes = await page.getByTestId('dv-tile').evaluateAll((els) =>
    els.map((e, i) => {
      const r = e.getBoundingClientRect();
      return { i, bottom: r.bottom + scrollY };
    }),
  );
  const lowest = boxes.reduce((a, b) => (b.bottom > a.bottom ? b : a));
  const footerTop = (await page.getByTestId('dv-footer').boundingBox())!.y + (await page.evaluate(() => scrollY));
  expect(lowest.bottom, 'bas du dernier rang de tuiles au-dessus du haut du pied de page').toBeLessThanOrEqual(footerTop);

  // Défilé jusqu'au pied de la montagne : la tuile la plus basse est entièrement à l'écran ET peinte.
  const tile = page.getByTestId('dv-tile').nth(lowest.i);
  await tile.scrollIntoViewIfNeeded();
  // 0,99 et non 1 : sur téléphone la tuile est à l'échelle ~0,3, ses bords tombent entre deux pixels
  // (mesuré : 0,9999997).
  await expect(tile).toBeInViewport({ ratio: 0.99 });
  const t = (await tile.boundingBox())!;
  const painted = await tile.evaluate((el, [x, y]) => document.elementFromPoint(x, y)?.closest('[data-testid="dv-tile"]') === el, [t.x + t.width / 2, t.y + t.height - 2]);
  expect(painted, 'le dernier rang n’est pas rogné par la scène').toBe(true);
  if (await header.count()) await barBelowSiteHeader(page, 'défilé au pied de la montagne');
  if (shots) await evidence(page, shots.foot);
}

test.describe('dossier vivant — ouverture : en-tête du site et montagne entière', () => {
  test.setTimeout(60_000);

  test('PC 1920 × 947 : en-tête du site au-dessus de la barre, dernier rang entier avant le pied de page', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 947 });
    await introShowsSiteHeaderAndWholeMountain(page, { top: 'pc-ouverture-haut.png', foot: 'pc-ouverture-pied-montagne.png' });
  });

  test('téléphone 390 × 844 : en-tête du site au-dessus de la barre, dernier rang entier avant le pied de page', async ({ page }) => {
    await page.setViewportSize(PHONE);
    await introShowsSiteHeaderAndWholeMountain(page, { top: 'telephone-ouverture-haut.png', foot: 'telephone-ouverture-pied-montagne.png' });
  });
});

/* --------------------------------------------------------------------------------------------
   Clic sur un projet = la fleur ; second clic = la page portfolio
   -------------------------------------------------------------------------------------------- */

interface ApiMontagne {
  nodes: { id: string; curated: boolean }[];
  summit: { id: string; members: string[] }[];
}

/** Les projets curés (ceux qui ont une page portfolio) et les cartes du sommet, lus sur l'API réelle. */
async function montagneFacts(request: APIRequestContext): Promise<{ curated: Set<string>; cards: Set<string> }> {
  const m = (await (await request.get(`${API}/api/v1/montagne?lang=fr`)).json()) as ApiMontagne;
  return { curated: new Set(m.nodes.filter((n) => n.curated).map((n) => n.id)), cards: new Set(m.summit.map((c) => c.id)) };
}

/** Une tuile (pas une carte) de la montagne affichée dont le projet est curé. */
async function curatedTile(page: Page, curated: Set<string>): Promise<Locator> {
  const ids = await page.getByTestId('dv-tile').evaluateAll((els) => els.map((e) => e.getAttribute('data-project-id') ?? ''));
  const id = ids.find((x) => curated.has(x));
  expect(id, 'au moins une tuile curée sur la montagne').toBeTruthy();
  return page.locator(`[data-testid="dv-tile"][data-project-id="${id}"]`);
}

const flower = (page: Page) => page.getByTestId('montagne-flower');

test.describe('dossier vivant — la fleur d’un projet (grand écran)', () => {
  test.setTimeout(120_000);

  test('tuile : clic → sa fleur ; Échap ferme et rend le focus ; Entrée au clavier ; second clic → portfolio', async ({ page, request }) => {
    const { curated } = await montagneFacts(request);
    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });
    const tile = await curatedTile(page, curated);
    const id = (await tile.getAttribute('data-project-id'))!;
    // Un vrai bouton, nommé par le projet.
    await expect(tile).toHaveRole('button');
    await expect(tile).toHaveAccessibleName(new RegExp(`\\S`));

    await tile.click();
    await expect(flower(page)).toBeVisible();
    await expect(flower(page)).toHaveAttribute('data-project-id', id);
    await page.keyboard.press('Escape');
    await expect(flower(page)).toHaveCount(0);
    await expect(tile).toBeFocused();

    // Clavier : Entrée sur la tuile focalisée ouvre la même fleur ; le bouton de fermeture ferme.
    await page.keyboard.press('Enter');
    await expect(flower(page)).toHaveAttribute('data-project-id', id);
    await page.getByTestId('montagne-flower-close').click();
    await expect(flower(page)).toHaveCount(0);
    await expect(tile).toBeFocused();

    // Second clic : le bouton de la fleur mène à la page portfolio du projet.
    await tile.click();
    await page.getByTestId('montagne-flower-portfolio').click();
    await expect(page).toHaveURL(new RegExp(`/fr/portfolio/${id}$`), { timeout: 30_000 });
  });

  test('carte du sommet : la fleur de ses preuves ; la tuile du visiteur n’ouvre rien', async ({ page, request }) => {
    const { cards } = await montagneFacts(request);
    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });
    const cardId = [...cards][0];
    await page.locator(`[data-testid="dv-card"][data-project-id="${cardId}"]`).click();
    await expect(flower(page)).toHaveAttribute('data-project-id', cardId);
    await expect(flower(page)).toHaveAttribute('data-variant', 'proofs');
    // Fermer par le fond (hors de la colonne de la fleur).
    await flower(page).click({ position: { x: 4, y: 4 } });
    await expect(flower(page)).toHaveCount(0);

    await page.getByTestId('dv-visitor-tile').click();
    await page.waitForTimeout(300);
    await expect(flower(page)).toHaveCount(0);
  });

  test('ligne de preuve du dossier : clic → la fleur du bon projet ; second clic → portfolio', async ({ page, request }) => {
    const { curated } = await montagneFacts(request);
    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });
    for (const id of PROOFS) {
      const row = page.locator(`[data-testid="dv-proof-row"][data-project-id="${id}"]`);
      await row.click();
      await expect(flower(page)).toHaveAttribute('data-project-id', id);
      if (id === PROOFS[1]) await evidence(page, 'fleur-depuis-ligne-de-preuve.png');
      await page.keyboard.press('Escape');
      await expect(flower(page)).toHaveCount(0);
      // Le focus revient au bouton du projet de la ligne.
      await expect(row.getByRole('button')).toBeFocused();
    }
    const id = PROOFS.find((x) => curated.has(x));
    expect(id, 'au moins une preuve du scénario est un projet curé').toBeTruthy();
    await page.locator(`[data-testid="dv-proof-row"][data-project-id="${id}"]`).click();
    await page.getByTestId('montagne-flower-portfolio').click();
    await expect(page).toHaveURL(new RegExp(`/fr/portfolio/${id}$`), { timeout: 30_000 });
  });

  test('le rejeu continue pendant que la fleur est ouverte', async ({ page, request }) => {
    const { curated } = await montagneFacts(request);
    await page.goto('/fr/dossier-vivant?speed=2');
    await page.getByTestId('dv-pause').click();
    await expect(page.getByTestId('dv-visitor-tile')).toBeVisible({ timeout: 10_000 });
    const root = page.getByTestId('dv-root');
    await (await curatedTile(page, curated)).click();
    await expect(flower(page)).toBeVisible();
    await expect(root).toHaveAttribute('data-run', 'running');
    // L'horloge avance derrière la fleur : les totaux changent, la barre reste en lecture.
    const before = await page.getByTestId('dv-totals').textContent();
    await expect.poll(() => page.getByTestId('dv-totals').textContent(), { timeout: 10_000 }).not.toBe(before);
    await expect(page.getByTestId('dv-pause')).toHaveText(FR.dossierVivant.bar.pause);
    await page.keyboard.press('Escape');
    await expect(flower(page)).toHaveCount(0);
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
  });

  test('run réel (flux SSE servi en continu) : ouvrir puis fermer la fleur ne coupe pas le run, la remise arrive', async ({ page, request }) => {
    const { curated } = await montagneFacts(request);
    const run = await streamLiveRun(page, liveRun());
    try {
      await page.goto('/fr/dossier-vivant');
      await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
      await page.getByTestId('dv-launch').click();
      const root = page.getByTestId('dv-root');
      await expect(root).toHaveAttribute('data-mode', 'live');
      await expect(page.getByTestId('dv-visitor-tile')).toContainText('Jeanne Martin', { timeout: 15_000 });
      await expect(root).toHaveAttribute('data-run', 'running');

      const tile = await curatedTile(page, curated);
      await tile.click();
      await expect(flower(page)).toHaveAttribute('data-project-id', (await tile.getAttribute('data-project-id'))!);
      await expect(root).toHaveAttribute('data-run', 'running');
      // La remise part du serveur PENDANT que la fleur est ouverte : le flux n'a pas été coupé.
      run.release();
      await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
      await expect(flower(page)).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(flower(page)).toHaveCount(0);
      await expect(tile).toBeFocused();
      for (const id of SECTIONS) await expect(page.locator(`[data-testid="dv-section"][data-section="${id}"]`)).toHaveAttribute('data-status', 'done');
      await expect(page.getByTestId('dv-download')).toBeVisible();
    } finally {
      await run.close();
    }
  });
});

test.describe('dossier vivant — la fleur d’un projet (téléphone 390 × 844)', () => {
  test.setTimeout(120_000);
  test.use({ viewport: PHONE });

  test('projet retenu (bandeau) et ligne de preuve : la fleur du bon projet ; les tuiles de ~12 px ne sont pas des cibles', async ({ page }) => {
    // Au travail : le bandeau des retenus est en vue, sous la montagne.
    await page.goto('/fr/dossier-vivant?t=40');
    await expect(page.getByTestId('dv-root')).toHaveAttribute('data-layout', 'compact');
    const chip = page.getByTestId('dv-kept-chip').first();
    await expect(chip).toBeVisible({ timeout: 30_000 });
    // Cible tactile : 44 px de haut, la pastille dessinée garde ses 28 px.
    expect((await chip.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    await expect(page.locator('button[data-testid="dv-tile"], button[data-testid="dv-card"]')).toHaveCount(0);
    const keptId = (await chip.getAttribute('data-kept-id'))!;
    await chip.click();
    await expect(flower(page)).toHaveAttribute('data-project-id', keptId);
    await page.keyboard.press('Escape');
    await expect(flower(page)).toHaveCount(0);
    await expect(chip).toBeFocused();

    // Remis : une ligne de preuve du dossier plein écran.
    await page.goto('/fr/dossier-vivant?t=120');
    await expect(page.getByTestId('dv-dossier')).toHaveAttribute('data-ready', '', { timeout: 30_000 });
    const row = page.locator(`[data-testid="dv-proof-row"][data-project-id="${PROOFS[2]}"]`);
    await row.click();
    await expect(flower(page)).toHaveAttribute('data-project-id', PROOFS[2]);
    await evidence(page, 'telephone-fleur-depuis-ligne-de-preuve.png');
  });
});
