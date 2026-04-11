'use client';

import { useEffect, useRef, useState } from 'react';
// Import de type uniquement — ne charge PAS Three.js dans le bundle initial.
// Utilisé uniquement pour les annotations TypeScript dans ce fichier.
import type { Line, LineBasicMaterial, BufferGeometry } from 'three';

/**
 * ConstellationBackground — fond 3D interactif avec Three.js.
 *
 * Architecture :
 * - import('three') est dynamique (dans useEffect) : 0 impact sur le SSR/LCP.
 * - Un THREE.Group regroupe tous les objets et tourne lentement sur les 3 axes.
 * - 100 points driftent lentement dans une sphère de rayon 8 unités.
 * - Les paires à moins de 2.5u sont reliées par des LineSegments (buffer pré-alloué).
 * - Connexions calculées UNE SEULE FOIS à l'init — 0 CPU en runtime, 0 clignotement.
 * - Sur mobile (< 768px) : Three.js est complètement skippé (trop coûteux pour un effet décoratif).
 * - Canvas transparent (alpha:true, clearColor alpha=0) — les blobs aurora restent visibles dessous.
 *
 * Optimisations perf :
 * - distance² partout (Math.sqrt évité dans la hot path O(n²))
 * - Connexions statiques calculées à l'init (0 CPU en runtime, 0 clignotement)
 * - Trails des points normaux supprimés (coût élevé, effet quasi invisible)
 * - Skip complet sur mobile
 */
export default function ConstellationBackground() {
  const mountRef = useRef<HTMLDivElement>(null);
  // isLoaded contrôle le fade-in CSS — passe à true après que Three.js est initialisé
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    if (!mountRef.current) return;

    // Fix 5 — Sur mobile : skip complètement Three.js.
    // La constellation est purement décorative — économiser tout le CPU/GPU sur petit écran.
    if (window.innerWidth < 768) return;

    // Ces refs sont capturées dans la closure de cleanup pour éviter de refermer le mauvais scope
    let animationId: number;
    let renderer: any;

    // Import dynamique Three.js — différé de 3s pour laisser le LCP se rendre d'abord
    const startDelay = setTimeout(() => {
    import('three').then((THREE) => {
      const mount = mountRef.current;
      if (!mount) return;

      // =========================================================
      // SCENE SETUP
      // =========================================================
      const scene = new THREE.Scene();
      const camera = new THREE.PerspectiveCamera(
        60,
        window.innerWidth / window.innerHeight,
        0.1,
        100
      );
      camera.position.z = 12;

      // Fix 4 — antialias désactivé sur mobile (inutile ici car on skip mobile via Fix 5,
      // mais conservé pour la cohérence si la limite de screen width change)
      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setSize(window.innerWidth, window.innerHeight);
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      // alpha=0 → transparent, les blobs aurora CSS sont visibles en dessous
      renderer.setClearColor(0x000000, 0);
      mount.appendChild(renderer.domElement);

      // =========================================================
      // POINTS (étoiles) — 100 points distribués en sphère
      // Fix 1 : réduit de 120 → 100 (desktop). Mobile skipé via Fix 5.
      // =========================================================
      const POINT_COUNT = 100;
      const positions = new Float32Array(POINT_COUNT * 3);
      const velocities: Array<{ x: number; y: number; z: number }> = [];

      for (let i = 0; i < POINT_COUNT; i++) {
        // Distribution sphérique uniforme (méthode angles polaires)
        const theta = Math.random() * Math.PI * 2;
        const phi = Math.acos(2 * Math.random() - 1);
        const r = 3 + Math.random() * 5; // rayon entre 3 et 8 unités

        positions[i * 3]     = r * Math.sin(phi) * Math.cos(theta);
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
      // Chromium génère un texture atlas côté GPU — plus efficace qu'un ShaderMaterial custom.
      const glowCanvas = document.createElement('canvas');
      glowCanvas.width = 64;
      glowCanvas.height = 64;
      const glowCtx = glowCanvas.getContext('2d')!;

      // Dégradé radial centré : blanc opaque au centre, transparent aux bords
      const gradient = glowCtx.createRadialGradient(32, 32, 0, 32, 32, 32);
      gradient.addColorStop(0,   'rgba(255, 255, 255, 1.0)');  // centre : blanc pur
      gradient.addColorStop(0.3, 'rgba(147, 197, 253, 0.8)'); // bleu clair (93c5fd)
      gradient.addColorStop(0.7, 'rgba(59, 130, 246, 0.3)');  // bleu primaire, semi-transparent
      gradient.addColorStop(1.0, 'rgba(0, 0, 0, 0)');          // bord : transparent

      glowCtx.fillStyle = gradient;
      glowCtx.fillRect(0, 0, 64, 64);

      const glowTexture = new THREE.CanvasTexture(glowCanvas);

      const pointsMat = new THREE.PointsMaterial({
        color: 0xffffff,          // blanc — la teinte finale vient de la texture
        size: 0.35,               // plus grand pour que le halo glow soit visible
        map: glowTexture,
        transparent: true,
        opacity: 0.9,
        blending: THREE.AdditiveBlending,  // mode lumière — zones denses = plus lumineux
        depthWrite: false,        // évite les artefacts de profondeur avec l'additive blending
        sizeAttenuation: true,
      });

      const pointsMesh = new THREE.Points(pointsGeo, pointsMat);

      // =========================================================
      // GROUP — rotation globale lente sur les 3 axes
      // Tout est enfant du group pour tourner ensemble
      // =========================================================
      const group = new THREE.Group();
      group.add(pointsMesh);
      scene.add(group);

      // =========================================================
      // LIGNES DE CONNEXION — buffer pré-alloué, calculé une seule fois à l'init.
      // MAX_LINES * 6 floats = 2 points (x,y,z) * MAX_LINES segments
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
      // CONNEXIONS STATIQUES — calculées une seule fois à l'init.
      // Les points driftent à 0.002u/frame : les lignes statiques
      // sont visuellement indiscernables des lignes dynamiques,
      // mais sans clignotement ni coût O(n²) en runtime.
      // =========================================================
      {
        let lineIdx = 0;
        for (let i = 0; i < POINT_COUNT; i++) {
          for (let j = i + 1; j < POINT_COUNT; j++) {
            const dx = positions[i * 3]     - positions[j * 3];
            const dy = positions[i * 3 + 1] - positions[j * 3 + 1];
            const dz = positions[i * 3 + 2] - positions[j * 3 + 2];
            const d2 = dx * dx + dy * dy + dz * dz;
            if (d2 < 6.25 && lineIdx < MAX_LINES) {  // 2.5² = 6.25
              linePositions[lineIdx * 6]     = positions[i * 3];
              linePositions[lineIdx * 6 + 1] = positions[i * 3 + 1];
              linePositions[lineIdx * 6 + 2] = positions[i * 3 + 2];
              linePositions[lineIdx * 6 + 3] = positions[j * 3];
              linePositions[lineIdx * 6 + 4] = positions[j * 3 + 1];
              linePositions[lineIdx * 6 + 5] = positions[j * 3 + 2];
              lineIdx++;
            }
          }
        }
        lineGeo.attributes.position.needsUpdate = true;
        lineGeo.setDrawRange(0, lineIdx * 2);
      }

      // Fix 3 — Trails des points normaux supprimés.
      // 100 objets Line × mise à jour buffer/frame = coût élevé pour un effet quasi invisible.
      // Les shooting stars conservent leur trail (visible et justifié).

      // =========================================================
      // SHOOTING STARS — météores traversant la sphère
      // Chaque shooting star est une THREE.Line avec un trail de SHOOTING_TRAIL_LENGTH positions.
      // L'opacity globale du matériau démarre à 0.8 et descend pendant les dernières frames de vie
      // (LineBasicMaterial ne supporte pas les vertex colors nativement — pas de dégradé par vertex).
      // =========================================================
      const MAX_SHOOTING_STARS = 3;   // simultanés max — évite la surcharge visuelle
      const SHOOTING_TRAIL_LENGTH = 50; // longueur du trail en nombre de positions mémorisées

      interface ShootingStar {
        active: boolean;                                           // true = en vol, false = slot libre
        position: { x: number; y: number; z: number };            // tête de la shooting star
        velocity: { x: number; y: number; z: number };            // direction × vitesse (unités/frame)
        trail: Array<{ x: number; y: number; z: number }>;        // historique des positions (tête en [0])
        life: number;                                              // frames restantes avant disparition forcée
        maxLife: number;                                           // vie totale (pour le fade-out progressif)
        line: Line;           // THREE.Line — type importé statiquement depuis 'three'
        mat: LineBasicMaterial;
        geo: BufferGeometry;
      }

      // Allouer le pool de shooting stars — les objets Three.js sont créés une fois,
      // réactivés/désactivés selon le besoin pour éviter les allocs/GC en cours d'animation.
      const shootingStars: ShootingStar[] = [];
      for (let i = 0; i < MAX_SHOOTING_STARS; i++) {
        const ssGeo = new THREE.BufferGeometry();
        // Buffer pré-alloué : SHOOTING_TRAIL_LENGTH positions × 3 floats (x,y,z)
        const ssPos = new Float32Array(SHOOTING_TRAIL_LENGTH * 3);
        ssGeo.setAttribute('position', new THREE.BufferAttribute(ssPos, 3));
        // setDrawRange(0, 0) : rien affiché tant que inactive
        ssGeo.setDrawRange(0, 0);

        const ssMat = new THREE.LineBasicMaterial({
          color: 0xe0f0ff,                      // blanc/bleu clair — s'intègre naturellement
          transparent: true,
          opacity: 0.0,                          // invisible au départ
          blending: THREE.AdditiveBlending,      // cohérent avec les étoiles et trails existants
          depthWrite: false,                     // pas d'artefacts de profondeur avec additive blending
        });

        const ssLine = new THREE.Line(ssGeo, ssMat);
        ssLine.visible = false; // caché jusqu'à activation
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
       * - Origine : point aléatoire sur la surface de la sphère (rayon ~10, légèrement hors scène)
       * - Vélocité : vers le centre + déviation aléatoire, normalisée puis multipliée par la vitesse cible.
       *   Vitesse de 0.15 unité/frame ≈ traversée en ~1–1.5s à 60fps sur un rayon de 10u.
       */
      const spawnShootingStar = () => {
        // Trouver un slot libre dans le pool
        const slot = shootingStars.find((s) => !s.active);
        if (!slot) return; // tous les slots occupés, on attend le prochain timer

        // Position de départ : surface de la sphère de rayon 10
        const theta = Math.random() * Math.PI * 2;
        const phi   = Math.acos(2 * Math.random() - 1);
        const r     = 10;
        const startX = r * Math.sin(phi) * Math.cos(theta);
        const startY = r * Math.sin(phi) * Math.sin(theta);
        const startZ = r * Math.cos(phi);

        // Direction : vers le centre (0,0,0) + petite déviation aléatoire (±1 unité)
        // → trajectoire globalement diagonale, pas strictement radiale
        const dirX = -startX + (Math.random() - 0.5) * 2;
        const dirY = -startY + (Math.random() - 0.5) * 2;
        const dirZ = -startZ + (Math.random() - 0.5) * 2;

        // Normalisation puis mise à l'échelle — 0.15 u/frame ≈ traversée en ~1.1s à 60fps
        const len  = Math.sqrt(dirX * dirX + dirY * dirY + dirZ * dirZ);
        const speed = 0.15;

        // Vie max = distance totale à parcourir (diamètre ~20u) / vitesse + marge
        // → la shooting star disparaît au plus tard après ~160 frames (~2.7s à 60fps)
        const maxLife = Math.round((r * 2.2) / speed);

        // Réinitialiser le slot
        slot.active   = true;
        slot.position = { x: startX, y: startY, z: startZ };
        slot.velocity = {
          x: (dirX / len) * speed,
          y: (dirY / len) * speed,
          z: (dirZ / len) * speed,
        };
        slot.trail    = [];
        slot.life     = maxLife;
        slot.maxLife  = maxLife;
        slot.line.visible = true;
        slot.mat.opacity  = 0.8; // pleine opacité au départ
      };

      /**
       * Timer récursif — planifie un spawn toutes les 4–8 secondes (aléatoire).
       * Utilise setTimeout récursif plutôt que setInterval pour varier l'intervalle à chaque fois.
       */
      let shootingStarTimer: ReturnType<typeof setTimeout>;
      const scheduleNextShootingStar = () => {
        const delay = Math.random() * 4000 + 4000; // 4000–8000ms
        shootingStarTimer = setTimeout(() => {
          spawnShootingStar();
          scheduleNextShootingStar(); // replanifie
        }, delay);
      };
      // Premier spawn décalé de 2s pour laisser la scène se stabiliser
      shootingStarTimer = setTimeout(() => {
        spawnShootingStar();
        scheduleNextShootingStar();
      }, 2000);

      // =========================================================
      // ANIMATION LOOP
      // =========================================================

      const animate = () => {
        animationId = requestAnimationFrame(animate);

        // Rotation globale lente — Z plus lent pour un effet subtil
        group.rotation.x += 0.0003;
        group.rotation.y += 0.0005;
        group.rotation.z += 0.0001;

        const pos = pointsGeo.attributes.position.array as Float32Array;

        // --- Drift des points ---
        for (let i = 0; i < POINT_COUNT; i++) {
          const ix = i * 3;

          // Appliquer le drift
          pos[ix]     += velocities[i].x;
          pos[ix + 1] += velocities[i].y;
          pos[ix + 2] += velocities[i].z;

          // Fix 1 — Bounce : comparaison au carré pour éviter Math.sqrt dans la hot path.
          // dist² > 64 équivaut à dist > 8 (rayon de la sphère de confinement).
          const dist2 = pos[ix] ** 2 + pos[ix + 1] ** 2 + pos[ix + 2] ** 2;
          if (dist2 > 64) {  // 8² = 64
            velocities[i].x *= -1;
            velocities[i].y *= -1;
            velocities[i].z *= -1;
          }
        }
        pointsGeo.attributes.position.needsUpdate = true;

        // --- Update shooting stars ---
        // Les trails des points normaux ont été supprimés (Fix 3).
        // Les shooting stars conservent leur trail — visible et justifié.
        for (const ss of shootingStars) {
          if (!ss.active) continue;

          // Décrémenter la vie — quand life <= 0 ou la star a quitté la sphère → désactiver
          ss.life--;

          // Avancer la position selon la vélocité
          ss.position.x += ss.velocity.x;
          ss.position.y += ss.velocity.y;
          ss.position.z += ss.velocity.z;

          // Fix 1 — distance au carré pour éviter Math.sqrt.
          // distSS² > 144 équivaut à distSS > 12 (rayon de sortie de la sphère).
          const distSS2 = ss.position.x ** 2 + ss.position.y ** 2 + ss.position.z ** 2;
          const outOfBounds = distSS2 > 144;  // 12² = 144

          if (ss.life <= 0 || outOfBounds) {
            // Désactiver proprement — le slot sera réutilisé au prochain spawn
            ss.active = false;
            ss.line.visible   = false;
            ss.mat.opacity    = 0.0;
            ss.trail          = [];
            ss.geo.setDrawRange(0, 0);
            ss.geo.attributes.position.needsUpdate = true;
            continue;
          }

          // Enregistrer la position courante dans le trail (tête en [0])
          ss.trail.unshift({ x: ss.position.x, y: ss.position.y, z: ss.position.z });
          if (ss.trail.length > SHOOTING_TRAIL_LENGTH) ss.trail.pop();

          // Mettre à jour le buffer geometry avec les positions du trail
          const ssPosBuf = ss.geo.attributes.position.array as Float32Array;
          for (let t = 0; t < ss.trail.length; t++) {
            ssPosBuf[t * 3]     = ss.trail[t].x;
            ssPosBuf[t * 3 + 1] = ss.trail[t].y;
            ssPosBuf[t * 3 + 2] = ss.trail[t].z;
          }
          ss.geo.attributes.position.needsUpdate = true;
          // setDrawRange : n'afficher que les positions déjà remplies (trail croissant au début)
          ss.geo.setDrawRange(0, ss.trail.length);

          // Fade-out progressif : opacity pleine pendant les 60% de la vie, puis descend à 0
          // Cela crée un effet de disparition naturelle en fin de trajectoire
          const lifeRatio = ss.life / ss.maxLife; // 1.0 → 0.0
          if (lifeRatio > 0.6) {
            ss.mat.opacity = 0.8;           // pleine opacité en début de trajectoire
          } else {
            // fade de 0.8 → 0.0 sur les 40% finaux
            ss.mat.opacity = (lifeRatio / 0.6) * 0.8;
          }
        }

        renderer.render(scene, camera);
      };

      animate();
      // Déclenche le fade-in CSS (opacity 0 → 1 en 1.5s)
      setIsLoaded(true);

      // =========================================================
      // RESIZE — maintenir le ratio caméra/renderer
      // =========================================================
      const handleResize = () => {
        camera.aspect = window.innerWidth / window.innerHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(window.innerWidth, window.innerHeight);
      };
      window.addEventListener('resize', handleResize);

      // =========================================================
      // CLEANUP — appelé au unmount React
      // Libère GPU memory + retire le canvas du DOM
      // Fix 3 — trailObjects supprimés, pas de dispose à faire pour eux.
      // =========================================================
      return () => {
        window.removeEventListener('resize', handleResize);
        cancelAnimationFrame(animationId);
        renderer.dispose();
        // Disposer chaque géométrie et matériau pour éviter les fuites GPU
        pointsGeo.dispose();
        pointsMat.dispose();
        glowTexture.dispose(); // libère la texture canvas du GPU
        lineGeo.dispose();
        lineMat.dispose();
        // Disposer les géométries et matériaux des shooting stars pour libérer la mémoire GPU
        clearTimeout(shootingStarTimer); // stoppe le spawn timer
        shootingStars.forEach((ss) => {
          ss.geo.dispose();
          ss.mat.dispose();
        });
        if (mount && renderer.domElement.parentNode === mount) {
          mount.removeChild(renderer.domElement);
        }
      };
    });
    }, 3000); // 3s de délai — laisse le LCP et le thread principal se libérer d'abord

    // Cleanup : annule le timer si unmount avant les 3s, ou stoppe le renderer si déjà démarré
    return () => {
      clearTimeout(startDelay);
      if (animationId) cancelAnimationFrame(animationId);
      if (renderer) renderer.dispose();
    };
  }, []);

  return (
    <div
      ref={mountRef}
      // z-index 1 : au-dessus des blobs aurora (z-0) mais sous le contenu (z-10)
      // pointer-events-none : le canvas ne capture aucun event utilisateur
      className="fixed inset-0 z-[1] pointer-events-none"
      style={{ opacity: isLoaded ? 1 : 0, transition: 'opacity 1.5s ease' }}
    />
  );
}
