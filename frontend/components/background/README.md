# Fonds animés — architecture plug-and-play

Système de fonds animés branchables pour maicivy. Un fond ("plugin") tourne à la
fois, choisi aléatoirement à chaque visite ou explicitement par le visiteur via un
sélecteur. Ajouter un fond = écrire un plugin + une ligne dans le manifeste.

## Vue d'ensemble

```
lib/backgrounds/
  types.ts        → le contrat (BackgroundInitFn, BackgroundInstance…)
  registry.ts     → manifeste léger { id, name, mobile, load } + helpers
  plugins/
    constellation.ts → fond 3D Three.js (étoiles + traits + shooting stars)
    conwaygol.ts     → Conway's Game of Life (grille cachée, glows désalignés)

components/background/
  BackgroundProvider.tsx → état de sélection partagé (préférence vs fond actif)
  BackgroundHost.tsx     → la COQUILLE : invariantes + cycle de vie d'un plugin
  BackgroundSwitcher.tsx → le sélecteur top-right (dans le Header)
```

### Deux couches

1. **La coquille** (`BackgroundHost`) possède le conteneur `fixed inset-0` et centralise
   **une seule fois** toutes les invariantes communes :
   - skip mobile (selon `mobile` du plugin), `prefers-reduced-motion`,
   - defer post-LCP (seulement pendant la fenêtre de chargement initial),
   - RAF loop unique (appelle `frame(dt)`), resize, `pointermove` (forwardé),
   - fade-in CSS, et surtout le **dispose complet** au switch/unmount (zéro fuite).

2. **Les plugins fins** n'écrivent QUE leur rendu via un `BackgroundInitFn`. Ils ne
   peuvent pas oublier une invariante : elles ne sont pas dans leur scope. C'est ce
   qui garantit le "plug & play sans rien casser".

## Ajouter un fond

1. Créer `lib/backgrounds/plugins/<id>.ts` qui **default-exporte** un `BackgroundInitFn` :

   ```ts
   import type { BackgroundInitFn } from '../types';

   const initMyBg: BackgroundInitFn = async (ctx) => {
     // ctx = { mount, width, height, dpr, reducedMotion }
     const canvas = document.createElement('canvas');
     ctx.mount.appendChild(canvas);
     // … setup …
     return {
       frame: (dtMs) => { /* dessin par frame */ },
       resize: (w, h) => { /* ajuste buffers */ },
       onPointerMove: (x, y) => { /* optionnel : interactif */ },
       dispose: () => { /* libère TOUT (timers, GPU, DOM) */ },
     };
   };
   export default initMyBg;
   ```

2. Ajouter une entrée dans `registry.ts` :

   ```ts
   { id: 'mybg', name: 'My BG', mobile: true, load: () => import('./plugins/mybg') }
   ```

Le sélecteur et le host se mettent à jour automatiquement depuis le manifeste.

### Règles à respecter dans un plugin

- **Ne PAS** appeler `requestAnimationFrame` : la coquille gère la RAF loop, mets la
  logique par-frame dans `frame(dt)`.
- **Tout** ce qui est alloué doit être libéré dans `dispose()` (timers, WebGL, nodes DOM).
- `import()` dynamique des grosses deps (ex. `await import('three')`) dans `init` → hors
  bundle initial.
- Faible opacité, tempo calme : c'est un fond derrière du contenu, pas le sujet.
- **Theme-aware** si possible (lisible en clair ET sombre — voir ConwayGOL).

## Modèle de sélection

Géré par `BackgroundProvider`. Deux notions distinctes :

- **`preference`** : ce que l'utilisateur choisit / ce que le sélecteur coche
  (`random` | `none` | `<pluginId>`). Persistée dans `localStorage` (`maicivy_bg`).
- **`activeId`** : le fond CONCRET rendu. Quand `preference = random`, c'est un plugin
  tiré au sort.

Ordre de résolution au chargement (client) :

1. `?bg=<id>` dans l'URL → override one-shot (ne persiste pas ; pratique pour partager/tester).
2. Sinon `localStorage` → le choix verrouillé du visiteur.
3. Sinon le défaut : **aléatoire** ; mais **`none` sous `prefers-reduced-motion`** (défaut calme).

**`prefers-reduced-motion` ne change QUE le défaut.** Un choix explicite (URL ou
localStorage), fond animé inclus, est toujours honoré, et le sélecteur reste **toujours
visible** → l'utilisateur garde le contrôle. (Régression corrigée : au départ reduced-motion
forçait `none` + masquait le sélecteur → plus aucun fond ni contrôle.)

## Plugins existants

### `constellation` (`mobile: false`)

Fond 3D Three.js : 100 points en sphère qui driftent, reliés par des traits (topologie
figée à l'init, coordonnées resynchronisées chaque frame pour suivre le drift), + shooting
stars. Three.js trop lourd sur mobile → skip mobile.

### `conwaygol` (`mobile: true`)

Conway's Game of Life (règles B3/S23) avec un parti pris visuel : **on ne montre jamais la
grille**.

- **Substrat caché** : une grille Conway tourne en dessous, plus grande que l'écran (marge
  tampon hors-champ).
- **Peau visible** : chaque cellule vivante = un **glow bleu doux**, dessiné à
  `centre_cellule + jitter`, où le jitter est un décalage pseudo-aléatoire **stable par
  cellule** (hash des coordonnées). Le lattice régulier est donc cassé volontairement → des
  lueurs organiques flottantes, pas un damier. Les glows sont **plus grands qu'une case**
  (~1.9×) et se chevauchent → champ lumineux continu.
- **Fade in/out** : chaque cellule a une intensité interpolée à 60fps (indépendante des ticks
  de sim, lents ~8/s) → naissance qui éclôt, mort qui s'éteint. Les gliders laissent une
  traînée de glows déclinants.
- **Anti-stagnation = gliders** : à intervalle aléatoire, un glider est posé **hors-champ**
  sur un bord tiré au hasard, orienté pour entrer dans l'écran. Il le traverse en diagonale
  et meurt à la bordure opposée. Pas de drip ambiant : champ calme traversé de lueurs.
- **Souris = peindre la vie** : un rayon de cellules naît (clairsemé) sous le curseur quand
  on bouge → burst de glow + matière que Conway fait évoluer.
- **Theme-aware** : additif (`lighter`) sur sombre, blending normal + bleu plus dense sur
  clair (sinon l'additif s'efface sur fond blanc). Opacité de base plafonnée pour que les
  amas denses ne "crament" pas vers le blanc.

Un moteur plus vivant (Brian's Brain) est prévu comme plugin séparé plus tard.

### `fractal` (`mobile: false`)

Fractale **Mandelbrot↔Julia hybride** en zoom (plongée), via un fragment shader WebGL.

- **Hybride** : l'itération `z = z² + c` interpole entre Mandelbrot (`z₀=0, c=pixel`) et Julia
  (`z₀=pixel, c=graine`) via `uMorph`, qui oscille lentement → la structure se métamorphose.
- **Graine Julia** : dérive sur une courbe de Lissajous lente → mutation perpétuelle.
- **Plongée** : `uScale` décroît sur un cycle ~22s, avec cross-fade aux bords pour boucler sans
  couture (le `float` 32-bit casse vers ~1e-6 → on reste dans une profondeur sûre, pas un zoom
  *littéralement* infini).
- **Souris** : le centre du zoom glisse vers le point sous le curseur → on dirige la plongée.
- **Look** : palette vive (cosine palette), theme-aware. L'**intérieur** de l'ensemble est rendu
  **transparent** (les blobs aurora restent visibles) — seuls les filaments colorés du bord
  s'affichent.
- **Perf** : le plus lourd des trois. Résolution interne réduite (`RES_SCALE`), DPR plafonné,
  itérations bornées (montent avec la profondeur). `preserveDrawingBuffer:true` pour permettre
  la lecture pixels en test.

## Tests E2E

`tests/e2e/background-switcher.spec.ts` clique réellement le sélecteur : visibilité,
options, montage/démontage du `<canvas>`, persistance localStorage, comportement
reduced-motion, et rendu non-vide de ConwayGOL après peinture (échantillonnage de pixels).
