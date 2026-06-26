import type { BackgroundInitFn } from '../types';

/**
 * Plugin "Schrödinger" — fond DISCRET : des fonctions d'onde ψ(x,t) quantiques + l'équation.
 *
 * QUOI : quelques "voies" horizontales, chacune un paquet d'onde gaussien qui oscille (slosh) comme
 *   une particule dans un puits. On rend :
 *     - la DENSITÉ DE PROBABILITÉ |ψ|² → un nuage violet ("où est la particule") ;
 *     - la fonction d'onde COMPLEXE Re(ψ) ET Im(ψ) (en quadrature, 90°) → deux ondes qui frétillent
 *       dans l'enveloppe, COLORÉES PAR LA PHASE (domain coloring : la teinte = arg ψ) ;
 *     - des TRAÎNÉES (motion-blur) → le paquet laisse un sillage fantôme en glissant ;
 *     - l'équation de Schrödinger dépendante du temps, en filigrane central.
 * INTERACTION — la MESURE : survoler une voie = "observer" → la fonction d'onde s'effondre (collapse)
 *   en un pic localisé sous le curseur, puis se re-disperse quand on s'éloigne. L'effet observateur.
 * POURQUOI : reste sobre (traits fins, faible opacité, ondes localisées dans l'enveloppe). Le paquet
 *   est analytique (état ~cohérent) → oscille/respire sans diverger → boucle infinie propre.
 * COMMENT : Canvas 2D (léger, mobile OK) pour ψ + un <div> pour la formule. Aucune RAF (la coquille
 *   appelle frame(dt)). dispose() retire canvas + div.
 *
 * Physique (qualitative, pour le look) :
 *   - xc(t) = x₀ + A·cos(ωt+φ) → oscillation dans un puits ; vitesse ∝ −sin(ωt+φ).
 *   - ψ = env(x)·e^{i(k·(x−xc)+θ(t))} ; k ∝ vitesse → le frétillement gèle aux rebroussements (p=0),
 *     file au centre (p max). Re = env·cos, Im = env·sin, |ψ|² = env². Phase = k·(x−xc)+θ → la teinte.
 */

// ----------------------------------------------------------------------------
// Réglages (tout est ici pour ajuster le rendu à l'œil)
// ----------------------------------------------------------------------------
const TAU = Math.PI * 2;
const DENSITY_FRAC = 0.42;   // hauteur du nuage |ψ|², en fraction de l'espacement entre voies
const WAVE_FRAC = 0.32;      // amplitude des ondes Re/Im, en fraction de l'espacement
const SLOSH_FRAC = 0.2;      // amplitude d'oscillation horizontale, en fraction de la largeur
const DENSITY_STEP = 5;      // pas d'échantillonnage du nuage |ψ|² (px CSS)
const WAVE_STEP = 6;         // pas des ondes colorées (un peu + grossier → moins de segments)
const MIN_ENV = 0.05;        // on ne trace l'onde QUE là où l'enveloppe est sensible → onde localisée

// Coloration par phase (domain coloring) — plage de teinte limitée pour rester dans la veine "froide".
const HUE_MIN = 200;         // cyan
const HUE_SPAN = 150;        // → jusqu'à ~350 (magenta/violet) ; la teinte défile avec la phase
const HUE_BUCKET = 14;       // quantification de teinte (deg) → on BATCH les segments de même bucket
const IM_HUE_SHIFT = 30;     // Im légèrement décalé en teinte pour le distinguer de Re

// Traînées : on n'efface pas net, on atténue (destination-out) → sillage qui s'estompe.
const TRAIL_FADE = 0.1;      // 0..1, + grand = traînée + courte

// Mesure / collapse : survoler une voie l'effondre en un pic localisé sous le curseur.
const COLLAPSE_GAIN = 1.4;   // proximité×énergie → quantité de collapse (clampé à 1)
const COLLAPSE_MIN_SIGMA = 0.18; // largeur du paquet effondré (fraction de sa largeur normale)
const COLLAPSE_AMP_BOOST = 1.6;  // sur-amplitude du pic au collapse
const MOUSE_VSIGMA = 84;     // portée verticale de l'interaction (px CSS)
const MOUSE_DECAY = 1.7;     // décroissance/s de l'énergie souris quand on ne bouge plus

// Opacités de base selon le thème.
const DENSITY_FILL_DARK = 0.1, DENSITY_FILL_LIGHT = 0.09;
const WAVE_ALPHA_DARK = 0.42, WAVE_ALPHA_LIGHT = 0.5;
const FORMULA_ALPHA_DARK = 0.14, FORMULA_ALPHA_LIGHT = 0.12;

const REDUCED_MOTION_SCALE = 0.18; // tempo quasi figé sous prefers-reduced-motion (défensif)

// Équation de Schrödinger dépendante du temps (1D, 1 particule) — affichée en filigrane.
const SCHRODINGER_EQ = 'iℏ ∂ψ/∂t = −ℏ²/2m ∂²ψ/∂x² + V(x)ψ';

const initSchrodinger: BackgroundInitFn = async (ctx) => {
  const { mount, dpr, reducedMotion } = ctx;

  // --- Canvas 2D plein écran (le mount est fixed inset-0, pointer-events-none) ---
  const canvas = document.createElement('canvas');
  canvas.style.position = 'absolute';
  canvas.style.inset = '0';
  canvas.style.width = '100%';
  canvas.style.height = '100%';
  canvas.style.display = 'block';
  mount.appendChild(canvas);
  const c = canvas.getContext('2d')!;

  let width = ctx.width;
  let height = ctx.height;
  canvas.width = Math.floor(width * dpr);
  canvas.height = Math.floor(height * dpr);

  // --- Filigrane formule : un <div> au-dessus du canvas (rendu net via la police) ---
  const formula = document.createElement('div');
  formula.textContent = SCHRODINGER_EQ;
  formula.setAttribute('aria-hidden', 'true');
  Object.assign(formula.style, {
    position: 'absolute',
    top: '46%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    whiteSpace: 'nowrap',
    fontStyle: 'italic',
    fontFamily: '"Cambria Math", "STIX Two Math", Cambria, "Times New Roman", serif',
    letterSpacing: '0.02em',
    pointerEvents: 'none',
    userSelect: 'none',
    fontWeight: '500',
  } as Partial<CSSStyleDeclaration>);
  mount.appendChild(formula);

  // Nombre de voies : 2 sur petit écran (perf), 3 sinon.
  const laneCount = width < 700 ? 2 : 3;

  // Paramètres STABLES par voie (dérivés de l'index → pas de RNG, pas de tremblement).
  interface Lane {
    omega: number;       // pulsation du slosh (rad/s)
    phi: number;         // phase initiale (désynchronise les voies)
    sigmaFrac: number;   // largeur du paquet (fraction de largeur écran)
    localCycles: number; // nb d'oscillations visibles dans le paquet (à vitesse max)
    carrierW: number;    // rotation de phase interne (rad/s)
    depth: number;       // 0.7..1 → opacité (profondeur visuelle)
  }
  const lanes: Lane[] = Array.from({ length: laneCount }, (_, i) => {
    const dir = i % 2 === 0 ? 1 : -1;
    return {
      omega: dir * (0.26 + i * 0.06),
      phi: i * 2.1,
      sigmaFrac: 0.05 + i * 0.008,
      localCycles: 6 + i * 1.5,
      carrierW: 0.6 + i * 0.15,
      depth: 0.72 + (i / Math.max(1, laneCount - 1)) * 0.28,
    };
  });

  // Géométrie dépendante du viewport (recalculée au resize).
  let spacing = height / (laneCount + 1);
  const recomputeGeometry = () => {
    spacing = height / (laneCount + 1);
    formula.style.fontSize = `${Math.min(width * 0.052, 60)}px`;
  };
  recomputeGeometry();

  let t = 0;
  const motionScale = reducedMotion ? REDUCED_MOTION_SCALE : 1;

  // État souris : position + énergie qui décroît quand on arrête de bouger.
  let mouseX = -1, mouseY = -1, mouseEnergy = 0;

  const isDark = () => document.documentElement.classList.contains('dark');
  let lastDark: boolean | null = null;

  // Couleur HSL d'une teinte (phase), selon thème — clair = plus sombre/saturé pour rester visible.
  const hueColor = (hue: number, dark: boolean, alpha: number): string => {
    const h = ((hue % 360) + 360) % 360;
    return dark ? `hsla(${h},85%,68%,${alpha})` : `hsla(${h},75%,45%,${alpha})`;
  };

  // --- Rendu d'une frame ---
  const render = () => {
    c.setTransform(dpr, 0, 0, dpr, 0, 0); // dessiner en px CSS

    // TRAÎNÉES : on atténue l'image précédente (destination-out réduit l'alpha) au lieu d'effacer net.
    // → le canvas reste transparent (l'aurora transparaît) et les paquets laissent un sillage.
    c.globalCompositeOperation = 'destination-out';
    c.fillStyle = `rgba(0,0,0,${TRAIL_FADE})`;
    c.fillRect(0, 0, width, height);

    const dark = isDark();
    c.globalCompositeOperation = dark ? 'lighter' : 'source-over';
    c.lineJoin = 'round';
    c.lineCap = 'round';

    const densCol = dark ? '167,139,250' : '124,58,237'; // violet-400 / violet-600 (le nuage)
    const fillA = dark ? DENSITY_FILL_DARK : DENSITY_FILL_LIGHT;
    const waveA = dark ? WAVE_ALPHA_DARK : WAVE_ALPHA_LIGHT;

    const sloshAmp = width * SLOSH_FRAC;
    const densityAmp = spacing * DENSITY_FRAC;
    const waveAmp = spacing * WAVE_FRAC;
    const x0 = width * 0.5;
    const hasMouse = mouseEnergy > 0.004 && mouseX >= 0;
    let maxCollapse = 0; // pour faire "flasher" la formule au moment de la mesure

    for (let i = 0; i < lanes.length; i++) {
      const ln = lanes[i];
      const baseY = spacing * (i + 1);
      const angle = ln.omega * t + ln.phi;
      const analyticXc = x0 + sloshAmp * Math.cos(angle);
      const vDir = -Math.sin(angle); // direction/intensité de la quantité de mouvement ∈ [-1,1]

      // MESURE : proximité verticale × énergie souris → quantité de collapse (0 = libre, 1 = effondré).
      const vProx = hasMouse
        ? Math.exp(-((baseY - mouseY) * (baseY - mouseY)) / (2 * MOUSE_VSIGMA * MOUSE_VSIGMA))
        : 0;
      const collapse = Math.min(1, vProx * mouseEnergy * COLLAPSE_GAIN);
      if (collapse > maxCollapse) maxCollapse = collapse;

      // Effondrement : le paquet se localise sous le curseur et se resserre, puis se re-disperse.
      const xc = analyticXc + (mouseX - analyticXc) * collapse;
      const sigmaBase = width * ln.sigmaFrac * (1 + 0.12 * Math.sin(angle * 0.8));
      const sigma = sigmaBase * (1 - (1 - COLLAPSE_MIN_SIGMA) * collapse);
      const inv2s2 = 1 / (2 * sigma * sigma);
      const ampBoost = 1 + COLLAPSE_AMP_BOOST * collapse; // pic plus haut au collapse
      const k = ((ln.localCycles * TAU) / (4 * sigma)) * vDir; // nombre d'onde local ∝ vitesse
      const carrierPhase = -t * ln.carrierW;

      // --- |ψ|² : nuage de densité (aire remplie, violet — le "où est la particule") ---
      c.beginPath();
      c.moveTo(0, baseY);
      for (let x = 0; x <= width; x += DENSITY_STEP) {
        const dxc = x - xc;
        const env = Math.exp(-(dxc * dxc) * inv2s2);
        c.lineTo(x, baseY - env * env * densityAmp * ampBoost);
      }
      c.lineTo(width, baseY);
      c.closePath();
      c.fillStyle = `rgba(${densCol},${fillA * ln.depth})`;
      c.fill();

      // --- Re(ψ) et Im(ψ) : ondes en quadrature, COLORÉES PAR LA PHASE, localisées dans l'enveloppe.
      // quad=0 → Re (cos) ; quad=1 → Im (sin, décalé en teinte). Batch par bucket de teinte (perf).
      const drawWave = (quad: 0 | 1) => {
        const hueShift = quad ? IM_HUE_SHIFT : 0;
        let open = false;
        let prevBucket = 0;
        for (let x = 0; x <= width; x += WAVE_STEP) {
          const dxc = x - xc;
          const env = Math.exp(-(dxc * dxc) * inv2s2);
          if (env < MIN_ENV) { // hors de l'enveloppe → couper l'onde (elle est localisée)
            if (open) { c.stroke(); open = false; }
            continue;
          }
          const phase = k * dxc + carrierPhase;
          const val = quad ? Math.sin(phase) : Math.cos(phase);
          const y = baseY - env * val * waveAmp * ampBoost;
          const ph01 = ((phase / TAU) % 1 + 1) % 1;
          const bucket = Math.floor((HUE_MIN + ph01 * HUE_SPAN + hueShift) / HUE_BUCKET);
          if (!open) {
            c.beginPath(); c.moveTo(x, y); open = true; prevBucket = bucket;
          } else if (bucket !== prevBucket) {
            // Fin d'un run de teinte : on trace ce run dans sa couleur, on repart du point courant.
            c.lineTo(x, y);
            c.strokeStyle = hueColor(prevBucket * HUE_BUCKET + HUE_BUCKET / 2, dark, waveA * ln.depth);
            c.stroke();
            c.beginPath(); c.moveTo(x, y); prevBucket = bucket;
          } else {
            c.lineTo(x, y);
          }
        }
        if (open) {
          c.strokeStyle = hueColor(prevBucket * HUE_BUCKET + HUE_BUCKET / 2, dark, waveA * ln.depth);
          c.stroke();
        }
      };
      c.lineWidth = 1.1;
      drawWave(0);
      drawWave(1);
    }

    c.globalCompositeOperation = 'source-over';

    // Filigrane : couleur au switch de thème + pulsation douce + FLASH au moment d'une mesure.
    if (dark !== lastDark) {
      formula.style.color = dark ? 'rgb(196,181,253)' : 'rgb(91,33,182)'; // violet-300 / violet-800
      lastDark = dark;
    }
    const pulse = 0.7 + 0.3 * (0.5 + 0.5 * Math.sin(t * 0.6));
    const baseFA = dark ? FORMULA_ALPHA_DARK : FORMULA_ALPHA_LIGHT;
    formula.style.opacity = `${baseFA * pulse * (1 + 0.7 * maxCollapse)}`;
  };

  // --- Boucle appelée par la coquille ---
  const frame = (dtMs: number) => {
    t += (dtMs / 1000) * motionScale;
    if (mouseEnergy > 0) {
      mouseEnergy *= Math.exp(-MOUSE_DECAY * (dtMs / 1000));
      if (mouseEnergy < 0.004) mouseEnergy = 0;
    }
    render();
  };

  // --- Resize : réajuste le buffer + la géométrie (état temporel continu) ---
  const resize = (w: number, h: number) => {
    width = w;
    height = h;
    canvas.width = Math.floor(w * dpr);
    canvas.height = Math.floor(h * dpr);
    recomputeGeometry();
  };

  // --- Souris : recharge l'énergie de mesure + mémorise la position ---
  const onPointerMove = (x: number, y: number) => {
    mouseX = x;
    mouseY = y;
    mouseEnergy = 1;
  };

  // --- Libération : retrait du canvas + du filigrane ---
  const dispose = () => {
    if (canvas.parentNode === mount) mount.removeChild(canvas);
    if (formula.parentNode === mount) mount.removeChild(formula);
  };

  return { frame, resize, onPointerMove, dispose };
};

export default initSchrodinger;
