'use client';

import { useEffect, useRef, useState } from 'react';
// Import de type uniquement — ne charge PAS Three.js dans le bundle initial.
// Utilisé uniquement pour les annotations TypeScript dans ce fichier.
import type { Line, LineBasicMaterial } from 'three';

/**
 * ConstellationBackground — fond 3D interactif avec Three.js.
 *
 * Architecture :
 * - import('three') est dynamique (dans useEffect) : 0 impact sur le SSR/LCP.
 * - Un THREE.Group regroupe tous les objets et tourne lentement sur les 3 axes.
 * - 120 points driftent lentement dans une sphère de rayon 8 unités.
 * - Les paires à moins de 2.5u sont reliées par des LineSegments (buffer pré-alloué).
 * - Chaque point laisse un trail de 8 positions (LineBasicMaterial à opacité variable).
 * - Canvas transparent (alpha:true, clearColor alpha=0) — les blobs aurora restent visibles dessous.
 */
export default function ConstellationBackground() {
  const mountRef = useRef<HTMLDivElement>(null);
  // isLoaded contrôle le fade-in CSS — passe à true après que Three.js est initialisé
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    if (!mountRef.current) return;

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

      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setSize(window.innerWidth, window.innerHeight);
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      // alpha=0 → transparent, les blobs aurora CSS sont visibles en dessous
      renderer.setClearColor(0x000000, 0);
      mount.appendChild(renderer.domElement);

      // =========================================================
      // POINTS (étoiles) — 120 points distribués en sphère
      // =========================================================
      const POINT_COUNT = 120;
      const TRAIL_LENGTH = 8;
      const positions = new Float32Array(POINT_COUNT * 3);
      const velocities: Array<{ x: number; y: number; z: number }> = [];
      const trailHistory: Array<Array<{ x: number; y: number; z: number }>> = [];

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

        // Trail initialement vide
        trailHistory.push([]);
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
      // LIGNES DE CONNEXION — buffer pré-alloué, recalculé chaque frame
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
      // TRAILS — une Line par point, buffer de TRAIL_LENGTH positions
      // =========================================================
      const trailObjects: Line[] = [];

      for (let i = 0; i < POINT_COUNT; i++) {
        const trailGeo = new THREE.BufferGeometry();
        const trailPos = new Float32Array(TRAIL_LENGTH * 3);
        trailGeo.setAttribute('position', new THREE.BufferAttribute(trailPos, 3));

        const trailMat = new THREE.LineBasicMaterial({
          color: 0x60a5fa, // Tailwind blue-400
          transparent: true,
          opacity: 0.0, // commence invisible, monte selon la vitesse
        });

        const trailLine = new THREE.Line(trailGeo, trailMat);
        group.add(trailLine);
        trailObjects.push(trailLine);
      }

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

        // --- Drift des points + mise à jour historique trail ---
        for (let i = 0; i < POINT_COUNT; i++) {
          const ix = i * 3;

          // Historique : unshift = position la plus récente en tête
          trailHistory[i].unshift({ x: pos[ix], y: pos[ix + 1], z: pos[ix + 2] });
          if (trailHistory[i].length > TRAIL_LENGTH) trailHistory[i].pop();

          // Appliquer le drift
          pos[ix]     += velocities[i].x;
          pos[ix + 1] += velocities[i].y;
          pos[ix + 2] += velocities[i].z;

          // Bounce : inverser la vélocité si le point sort de la sphère de rayon 8
          const dist = Math.sqrt(pos[ix] ** 2 + pos[ix + 1] ** 2 + pos[ix + 2] ** 2);
          if (dist > 8) {
            velocities[i].x *= -1;
            velocities[i].y *= -1;
            velocities[i].z *= -1;
          }
        }
        pointsGeo.attributes.position.needsUpdate = true;

        // --- Recalcul des lignes de connexion ---
        // Complexité O(n²) — acceptable pour 120 points (7140 paires max)
        let lineIdx = 0;
        for (let i = 0; i < POINT_COUNT; i++) {
          for (let j = i + 1; j < POINT_COUNT; j++) {
            const dx = pos[i * 3]     - pos[j * 3];
            const dy = pos[i * 3 + 1] - pos[j * 3 + 1];
            const dz = pos[i * 3 + 2] - pos[j * 3 + 2];
            const d  = Math.sqrt(dx * dx + dy * dy + dz * dz);

            if (d < 2.5 && lineIdx < MAX_LINES) {
              linePositions[lineIdx * 6]     = pos[i * 3];
              linePositions[lineIdx * 6 + 1] = pos[i * 3 + 1];
              linePositions[lineIdx * 6 + 2] = pos[i * 3 + 2];
              linePositions[lineIdx * 6 + 3] = pos[j * 3];
              linePositions[lineIdx * 6 + 4] = pos[j * 3 + 1];
              linePositions[lineIdx * 6 + 5] = pos[j * 3 + 2];
              lineIdx++;
            }
          }
        }
        lineGeo.attributes.position.needsUpdate = true;
        // setDrawRange évite de rendre les segments vides du buffer pré-alloué
        lineGeo.setDrawRange(0, lineIdx * 2);

        // --- Update trails ---
        for (let i = 0; i < POINT_COUNT; i++) {
          const trail = trailHistory[i];
          if (trail.length < 2) continue;

          const trailLine = trailObjects[i];
          const trailPos = trailLine.geometry.attributes.position.array as Float32Array;

          for (let t = 0; t < trail.length; t++) {
            trailPos[t * 3]     = trail[t].x;
            trailPos[t * 3 + 1] = trail[t].y;
            trailPos[t * 3 + 2] = trail[t].z;
          }
          trailLine.geometry.attributes.position.needsUpdate = true;
          trailLine.geometry.setDrawRange(0, trail.length);

          // Opacité proportionnelle à la vitesse — trail plus visible sur les points rapides
          const speed = Math.sqrt(
            velocities[i].x ** 2 + velocities[i].y ** 2 + velocities[i].z ** 2
          );
          (trailLine.material as LineBasicMaterial).opacity = Math.min(speed * 80, 0.3);
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
        trailObjects.forEach((t) => {
          t.geometry.dispose();
          (t.material as LineBasicMaterial).dispose();
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
