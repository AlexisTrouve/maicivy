import type { BackgroundInitFn } from '../types';

/**
 * Plugin "ConwayGOL" — Conway's Game of Life (B3/S23) en fond, AVEC la grille cachée.
 *
 * Parti pris : on ne montre JAMAIS le damier. Chaque cellule vivante est rendue comme un
 * glow bleu doux, désaligné de sa case par un jitter pseudo-aléatoire STABLE (hash des
 * coordonnées), et plus grand que la case → des lueurs organiques flottantes, pas un lattice.
 *
 * Pièces maîtresses :
 * - Substrat : grille Conway plus grande que l'écran (marge tampon hors-champ pour les gliders).
 * - Rendu : intensité par cellule interpolée à 60fps (fade in/out), indépendante des ticks de
 *   sim (lents). Sprite de glow pré-rendu (drawImage), bien moins cher que createRadialGradient/frame.
 * - Anti-stagnation : gliders spawnés hors-champ à intervalle aléatoire, orientés pour entrer.
 * - Interaction : la souris peint des cellules (clairsemées) → burst de vie sous le curseur.
 * - Theme-aware : additif sur sombre, blending normal + bleu dense sur clair.
 *
 * Contrat coquille : ce plugin ne touche PAS requestAnimationFrame (frame(dt) est appelé par
 * BackgroundHost). dispose() libère timer + canvas.
 */

// ----------------------------------------------------------------------------
// Constantes de réglage (tout est ici pour ajuster le rendu à l'œil)
// ----------------------------------------------------------------------------
const CELL = 16; // px (CSS) par cellule
const MARGIN = 12; // cellules de tampon hors-champ (place pour spawner/évoluer les gliders)
const TICK_MS = 125; // période de simulation Conway (~8 ticks/s) → évolution lente et calme
const FADE_IN_MS = 250; // une cellule qui naît atteint sa pleine intensité en 250ms
const FADE_OUT_MS = 700; // une cellule qui meurt s'éteint en 700ms → traînée des gliders
const GLOW_R = CELL * 0.95; // rayon de base du halo (diamètre ~1.9× la case → déborde)
const JITTER_AMP = 0.45; // amplitude du désalignement, en fraction de CELL (±0.45 case)
const SEED_DENSITY = 0.08; // densité de l'amorce initiale dans la zone visible
const PAINT_R = 2; // rayon de peinture souris (en cellules)
const PAINT_DENSITY = 0.55; // proba qu'une cellule du pinceau naisse (clairsemé > blob plein)
const GLIDER_MIN_MS = 500; // intervalle min entre deux spawns de glider (→ ~2/s)
const GLIDER_MAX_MS = 1000; // intervalle max (→ ~1/s) : 1-2 gliders par seconde
const MAX_STEPS_PER_FRAME = 4; // borne anti-spirale après un onglet en arrière-plan
const BASE_ALPHA_DARK = 0.55; // plafond d'opacité sombre (anti-cramage des amas additifs)
const BASE_ALPHA_LIGHT = 0.8; // opacité claire (blending normal, doit rester visible sur blanc)
const SPRITE_PX = 64; // résolution du sprite de glow pré-rendu

// Les 5 cellules d'un glider selon sa direction de déplacement (offsets [dx,dy], y vers le bas).
// Une seule phase par direction suffit : Conway fait le reste.
const GLIDERS: Record<string, ReadonlyArray<readonly [number, number]>> = {
  SE: [[1, 0], [2, 1], [0, 2], [1, 2], [2, 2]], // descend vers la droite
  SW: [[1, 0], [0, 1], [0, 2], [1, 2], [2, 2]], // descend vers la gauche
  NE: [[0, 0], [1, 0], [2, 0], [2, 1], [1, 2]], // monte vers la droite
  NW: [[0, 0], [1, 0], [2, 0], [0, 1], [1, 2]], // monte vers la gauche
};

const initConwayGOL: BackgroundInitFn = async (ctx) => {
  const { mount, dpr } = ctx;

  // --- Canvas 2D plein écran (le mount est fixed inset-0) ---
  const canvas = document.createElement('canvas');
  canvas.style.position = 'absolute';
  canvas.style.inset = '0';
  canvas.style.width = '100%';
  canvas.style.height = '100%';
  canvas.style.display = 'block';
  mount.appendChild(canvas);
  const c = canvas.getContext('2d')!;

  // Dimensions courantes (mises à jour au resize)
  let width = ctx.width;
  let height = ctx.height;

  // État grille — réalloués dans setupGrid()
  let cols = 0;
  let rows = 0;
  let grid = new Uint8Array(0); // 0 mort / 1 vivant (lecture courante)
  let next = new Uint8Array(0); // scratch du prochain pas
  let intensity = new Float32Array(0); // 0..1 par cellule (fondu de rendu)
  let jitterX = new Float32Array(0); // décalage stable x (px)
  let jitterY = new Float32Array(0); // décalage stable y (px)
  let sizeFactor = new Float32Array(0); // variation de taille stable par cellule

  // Hash entier déterministe → [0,1[. POURQUOI : un jitter/size STABLE par coordonnée (pas de
  // tremblement frame à frame), sans stocker de RNG. COMMENT : mélange de bits type xorshift.
  const hash = (x: number, y: number, salt: number): number => {
    let h = (x * 374761393 + y * 668265263 + salt * 2147483647) | 0;
    h = (h ^ (h >> 13)) * 1274126177;
    h = h ^ (h >> 16);
    return (h >>> 0) / 4294967296;
  };

  // (Ré)alloue la grille selon le viewport courant, précalcule jitter/taille, amorce la vie.
  const setupGrid = () => {
    cols = Math.ceil(width / CELL) + MARGIN * 2;
    rows = Math.ceil(height / CELL) + MARGIN * 2;
    const n = cols * rows;
    grid = new Uint8Array(n);
    next = new Uint8Array(n);
    intensity = new Float32Array(n);
    jitterX = new Float32Array(n);
    jitterY = new Float32Array(n);
    sizeFactor = new Float32Array(n);

    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        const i = y * cols + x;
        jitterX[i] = (hash(x, y, 1) * 2 - 1) * JITTER_AMP * CELL;
        jitterY[i] = (hash(x, y, 2) * 2 - 1) * JITTER_AMP * CELL;
        sizeFactor[i] = 0.75 + hash(x, y, 3) * 0.65; // 0.75..1.4
        // Amorce : un peu de vie clairsemée dans la zone visible uniquement.
        const visible = x >= MARGIN && x < cols - MARGIN && y >= MARGIN && y < rows - MARGIN;
        if (visible && hash(x, y, 4) < SEED_DENSITY) grid[i] = 1;
      }
    }
  };

  setupGrid();

  // --- Sprite de glow pré-rendu (drawImage >> createRadialGradient par cellule/frame) ---
  const makeSprite = (dark: boolean): HTMLCanvasElement => {
    const s = document.createElement('canvas');
    s.width = SPRITE_PX;
    s.height = SPRITE_PX;
    const g = s.getContext('2d')!;
    const r = SPRITE_PX / 2;
    const grad = g.createRadialGradient(r, r, 0, r, r, r);
    if (dark) {
      // Additif : centre clair, dégradé bleu, bord transparent.
      grad.addColorStop(0, 'rgba(191, 219, 254, 1)'); // blue-200
      grad.addColorStop(0.35, 'rgba(96, 165, 250, 0.7)'); // blue-400
      grad.addColorStop(1, 'rgba(59, 130, 246, 0)'); // blue-500 → transparent
    } else {
      // Clair : bleu dense (blending normal) pour rester visible sur fond blanc.
      grad.addColorStop(0, 'rgba(37, 99, 235, 0.9)'); // blue-600
      grad.addColorStop(0.5, 'rgba(37, 99, 235, 0.45)');
      grad.addColorStop(1, 'rgba(37, 99, 235, 0)');
    }
    g.fillStyle = grad;
    g.fillRect(0, 0, SPRITE_PX, SPRITE_PX);
    return s;
  };

  const isDark = () => document.documentElement.classList.contains('dark');
  let lastDark = isDark();
  let sprite = makeSprite(lastDark);

  // --- Pas de simulation Conway (B3/S23), double buffer, bordures mortes ---
  const step = () => {
    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        let nb = 0;
        for (let dy = -1; dy <= 1; dy++) {
          const yy = y + dy;
          if (yy < 0 || yy >= rows) continue;
          for (let dx = -1; dx <= 1; dx++) {
            if (dx === 0 && dy === 0) continue;
            const xx = x + dx;
            if (xx < 0 || xx >= cols) continue;
            nb += grid[yy * cols + xx];
          }
        }
        const i = y * cols + x;
        // Naissance sur 3 voisins ; survie sur 2 ou 3 ; mort sinon.
        next[i] = grid[i] ? (nb === 2 || nb === 3 ? 1 : 0) : nb === 3 ? 1 : 0;
      }
    }
    const tmp = grid;
    grid = next;
    next = tmp;
  };

  // Pose un glider (5 cellules) à (ox,oy) selon sa direction. Cellules hors grille ignorées.
  const placeGlider = (ox: number, oy: number, dir: keyof typeof GLIDERS) => {
    for (const [dx, dy] of GLIDERS[dir]) {
      const x = ox + dx;
      const y = oy + dy;
      if (x >= 0 && x < cols && y >= 0 && y < rows) grid[y * cols + x] = 1;
    }
  };

  // Spawn d'un glider hors-champ sur un bord aléatoire, orienté pour ENTRER dans l'écran.
  const spawnGlider = () => {
    const edge = Math.floor(Math.random() * 4); // 0 gauche, 1 droite, 2 haut, 3 bas
    const spanX = () => MARGIN + Math.floor(Math.random() * Math.max(1, cols - 2 * MARGIN));
    const spanY = () => MARGIN + Math.floor(Math.random() * Math.max(1, rows - 2 * MARGIN));
    if (edge === 0) placeGlider(MARGIN - 4, spanY(), Math.random() < 0.5 ? 'SE' : 'NE');
    else if (edge === 1) placeGlider(cols - MARGIN + 1, spanY(), Math.random() < 0.5 ? 'SW' : 'NW');
    else if (edge === 2) placeGlider(spanX(), MARGIN - 4, Math.random() < 0.5 ? 'SE' : 'SW');
    else placeGlider(spanX(), rows - MARGIN + 1, Math.random() < 0.5 ? 'NE' : 'NW');
  };

  // Timer récursif de spawn (intervalle aléatoire, comme les shooting stars de la constellation).
  let gliderTimer: ReturnType<typeof setTimeout>;
  const scheduleGlider = () => {
    const delay = GLIDER_MIN_MS + Math.random() * (GLIDER_MAX_MS - GLIDER_MIN_MS);
    gliderTimer = setTimeout(() => {
      spawnGlider();
      scheduleGlider();
    }, delay);
  };
  spawnGlider(); // un glider d'emblée pour amorcer le mouvement
  scheduleGlider();

  // --- Rendu : interpole l'intensité (fade) puis dessine les glows visibles ---
  const render = (dt: number) => {
    c.setTransform(dpr, 0, 0, dpr, 0, 0); // dessiner en px CSS
    c.clearRect(0, 0, width, height);

    // Bascule de thème à la volée (ThemeProvider toggle la classe `dark`).
    const dark = isDark();
    if (dark !== lastDark) {
      sprite = makeSprite(dark);
      lastDark = dark;
    }
    c.globalCompositeOperation = dark ? 'lighter' : 'source-over';
    const baseAlpha = dark ? BASE_ALPHA_DARK : BASE_ALPHA_LIGHT;

    const inStep = dt / FADE_IN_MS;
    const outStep = dt / FADE_OUT_MS;

    for (let y = 0; y < rows; y++) {
      const sy = (y - MARGIN) * CELL;
      for (let x = 0; x < cols; x++) {
        const i = y * cols + x;
        const target = grid[i] ? 1 : 0;
        let cur = intensity[i];
        // Interpolation vers la cible (naissance = montée rapide, mort = descente lente).
        if (cur < target) cur = Math.min(target, cur + inStep);
        else if (cur > target) cur = Math.max(target, cur - outStep);
        intensity[i] = cur;
        if (cur <= 0.01) continue;

        const sx = (x - MARGIN) * CELL;
        // Skip dessin si totalement hors écran (les cellules tampon n'ont pas à s'afficher).
        if (sx < -GLOW_R || sx > width + GLOW_R || sy < -GLOW_R || sy > height + GLOW_R) continue;

        const r = GLOW_R * sizeFactor[i];
        c.globalAlpha = cur * baseAlpha;
        // Centre de case + jitter stable → glow désaligné, débordant.
        const cx = sx + CELL / 2 + jitterX[i];
        const cy = sy + CELL / 2 + jitterY[i];
        c.drawImage(sprite, cx - r, cy - r, r * 2, r * 2);
      }
    }
    c.globalAlpha = 1;
    c.globalCompositeOperation = 'source-over';
  };

  // --- Boucle appelée par la coquille : avance la sim au bon rythme, puis rend ---
  let acc = 0;
  const frame = (dt: number) => {
    acc += dt;
    let steps = 0;
    while (acc >= TICK_MS && steps < MAX_STEPS_PER_FRAME) {
      step();
      acc -= TICK_MS;
      steps++;
    }
    if (acc > TICK_MS) acc = TICK_MS; // évite l'accumulation après un long gel d'onglet
    render(dt);
  };

  // --- Resize : réajuste le canvas + reconstruit la grille (l'état n'est pas préservé) ---
  const resize = (w: number, h: number) => {
    width = w;
    height = h;
    canvas.width = Math.floor(w * dpr);
    canvas.height = Math.floor(h * dpr);
    setupGrid();
    spawnGlider();
  };
  // Dimension initiale du buffer canvas (resize() n'est appelé qu'au resize réel).
  canvas.width = Math.floor(width * dpr);
  canvas.height = Math.floor(height * dpr);

  // --- Souris : peindre la vie (cellules clairsemées dans un rayon sous le curseur) ---
  const onPointerMove = (clientX: number, clientY: number) => {
    const gx = Math.floor(clientX / CELL) + MARGIN;
    const gy = Math.floor(clientY / CELL) + MARGIN;
    for (let dy = -PAINT_R; dy <= PAINT_R; dy++) {
      for (let dx = -PAINT_R; dx <= PAINT_R; dx++) {
        if (dx * dx + dy * dy > PAINT_R * PAINT_R) continue;
        if (Math.random() > PAINT_DENSITY) continue;
        const x = gx + dx;
        const y = gy + dy;
        if (x >= 0 && x < cols && y >= 0 && y < rows) grid[y * cols + x] = 1;
      }
    }
  };

  // --- Libération : timer + retrait du canvas (pas de ressources GPU à disposer en 2D) ---
  const dispose = () => {
    clearTimeout(gliderTimer);
    if (canvas.parentNode === mount) mount.removeChild(canvas);
  };

  return { frame, resize, onPointerMove, dispose };
};

export default initConwayGOL;
