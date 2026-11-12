#!/usr/bin/env node
/**
 * montagne-art.mjs — convertit les illustrations Eikon de la page /montagne en webp servis par
 * le front (public/montagne/art/).
 *
 * QUOI : deux fonds (téléphone en portrait, grand écran en paysage) et une couverture par carte
 * du sommet. Les PNG sources sortent d'Eikon (~2,5 Mo chacun) et ne vivent pas dans le dépôt :
 * seul le webp optimisé y entre.
 * POURQUOI un script et pas une conversion à la main : refaire une illustration = relancer cette
 * commande, avec les mêmes cadrages et les mêmes budgets de poids — rien à se rappeler.
 * Budgets (vérifiés ici, l'exécution échoue au-delà) : couverture < 200 Ko, fond < 350 Ko. Le
 * fond est la première image de la page sur téléphone, d'où un budget serré.
 *
 * COMMENT :
 *  - fonds : l'or d'Eikon est un jaune saturé, plus vif que l'or du site (#d9b45f) ; on le
 *    désature et l'assombrit un peu À LA CONVERSION (pas en CSS : un `filter` sur une image plein
 *    écran se recalcule à chaque peinture, sur téléphone c'est du défilement qui saccade) ;
 *  - couvertures : réduites à 960 px de large — une carte fait au plus 236 px CSS (grand écran,
 *    1x) ou 168 px en 3x sur téléphone (504 px) : 960 couvre les deux avec marge ;
 *  - grove_engine : le sujet occupe les deux tiers gauches de l'illustration (le tiers droit est
 *    du ciel vide) ; on recadre sur le sujet, sinon la carte montre surtout du noir.
 *
 * Usage : node scripts/montagne-art.mjs <dossier des PNG Eikon>
 */

import { mkdirSync, statSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

import sharp from 'sharp';

const SOURCE = process.argv[2];
if (!SOURCE) {
  console.error('usage : node scripts/montagne-art.mjs <dossier des PNG Eikon>');
  process.exit(1);
}
const OUT = path.join(path.dirname(fileURLToPath(import.meta.url)), '../public/montagne/art');

const KB = 1024;
const COVER_BUDGET = 200 * KB;
const BACKDROP_BUDGET = 350 * KB;

/** Ton des fonds : or ramené vers celui du site, nuit un peu plus profonde. */
const BACKDROP_TONE = { saturation: 0.72, brightness: 0.88 };

/** Fichiers à produire : source, sortie, recadrage éventuel, largeur, qualité, budget. */
const JOBS = [
  { src: 'fond-desktop.png', out: 'fond-desktop.webp', width: 1536, quality: 82, budget: BACKDROP_BUDGET, tone: true },
  { src: 'fond-mobile.png', out: 'fond-mobile.webp', width: 1024, quality: 82, budget: BACKDROP_BUDGET, tone: true },
  // Recadrage sur le sujet (station, îles, vaisseau), cf. en-tête.
  { src: 'cover-grove_engine.png', out: 'cover-grove_engine.webp', width: 960, quality: 78, budget: COVER_BUDGET, crop: { left: 0, top: 0, width: 1120, height: 1024 } },
  { src: 'cover-melodicode.png', out: 'cover-melodicode.webp', width: 960, quality: 78, budget: COVER_BUDGET },
  { src: 'cover-theomen.png', out: 'cover-theomen.webp', width: 960, quality: 78, budget: COVER_BUDGET },
  { src: 'cover-maicivy.png', out: 'cover-maicivy.webp', width: 960, quality: 78, budget: COVER_BUDGET },
];

mkdirSync(OUT, { recursive: true });
let failed = false;
for (const job of JOBS) {
  let img = sharp(path.join(SOURCE, job.src));
  if (job.crop) img = img.extract(job.crop);
  if (job.tone) img = img.modulate(BACKDROP_TONE);
  const target = path.join(OUT, job.out);
  // effort 6 : encodage plus lent, fichier plus léger — un coût payé une fois, ici.
  const info = await img.resize({ width: job.width, withoutEnlargement: true }).webp({ quality: job.quality, effort: 6 }).toFile(target);
  const size = statSync(target).size;
  const ok = size < job.budget;
  failed ||= !ok;
  console.log(`${ok ? '✓' : '✕'} ${job.out}  ${info.width}×${info.height}  ${(size / KB).toFixed(1)} Ko  (budget ${job.budget / KB} Ko)`);
}
if (failed) process.exit(1);
