// Import de type uniquement — ne charge PAS Three.js dans le bundle de ce module.
// Three.js est import()é dynamiquement dans init() (voir plus bas).
import type { Line, LineBasicMaterial, BufferGeometry } from 'three';
import type { BackgroundInitFn } from '../types';

/**
 * Plugin "constellation" — fond 3D Three.js (étoiles + traits + shooting stars).
 *
 * Architecture (refactorée en plugin de la coquille <BackgroundHost>) :
 * - La coquille possède le conteneur, le defer post-LCP, le skip mobile, le reduced-motion,
 *   la RAF loop et le fade-in. Ce plugin n'écrit QUE le rendu.
 * - import('three') est dynamique dans init() : 0 impact SSR/LCP, three chargé seulement
 *   quand ce fond est sélectionné.
 * - Un THREE.Group regroupe tous les objets et tourne lentement sur les 3 axes.
 * - 100 points driftent lentement dans une sphère de rayon 8 unités.
 * - Topologie des connexions calculée UNE SEULE FOIS à l'init (0 clignotement), mais les
 *   coordonnées des extrémités sont resynchronisées chaque frame pour suivre le drift des points.
 * - Canvas transparent (alpha:true, clearColor alpha=0) — les blobs aurora restent visibles dessous.
 *
 * Optimisations perf :
 * - distance² partout (Math.sqrt évité dans la hot path O(n²))
 * - Topologie des connexions calculée à l'init (0 clignotement) ; resync des extrémités
 *   en O(nb_traits)/frame — simples recopies mémoire, pas le O(n²) du calcul de topologie
 * - Trails des points normaux supprimés (coût élevé, effet quasi invisible)
 */
const initConstellation: BackgroundInitFn = async (ctx) => {
  const THREE = await import('three');
  const { mount, width, height, dpr } = ctx;

  // =========================================================
  // SCENE SETUP
  // =========================================================
  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 100);
  camera.position.z = 12;

  const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  renderer.setSize(width, height);
  renderer.setPixelRatio(dpr); // dpr déjà plafonné par la coquille
  // alpha=0 → transparent, les blobs aurora CSS sont visibles en dessous
  renderer.setClearColor(0x000000, 0);
  mount.appendChild(renderer.domElement);

  // =========================================================
  // POINTS (étoiles) — 100 points distribués en sphère
  // =========================================================
  const POINT_COUNT = 100;
  const positions = new Float32Array(POINT_COUNT * 3);
  const velocities: Array<{ x: number; y: number; z: number }> = [];

  for (let i = 0; i < POINT_COUNT; i++) {
    // Distribution sphérique uniforme (méthode angles polaires)
    const theta = Math.random() * Math.PI * 2;
    const phi = Math.acos(2 * Math.random() - 1);
    const r = 3 + Math.random() * 5; // rayon entre 3 et 8 unités

    positions[i * 3] = r * Math.sin(phi) * Math.cos(theta);
    positions[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta);
    positions[i * 3 + 2] = r * Math.cos(phi);

    // Vélocité de drift très lente — chaque point se déplace librement
    velocities.push({
      x: (Math.random() - 0.5) * 0.002,
      y: (Math.random() - 0.5) * 0.002,
      z: (Math.random() - 0.5) * 0.002,
    });
  }

  const pointsGeo = new THREE.BufferGeometry();
  pointsGeo.setAttribute('position', new THREE.BufferAttribute(positions, 3));

  // --- Texture glow via canvas 2D ---
  // Dégradé radial blanc→bleu→transparent : simuler un halo lumineux autour de chaque étoile.
  const glowCanvas = document.createElement('canvas');
  glowCanvas.width = 64;
  glowCanvas.height = 64;
  const glowCtx = glowCanvas.getContext('2d')!;

  const gradient = glowCtx.createRadialGradient(32, 32, 0, 32, 32, 32);
  gradient.addColorStop(0, 'rgba(255, 255, 255, 1.0)'); // centre : blanc pur
  gradient.addColorStop(0.3, 'rgba(147, 197, 253, 0.8)'); // bleu clair (93c5fd)
  gradient.addColorStop(0.7, 'rgba(59, 130, 246, 0.3)'); // bleu primaire, semi-transparent
  gradient.addColorStop(1.0, 'rgba(0, 0, 0, 0)'); // bord : transparent

  glowCtx.fillStyle = gradient;
  glowCtx.fillRect(0, 0, 64, 64);

  const glowTexture = new THREE.CanvasTexture(glowCanvas);

  const pointsMat = new THREE.PointsMaterial({
    color: 0xffffff, // blanc — la teinte finale vient de la texture
    size: 0.35, // plus grand pour que le halo glow soit visible
    map: glowTexture,
    transparent: true,
    opacity: 0.9,
    blending: THREE.AdditiveBlending, // mode lumière — zones denses = plus lumineux
    depthWrite: false, // évite les artefacts de profondeur avec l'additive blending
    sizeAttenuation: true,
  });

  const pointsMesh = new THREE.Points(pointsGeo, pointsMat);

  // =========================================================
  // GROUP — rotation globale lente sur les 3 axes
  // =========================================================
  const group = new THREE.Group();
  group.add(pointsMesh);
  scene.add(group);

  // =========================================================
  // LIGNES DE CONNEXION — buffer pré-alloué
  // =========================================================
  const MAX_LINES = POINT_COUNT * 10;
  const linePositions = new Float32Array(MAX_LINES * 6);
  const lineGeo = new THREE.BufferGeometry();
  lineGeo.setAttribute('position', new THREE.BufferAttribute(linePositions, 3));

  const lineMat = new THREE.LineBasicMaterial({
    color: 0x3b82f6, // Tailwind blue-500
    transparent: true,
    opacity: 0.15,
  });

  const lineSegments = new THREE.LineSegments(lineGeo, lineMat);
  group.add(lineSegments);

  // =========================================================
  // TOPOLOGIE DES CONNEXIONS — figée une seule fois à l'init.
  // QUOI : on décide QUI est relié à QUI (les paires d'indices de points), pas les
  //   coordonnées. Les coordonnées des extrémités, elles, sont resynchronisées chaque
  //   frame dans frame() (voir « Sync des traits »).
  // POURQUOI figer la topologie : recalculer les paires chaque frame (O(n²)) ferait
  //   clignoter les liens dès qu'un point franchit le seuil de 2.5u. Figer = 0 clignotement.
  // POURQUOI NE PAS figer les coordonnées : les points driftent en continu (cumulatif).
  //   Des extrémités figées au init se détachent visiblement des étoiles au bout de quelques
  //   secondes — c'était le bug de désync étoiles/traits.
  // =========================================================
  const lineConnections: Array<[number, number]> = [];
  for (let i = 0; i < POINT_COUNT; i++) {
    for (let j = i + 1; j < POINT_COUNT; j++) {
      const dx = positions[i * 3] - positions[j * 3];
      const dy = positions[i * 3 + 1] - positions[j * 3 + 1];
      const dz = positions[i * 3 + 2] - positions[j * 3 + 2];
      const d2 = dx * dx + dy * dy + dz * dz;
      if (d2 < 6.25 && lineConnections.length < MAX_LINES) {
        // 2.5² = 6.25
        lineConnections.push([i, j]);
      }
    }
  }
  // setDrawRange : 2 sommets par segment. Fixe (topologie figée) ; seules les coords changeront.
  lineGeo.setDrawRange(0, lineConnections.length * 2);

  // =========================================================
  // SHOOTING STARS — météores traversant la sphère
  // =========================================================
  const MAX_SHOOTING_STARS = 3; // simultanés max
  const SHOOTING_TRAIL_LENGTH = 50; // longueur du trail en nombre de positions mémorisées

  interface ShootingStar {
    active: boolean;
    position: { x: number; y: number; z: number };
    velocity: { x: number; y: number; z: number };
    trail: Array<{ x: number; y: number; z: number }>;
    life: number;
    maxLife: number;
    line: Line;
    mat: LineBasicMaterial;
    geo: BufferGeometry;
  }

  // Pool de shooting stars — objets créés une fois, réactivés/désactivés (évite allocs/GC).
  const shootingStars: ShootingStar[] = [];
  for (let i = 0; i < MAX_SHOOTING_STARS; i++) {
    const ssGeo = new THREE.BufferGeometry();
    const ssPos = new Float32Array(SHOOTING_TRAIL_LENGTH * 3);
    ssGeo.setAttribute('position', new THREE.BufferAttribute(ssPos, 3));
    ssGeo.setDrawRange(0, 0); // rien affiché tant qu'inactive

    const ssMat = new THREE.LineBasicMaterial({
      color: 0xe0f0ff, // blanc/bleu clair
      transparent: true,
      opacity: 0.0,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });

    const ssLine = new THREE.Line(ssGeo, ssMat);
    ssLine.visible = false;
    group.add(ssLine); // enfant du group → bénéficie de la rotation globale

    shootingStars.push({
      active: false,
      position: { x: 0, y: 0, z: 0 },
      velocity: { x: 0, y: 0, z: 0 },
      trail: [],
      life: 0,
      maxLife: 0,
      line: ssLine,
      mat: ssMat,
      geo: ssGeo,
    });
  }

  /**
   * Spawn une shooting star sur un slot libre.
   * - Origine : point aléatoire sur la surface de la sphère (rayon 10, hors scène)
   * - Vélocité : vers le centre + déviation aléatoire, normalisée × vitesse cible.
   */
  const spawnShootingStar = () => {
    const slot = shootingStars.find((s) => !s.active);
    if (!slot) return;

    const theta = Math.random() * Math.PI * 2;
    const phi = Math.acos(2 * Math.random() - 1);
    const r = 10;
    const startX = r * Math.sin(phi) * Math.cos(theta);
    const startY = r * Math.sin(phi) * Math.sin(theta);
    const startZ = r * Math.cos(phi);

    // Direction : vers le centre + petite déviation aléatoire (trajectoire diagonale)
    const dirX = -startX + (Math.random() - 0.5) * 2;
    const dirY = -startY + (Math.random() - 0.5) * 2;
    const dirZ = -startZ + (Math.random() - 0.5) * 2;

    const len = Math.sqrt(dirX * dirX + dirY * dirY + dirZ * dirZ);
    const speed = 0.15; // ≈ traversée en ~1.1s à 60fps

    const maxLife = Math.round((r * 2.2) / speed);

    slot.active = true;
    slot.position = { x: startX, y: startY, z: startZ };
    slot.velocity = {
      x: (dirX / len) * speed,
      y: (dirY / len) * speed,
      z: (dirZ / len) * speed,
    };
    slot.trail = [];
    slot.life = maxLife;
    slot.maxLife = maxLife;
    slot.line.visible = true;
    slot.mat.opacity = 0.8;
  };

  /**
   * Timer récursif — planifie un spawn toutes les 4–8 secondes (aléatoire).
   * setTimeout récursif (pas setInterval) pour varier l'intervalle à chaque fois.
   */
  let shootingStarTimer: ReturnType<typeof setTimeout>;
  const scheduleNextShootingStar = () => {
    const delay = Math.random() * 4000 + 4000; // 4000–8000ms
    shootingStarTimer = setTimeout(() => {
      spawnShootingStar();
      scheduleNextShootingStar();
    }, delay);
  };
  // Premier spawn décalé de 2s pour laisser la scène se stabiliser
  shootingStarTimer = setTimeout(() => {
    spawnShootingStar();
    scheduleNextShootingStar();
  }, 2000);

  // =========================================================
  // FRAME — appelée par la RAF loop de la coquille à chaque frame.
  // Incréments fixes par frame (pas dt-based) — comportement visuel identique à l'origine.
  // =========================================================
  const frame = () => {
    // Rotation globale lente — Z plus lent pour un effet subtil
    group.rotation.x += 0.0003;
    group.rotation.y += 0.0005;
    group.rotation.z += 0.0001;

    const pos = pointsGeo.attributes.position.array as Float32Array;

    // --- Drift des points ---
    for (let i = 0; i < POINT_COUNT; i++) {
      const ix = i * 3;

      pos[ix] += velocities[i].x;
      pos[ix + 1] += velocities[i].y;
      pos[ix + 2] += velocities[i].z;

      // Bounce : comparaison au carré pour éviter Math.sqrt. dist² > 64 ⇔ dist > 8 (rayon).
      const dist2 = pos[ix] ** 2 + pos[ix + 1] ** 2 + pos[ix + 2] ** 2;
      if (dist2 > 64) {
        velocities[i].x *= -1;
        velocities[i].y *= -1;
        velocities[i].z *= -1;
      }
    }
    pointsGeo.attributes.position.needsUpdate = true;

    // --- Sync des traits avec le drift des étoiles ---
    // QUOI : recopier les positions COURANTES des points reliés dans le buffer des lignes.
    // POURQUOI : la topologie est figée à l'init, mais les points viennent de drifter. Sans
    //   cette recopie, les traits resteraient collés aux positions initiales → désync visible.
    // COMMENT : pour chaque paire [a,b] figée, lire pos[a]/pos[b] et écrire les 6 floats du
    //   segment k. O(nb_traits) recopies — négligeable, pas le O(n²) init.
    for (let k = 0; k < lineConnections.length; k++) {
      const a = lineConnections[k][0] * 3;
      const b = lineConnections[k][1] * 3;
      linePositions[k * 6] = pos[a];
      linePositions[k * 6 + 1] = pos[a + 1];
      linePositions[k * 6 + 2] = pos[a + 2];
      linePositions[k * 6 + 3] = pos[b];
      linePositions[k * 6 + 4] = pos[b + 1];
      linePositions[k * 6 + 5] = pos[b + 2];
    }
    lineGeo.attributes.position.needsUpdate = true;

    // --- Update shooting stars ---
    for (const ss of shootingStars) {
      if (!ss.active) continue;

      ss.life--;

      ss.position.x += ss.velocity.x;
      ss.position.y += ss.velocity.y;
      ss.position.z += ss.velocity.z;

      // distSS² > 144 ⇔ distSS > 12 (rayon de sortie de la sphère)
      const distSS2 = ss.position.x ** 2 + ss.position.y ** 2 + ss.position.z ** 2;
      const outOfBounds = distSS2 > 144;

      if (ss.life <= 0 || outOfBounds) {
        ss.active = false;
        ss.line.visible = false;
        ss.mat.opacity = 0.0;
        ss.trail = [];
        ss.geo.setDrawRange(0, 0);
        ss.geo.attributes.position.needsUpdate = true;
        continue;
      }

      // Enregistrer la position courante dans le trail (tête en [0])
      ss.trail.unshift({ x: ss.position.x, y: ss.position.y, z: ss.position.z });
      if (ss.trail.length > SHOOTING_TRAIL_LENGTH) ss.trail.pop();

      const ssPosBuf = ss.geo.attributes.position.array as Float32Array;
      for (let t = 0; t < ss.trail.length; t++) {
        ssPosBuf[t * 3] = ss.trail[t].x;
        ssPosBuf[t * 3 + 1] = ss.trail[t].y;
        ssPosBuf[t * 3 + 2] = ss.trail[t].z;
      }
      ss.geo.attributes.position.needsUpdate = true;
      ss.geo.setDrawRange(0, ss.trail.length);

      // Fade-out progressif : pleine opacité 60% de la vie, puis descente à 0
      const lifeRatio = ss.life / ss.maxLife;
      if (lifeRatio > 0.6) {
        ss.mat.opacity = 0.8;
      } else {
        ss.mat.opacity = (lifeRatio / 0.6) * 0.8;
      }
    }

    renderer.render(scene, camera);
  };

  // =========================================================
  // RESIZE — maintenir le ratio caméra/renderer (appelé par la coquille)
  // =========================================================
  const resize = (w: number, h: number) => {
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    renderer.setSize(w, h);
  };

  // =========================================================
  // DISPOSE — libère GPU memory + timers (appelé au switch/unmount par la coquille)
  // =========================================================
  const dispose = () => {
    clearTimeout(shootingStarTimer); // stoppe le spawn timer
    pointsGeo.dispose();
    pointsMat.dispose();
    glowTexture.dispose();
    lineGeo.dispose();
    lineMat.dispose();
    shootingStars.forEach((ss) => {
      ss.geo.dispose();
      ss.mat.dispose();
    });
    renderer.dispose();
    if (renderer.domElement.parentNode === mount) {
      mount.removeChild(renderer.domElement);
    }
  };

  return { frame, resize, dispose };
};

export default initConstellation;
